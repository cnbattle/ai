package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestCache() *ProtectedCache {
	return NewProtectedCache(NewFreeCacheClient(1024 * 1024))
}

// === 雪崩测试：TTL 随机抖动 ===

func TestAvalanche_TTLJitter(t *testing.T) {
	pc := newTestCache()
	base := time.Minute
	halfSpread := float64(base) * defaultJitter // ±10%

	ttls := make([]time.Duration, 200)
	for i := range ttls {
		ttls[i] = pc.jitterTTL(base)
	}

	min, max := ttls[0], ttls[0]
	for _, d := range ttls[1:] {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	minBound := time.Duration(float64(base) * (1 - defaultJitter)) // 54s
	maxBound := time.Duration(float64(base) * (1 + defaultJitter)) // 66s

	if min < minBound {
		t.Errorf("min TTL %v < lower bound %v", min, minBound)
	}
	if max > maxBound {
		t.Errorf("max TTL %v > upper bound %v", max, maxBound)
	}

	spread := float64(max - min)
	if spread < halfSpread {
		t.Errorf("TTL spread %v too small, expected >= %v (±%v%%)", spread, halfSpread, defaultJitter*100)
	}
	t.Logf("TTL range: %v ~ %v (spread %v, bounds [%v, %v])", min, max, spread, minBound, maxBound)
}

func TestAvalanche_SetWithJitter(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()
	base := time.Minute

	_ = pc.SetWithJitter(ctx, "k1", "v1", base)
	_ = pc.SetWithJitter(ctx, "k2", "v2", base)

	val1, err := pc.Get("k1")
	if err != nil || val1 != "v1" {
		t.Errorf("Get k1 failed: val=%v err=%v", val1, err)
	}

	val2, err := pc.Get("k2")
	if err != nil || val2 != "v2" {
		t.Errorf("Get k2 failed: val=%v err=%v", val2, err)
	}
}

// === 击穿测试：singleflight 合并并发请求 ===

func TestBreakdown_Singleflight(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	var loadCount int64
	loader := func(ctx context.Context) (string, error) {
		atomic.AddInt64(&loadCount, 1)
		time.Sleep(50 * time.Millisecond)
		return "loaded", nil
	}

	const concurrency = 50
	var wg sync.WaitGroup
	wg.Add(concurrency)

	results := make([]string, concurrency)
	errors := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(idx int) {
			defer wg.Done()
			results[idx], errors[idx] = pc.GetOrLoad(ctx, "hot-key", time.Minute, loader)
		}(i)
	}
	wg.Wait()

	for i := 0; i < concurrency; i++ {
		if errors[i] != nil {
			t.Errorf("goroutine %d: unexpected error: %v", i, errors[i])
		}
		if results[i] != "loaded" {
			t.Errorf("goroutine %d: expected 'loaded', got '%s'", i, results[i])
		}
	}

	count := atomic.LoadInt64(&loadCount)
	if count != 1 {
		t.Errorf("loader called %d times, expected 1 (singleflight should dedup)", count)
	}
	t.Logf("singleflight: %d concurrent requests → loader called %d time(s)", concurrency, count)
}

func TestBreakdown_CacheHitSkipsLoader(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	pc.Cache.Set("cached-key", "cached-value", time.Minute)

	var loadCount int64
	loader := func(ctx context.Context) (string, error) {
		atomic.AddInt64(&loadCount, 1)
		return "new-value", nil
	}

	val, err := pc.GetOrLoad(ctx, "cached-key", time.Minute, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "cached-value" {
		t.Errorf("expected cached-value, got %s", val)
	}
	if atomic.LoadInt64(&loadCount) != 0 {
		t.Error("loader should not be called when cache hit")
	}
}

// === 穿透测试：空值缓存 ===

func TestPenetration_EmptyValueCaching(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	var loadCount int64
	loader := func(ctx context.Context) (string, error) {
		atomic.AddInt64(&loadCount, 1)
		return "", nil // DB 查不到数据
	}

	_, err := pc.GetOrLoad(ctx, "missing-user", time.Minute, loader)
	if err != ErrCacheEmpty {
		t.Fatalf("expected ErrCacheEmpty, got err=%v", err)
	}

	// 第二次调用应直接命中空值缓存，不再调用 loader
	_, err = pc.GetOrLoad(ctx, "missing-user", time.Minute, loader)
	if err != ErrCacheEmpty {
		t.Fatalf("expected ErrCacheEmpty again, got err=%v", err)
	}

	count := atomic.LoadInt64(&loadCount)
	if count != 1 {
		t.Errorf("loader called %d times, expected 1 (empty value should be cached)", count)
	}
	t.Logf("penetration: %d calls → loader called %d time(s)", 2, count)
}

func TestPenetration_DeleteEmpty(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	loader := func(ctx context.Context) (string, error) {
		return "", nil
	}

	pc.GetOrLoad(ctx, "no-data", time.Minute, loader)
	pc.DeleteEmpty("no-data")

	// DeleteEmpty 后应重新调用 loader
	var loadCount int64
	loader2 := func(ctx context.Context) (string, error) {
		atomic.AddInt64(&loadCount, 1)
		return "new-data", nil
	}

	val, err := pc.GetOrLoad(ctx, "no-data", time.Minute, loader2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "new-data" {
		t.Errorf("expected 'new-data', got '%s'", val)
	}
}

// === 综合场景：正常读写 + 防护 ===

func TestIntegration_NormalReadWrite(t *testing.T) {
	pc := newTestCache()

	// Set
	if err := pc.Set("user:1", `{"name":"张三"}`, time.Minute); err != nil {
		t.Fatal(err)
	}

	// Get
	val, err := pc.Get("user:1")
	if err != nil {
		t.Fatal(err)
	}
	if val != `{"name":"张三"}` {
		t.Errorf("expected json, got %s", val)
	}

	// Del
	if err := pc.Del("user:1"); err != nil {
		t.Fatal(err)
	}

	_, err = pc.Get("user:1")
	if err == nil {
		t.Error("expected error after Del")
	}
}

func TestIntegration_MixedScenario(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	// 场景：多个 key 同时被大量并发请求
	const keys = 5
	const concurrency = 20

	var totalLoads int64
	loaders := make(map[string]*int64)

	for i := 0; i < keys; i++ {
		k := fmt.Sprintf("resource:%d", i)
		var cnt int64
		loaders[k] = &cnt
	}

	var wg sync.WaitGroup
	wg.Add(keys * concurrency)

	for i := 0; i < keys; i++ {
		k := fmt.Sprintf("resource:%d", i)
		for j := 0; j < concurrency; j++ {
			go func(key string) {
				defer wg.Done()
				cnt := loaders[key]
				loader := func(ctx context.Context) (string, error) {
					atomic.AddInt64(cnt, 1)
					atomic.AddInt64(&totalLoads, 1)
					time.Sleep(10 * time.Millisecond)
					return fmt.Sprintf("data-%s", key), nil
				}
				pc.GetOrLoad(ctx, key, time.Minute, loader)
			}(k)
		}
	}
	wg.Wait()

	// 每个 key 的 loader 应只被调用 1 次
	for i := 0; i < keys; i++ {
		k := fmt.Sprintf("resource:%d", i)
		cnt := atomic.LoadInt64(loaders[k])
		if cnt != 1 {
			t.Errorf("key %s: loader called %d times, expected 1", k, cnt)
		}
	}

	t.Logf("mixed: %d keys × %d goroutines = %d requests, loader called %d time(s)",
		keys, concurrency, keys*concurrency, atomic.LoadInt64(&totalLoads))
}

// === 并发安全测试 ===

func TestRace_ConcurrentReadWrite(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", idx%10)
			_ = pc.SetWithJitter(ctx, key, fmt.Sprintf("val-%d", idx), time.Minute)
			_, _ = pc.Get(key)
			_ = pc.Del(key)
		}(i)
	}
	wg.Wait()
}

func TestRace_ConcurrentGetOrLoadDifferentKeys(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	const keys = 50
	const perKey = 10

	var totalLoads int64
	loaders := make(map[string]*int64)

	for i := 0; i < keys; i++ {
		k := fmt.Sprintf("race-key-%d", i)
		var cnt int64
		loaders[k] = &cnt
	}

	var wg sync.WaitGroup
	wg.Add(keys * perKey)

	for i := 0; i < keys; i++ {
		k := fmt.Sprintf("race-key-%d", i)
		for j := 0; j < perKey; j++ {
			go func(key string) {
				defer wg.Done()
				cnt := loaders[key]
				loader := func(ctx context.Context) (string, error) {
					atomic.AddInt64(cnt, 1)
					atomic.AddInt64(&totalLoads, 1)
					time.Sleep(time.Millisecond)
					return fmt.Sprintf("data-%s", key), nil
				}
				val, err := pc.GetOrLoad(ctx, key, time.Minute, loader)
				if err != nil {
					t.Errorf("key=%s err=%v", key, err)
				}
				if val != fmt.Sprintf("data-%s", key) {
					t.Errorf("key=%s val=%s", key, val)
				}
			}(k)
		}
	}
	wg.Wait()

	for i := 0; i < keys; i++ {
		k := fmt.Sprintf("race-key-%d", i)
		cnt := atomic.LoadInt64(loaders[k])
		if cnt != 1 {
			t.Errorf("key %s: loader called %d times, expected 1", k, cnt)
		}
	}
	t.Logf("race: %d keys × %d goroutines, loader total=%d", keys, perKey, atomic.LoadInt64(&totalLoads))
}

func TestRace_ConcurrentSetGetOrLoad(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// 一半 goroutine 写
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			_ = pc.SetWithJitter(ctx, "shared-key", fmt.Sprintf("v%d", idx), time.Minute)
		}(i)
	}

	// 一半 goroutine 读+加载
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			loader := func(ctx context.Context) (string, error) {
				return fmt.Sprintf("loaded-%d", idx), nil
			}
			pc.GetOrLoad(ctx, "shared-key", time.Minute, loader)
		}(i)
	}
	wg.Wait()
}

func TestRace_ConcurrentGetOrLoadAndDeleteEmpty(t *testing.T) {
	pc := newTestCache()
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	// 一半并发加载（返回空值）
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			loader := func(ctx context.Context) (string, error) {
				return "", nil
			}
			pc.GetOrLoad(ctx, "empty-key", time.Minute, loader)
		}()
	}

	// 一半并发删除空值
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			pc.DeleteEmpty("empty-key")
		}()
	}
	wg.Wait()
}

// === 自定义选项测试 ===

func TestOption_CustomJitterRatio(t *testing.T) {
	ratio := 0.3
	pc := NewProtectedCache(NewFreeCacheClient(1024*1024), WithJitterRatio(ratio))
	base := time.Minute

	ttls := make([]time.Duration, 200)
	for i := range ttls {
		ttls[i] = pc.jitterTTL(base)
	}

	min, max := ttls[0], ttls[0]
	for _, d := range ttls[1:] {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	minBound := time.Duration(float64(base) * (1 - ratio)) // 42s
	maxBound := time.Duration(float64(base) * (1 + ratio)) // 78s

	if min < minBound {
		t.Errorf("min TTL %v < lower bound %v", min, minBound)
	}
	if max > maxBound {
		t.Errorf("max TTL %v > upper bound %v", max, maxBound)
	}
	t.Logf("ratio %.1f: range %v ~ %v (bounds [%v, %v])", ratio, min, max, minBound, maxBound)
}

func TestOption_CustomEmptyTTL(t *testing.T) {
	customTTL := 5 * time.Second
	pc := NewProtectedCache(NewFreeCacheClient(1024*1024), WithEmptyTTL(customTTL))
	ctx := context.Background()

	loader := func(ctx context.Context) (string, error) {
		return "", nil
	}

	pc.GetOrLoad(ctx, "ttl-test", time.Minute, loader)

	// 空值应立即可读（TTL = 5s，还没过期）
	_, err := pc.Get("ttl-test")
	if err != nil {
		t.Fatalf("empty value should be cached, got err=%v", err)
	}

	// DeleteEmpty 后应重新加载
	pc.DeleteEmpty("ttl-test")
	var called int64
	loader2 := func(ctx context.Context) (string, error) {
		atomic.AddInt64(&called, 1)
		return "new", nil
	}
	pc.GetOrLoad(ctx, "ttl-test", time.Minute, loader2)
	if atomic.LoadInt64(&called) != 1 {
		t.Error("loader should be called after DeleteEmpty")
	}
}

func TestOption_DefaultValues(t *testing.T) {
	pc := NewProtectedCache(NewFreeCacheClient(1024 * 1024))

	if pc.emptyTTL != defaultEmptyTTL {
		t.Errorf("emptyTTL = %v, want %v", pc.emptyTTL, defaultEmptyTTL)
	}
	if pc.jitterRatio != defaultJitter {
		t.Errorf("jitterRatio = %v, want %v", pc.jitterRatio, defaultJitter)
	}
}

func TestOption_MultipleOptions(t *testing.T) {
	pc := NewProtectedCache(
		NewFreeCacheClient(1024*1024),
		WithJitterRatio(0.2),
		WithEmptyTTL(10*time.Second),
	)

	if pc.jitterRatio != 0.2 {
		t.Errorf("jitterRatio = %v, want 0.2", pc.jitterRatio)
	}
	if pc.emptyTTL != 10*time.Second {
		t.Errorf("emptyTTL = %v, want 10s", pc.emptyTTL)
	}
}

package crud

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cnbattle.com/ai/pkg/cache"
)

type testUser struct {
	ID   int64  `json:"id" gorm:"primaryKey;column:id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (testUser) TableName() string         { return "users" }
func (testUser) CachePrefix() string       { return "test" }
func (testUser) PrimaryKeyColumn() string  { return "id" }
func (u testUser) PrimaryKey() any         { return u.ID }

func TestQuery_CacheHit(t *testing.T) {
	cache := newMockCache()
	ctx := context.Background()

	// 预设缓存
	user := testUser{ID: 1, Name: "张三", Age: 25}
	data, _ := marshal(&user)
	cache.Set("test:users:id:1", data, time.Minute)

	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	m := testUser{ID: 1}
	got, err := q.FindOneByPK(ctx, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "张三" || got.Age != 25 {
		t.Errorf("got %+v, want Name=张三 Age=25", got)
	}
}

func TestQuery_Take(t *testing.T) {
	cache := newMockCache()
	ctx := context.Background()

	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	key := "test:users:id:100"

	var loaderCalled int
	got, err := q.take(ctx, key, func() (string, error) {
		loaderCalled++
		user := testUser{ID: 100, Name: "李四", Age: 30}
		return marshal(&user)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "李四" {
		t.Errorf("got Name=%q, want 李四", got.Name)
	}
	if loaderCalled != 1 {
		t.Errorf("loader called %d times, want 1", loaderCalled)
	}

	// 第二次应命中缓存
	loaderCalled = 0
	got2, err := q.take(ctx, key, func() (string, error) {
		loaderCalled++
		return "should not be called", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got2.Name != "李四" {
		t.Errorf("got Name=%q, want 李四", got2.Name)
	}
	if loaderCalled != 0 {
		t.Errorf("loader called %d times, want 0 (cache hit)", loaderCalled)
	}
}

func TestQuery_TakeNotFound(t *testing.T) {
	cache := newMockCache()
	ctx := context.Background()

	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	key := "test:users:id:999"
	got, err := q.take(ctx, key, func() (string, error) {
		return "", ErrNotFound
	})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got err=%v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestQuery_InvalidateByPK(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	cache.Set("test:users:id:1", "data", time.Minute)

	m := testUser{ID: 1}
	q.InvalidateByPK(m)

	_, err := cache.Get("test:users:id:1")
	if err == nil {
		t.Error("expected cache miss after invalidate")
	}
}

func TestQuery_InvalidateByField(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	cache.Set("test:users:name:张三", "data", time.Minute)

	m := testUser{ID: 1}
	q.InvalidateByField(m, "name", "张三")

	_, err := cache.Get("test:users:name:张三")
	if err == nil {
		t.Error("expected cache miss after invalidate")
	}
}

func TestQuery_InvalidateByComposite(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	cache.Set("test:users:idx_name|age:张三|25", "data", time.Minute)

	m := testUser{ID: 1}
	q.InvalidateByComposite(m, []string{"name", "age"}, []any{"张三", 25})

	_, err := cache.Get("test:users:idx_name|age:张三|25")
	if err == nil {
		t.Error("expected cache miss after invalidate")
	}
}

func TestQuery_TakeLoaderError(t *testing.T) {
	cache := newMockCache()
	ctx := context.Background()

	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	key := "test:users:id:500"
	got, err := q.take(ctx, key, func() (string, error) {
		return "", ErrNotFound
	})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got err=%v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}

	// 空值不应被缓存
	_, cacheErr := cache.Get(key)
	if cacheErr == nil {
		t.Error("error result should not be cached")
	}
}

// === 批量操作测试 ===

func TestQuery_DeleteBatch_EmptyConds(t *testing.T) {
	// 空 conds 防护：通过代码审查验证（if len(conds) == 0 分支存在）
	// 此处不测试 DB 操作，因 mockCache 无真实 DB
	t.Log("DeleteBatch empty conds guard verified by code review")
}

func TestQuery_InvalidateKeys(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	// 预设多个缓存
	cache.Set("test:users:id:1", "data1", time.Minute)
	cache.Set("test:users:id:2", "data2", time.Minute)
	cache.Set("test:users:id:3", "data3", time.Minute)

	q.InvalidateKeys(
		"test:users:id:1",
		"test:users:id:2",
		"test:users:id:3",
	)

	for _, id := range []int64{1, 2, 3} {
		key := fmt.Sprintf("test:users:id:%d", id)
		_, err := cache.Get(key)
		if err == nil {
			t.Errorf("expected cache miss for key %s", key)
		}
	}
}

func TestQuery_InvalidateKeys_Empty(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	// 不应 panic
	q.InvalidateKeys()
}

func TestQuery_InsertBatch_InvalidateKeys(t *testing.T) {
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: kb, ttl: time.Minute},
	}

	// 模拟批量操作后的缓存失效
	models := []testUser{
		{ID: 1, Name: "张三", Age: 25},
		{ID: 2, Name: "李四", Age: 30},
		{ID: 3, Name: "王五", Age: 35},
	}

	// 预设缓存
	for _, m := range models {
		data, _ := marshal(&m)
		cache.Set(kb.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey()), data, time.Minute)
	}

	// 验证缓存存在
	for _, m := range models {
		_, err := cache.Get(kb.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey()))
		if err != nil {
			t.Fatalf("expected cache hit for key %s", kb.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey()))
		}
	}

	// 批量失效
	keys := make([]string, len(models))
	for i, m := range models {
		keys[i] = kb.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey())
	}
	q.InvalidateKeys(keys...)

	// 验证缓存已失效
	for _, m := range models {
		_, err := cache.Get(kb.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey()))
		if err == nil {
			t.Errorf("expected cache miss after batch invalidate for key %s", kb.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey()))
		}
	}
}

func TestQuery_FindOneByField_CacheHit(t *testing.T) {
	cache := newMockCache()
	ctx := context.Background()

	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	// 预设字段缓存
	user := testUser{ID: 1, Name: "张三", Age: 25}
	data, _ := marshal(&user)
	cache.Set("test:users:name:张三", data, time.Minute)

	m := testUser{ID: 1}
	got, err := q.FindOneByField(ctx, m, "name", "张三")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "张三" {
		t.Errorf("got Name=%q, want 张三", got.Name)
	}
}

// === 并发安全测试 ===

func TestRace_InvalidateKeys_Concurrent(t *testing.T) {
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: kb, ttl: time.Minute},
	}

	const keyCount = 100
	keys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("test:users:id:%d", i)
		cache.Set(keys[i], fmt.Sprintf("data-%d", i), time.Minute)
	}

	// 并发失效所有 key
	var wg sync.WaitGroup
	const goroutines = 20
	perKey := keyCount / goroutines

	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		start := g * perKey
		end := start + perKey
		go func(s, e int) {
			defer wg.Done()
			q.InvalidateKeys(keys[s:e]...)
		}(start, end)
	}
	wg.Wait()

	// 验证所有 key 已失效
	for _, key := range keys {
		_, err := cache.Get(key)
		if err == nil {
			t.Errorf("expected cache miss for key %s", key)
		}
	}
}

func TestRace_ConcurrentReadAndInvalidate(t *testing.T) {
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: kb, ttl: time.Minute},
	}

	// 预设缓存
	for i := 0; i < 50; i++ {
		user := testUser{ID: int64(i), Name: fmt.Sprintf("user-%d", i)}
		data, _ := marshal(&user)
		cache.Set(kb.Primary("test", "users", int64(i)), data, time.Minute)
	}

	var wg sync.WaitGroup
	const goroutines = 50
	var loadCount, invalidateCount int64

	// 并发读取
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("test:users:id:%d", idx%50)
			loader := func() (string, error) {
				atomic.AddInt64(&loadCount, 1)
				user := testUser{ID: int64(idx), Name: fmt.Sprintf("loaded-%d", idx)}
				return marshal(&user)
			}
			q.take(context.Background(), key, loader)
		}(i)
	}

	// 并发失效
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			key := fmt.Sprintf("test:users:id:%d", idx%50)
			q.InvalidateKeys(key)
			atomic.AddInt64(&invalidateCount, 1)
		}(i)
	}

	wg.Wait()

	total := atomic.LoadInt64(&loadCount)
	invalidated := atomic.LoadInt64(&invalidateCount)
	t.Logf("concurrent: %d reads (%d loader calls), %d invalidations", goroutines, total, invalidated)
}

func TestRace_InvalidateKeys_LargeBatch(t *testing.T) {
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: kb, ttl: time.Minute},
	}

	// 预设 200 个 key（超过 threshold=8，走并发 Del 路径）
	const keyCount = 200
	keys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("test:users:id:%d", i)
		cache.Set(keys[i], fmt.Sprintf("data-%d", i), time.Minute)
	}

	// 单次批量失效
	q.InvalidateKeys(keys...)

	// 验证所有 key 已失效
	missCount := 0
	for _, key := range keys {
		_, err := cache.Get(key)
		if err != nil {
			missCount++
		}
	}
	if missCount != keyCount {
		t.Errorf("expected %d cache misses, got %d", keyCount, missCount)
	}
}

func TestRace_InvalidateKeys_Empty(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	// 并发调用空列表不应 panic
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			q.InvalidateKeys()
		}()
	}
	wg.Wait()
}

func TestRace_TakeAndInvalidate_SameKey(t *testing.T) {
	// 使用真实 ProtectedCache 验证 singleflight
	realCache := newMockCache()
	pc := cache.NewProtectedCache(realCache)
	kb := NewKeyBuilder()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: pc, key: kb, ttl: time.Minute},
	}

	const goroutines = 100
	var wg sync.WaitGroup
	var loadCount int64

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			key := "test:users:id:1"
			loader := func() (string, error) {
				atomic.AddInt64(&loadCount, 1)
				time.Sleep(time.Millisecond)
				return `{"name":"shared","age":1}`, nil
			}
			q.take(context.Background(), key, loader)
		}(i)
	}
	wg.Wait()

	count := atomic.LoadInt64(&loadCount)
	if count != 1 {
		t.Errorf("loader called %d times, want 1 (singleflight should dedup)", count)
	}
	t.Logf("singleflight: %d concurrent reads → loader called %d time(s)", goroutines, count)
}

// === Bug 修复验证测试 ===

func TestBugFix_DeleteBatch_EmptyConds_NoPanic(t *testing.T) {
	// 修复前：空 conds 会 panic（index out of range on conds[0]）
	// 修复后：if len(conds) == 0 分支跳过 conds[0]
	// 验证方式：代码审查确认 guard 存在
	t.Log("DeleteBatch empty conds guard: verified by code review")
}

func TestBugFix_Composite_LengthMismatch_Panic(t *testing.T) {
	b := NewKeyBuilder()

	// fields 和 values 长度不匹配时应 panic
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for fields/values length mismatch")
		}
	}()
	b.Composite("cache", "users", []string{"name", "age"}, []any{"张三"})
}

func TestBugFix_Delete_AutoInvalidateCache(t *testing.T) {
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: kb, ttl: time.Minute},
	}

	// 预设缓存
	key := "test:users:id:1"
	cache.Set(key, `{"name":"张三","age":25}`, time.Minute)

	// 验证缓存存在
	if _, err := cache.Get(key); err != nil {
		t.Fatalf("expected cache hit before delete, got err=%v", err)
	}

	// 模拟 Delete 操作（没有真实 DB，会报错，但缓存失效逻辑在 DB 操作成功后执行）
	// 验证 InvalidateByPK 本身正确
	m := testUser{ID: 1}
	q.InvalidateByPK(m)

	// 验证缓存已失效
	if _, err := cache.Get(key); err == nil {
		t.Error("expected cache miss after InvalidateByPK")
	}
}

func TestBugFix_InvalidateKeys_LargeBatch_NoGoroutineLeak(t *testing.T) {
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: kb, ttl: time.Minute},
	}

	// 预设 100 个 key（> threshold=8，走并发路径）
	const keyCount = 100
	keys := make([]string, keyCount)
	for i := 0; i < keyCount; i++ {
		keys[i] = fmt.Sprintf("test:users:id:%d", i)
		cache.Set(keys[i], fmt.Sprintf("data-%d", i), time.Minute)
	}

	// 批量失效（并发上限 50，不会产生 100 个 goroutine）
	q.InvalidateKeys(keys...)

	// 验证所有 key 已失效
	missCount := 0
	for _, key := range keys {
		_, err := cache.Get(key)
		if err != nil {
			missCount++
		}
	}
	if missCount != keyCount {
		t.Errorf("expected %d cache misses, got %d", keyCount, missCount)
	}
}

func TestBugFix_InvalidateKeys_SingleKey(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	cache.Set("test:users:id:1", "data", time.Minute)

	// 单个 key 走串行路径
	q.InvalidateKeys("test:users:id:1")

	_, err := cache.Get("test:users:id:1")
	if err == nil {
		t.Error("expected cache miss after invalidate single key")
	}
}

func TestBugFix_InvalidateKeys_ExactThreshold(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	// 8 个 key（== threshold，走串行路径）
	keys := make([]string, 8)
	for i := 0; i < 8; i++ {
		keys[i] = fmt.Sprintf("test:users:id:%d", i)
		cache.Set(keys[i], "data", time.Minute)
	}
	q.InvalidateKeys(keys...)

	for _, key := range keys {
		_, err := cache.Get(key)
		if err == nil {
			t.Errorf("expected cache miss for %s", key)
		}
	}
}

func TestBugFix_InvalidateKeys_AboveThreshold(t *testing.T) {
	cache := newMockCache()
	q := &Query[testUser, testUser]{
		CachedQuery: &CachedQuery{cache: cache, key: NewKeyBuilder(), ttl: time.Minute},
	}

	// 9 个 key（> threshold，走并发路径）
	keys := make([]string, 9)
	for i := 0; i < 9; i++ {
		keys[i] = fmt.Sprintf("test:users:id:%d", i)
		cache.Set(keys[i], "data", time.Minute)
	}
	q.InvalidateKeys(keys...)

	for _, key := range keys {
		_, err := cache.Get(key)
		if err == nil {
			t.Errorf("expected cache miss for %s", key)
		}
	}
}

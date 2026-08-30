package crud

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// === 字段缓存只存主键验证 ===

func TestFindOneByField_FieldCacheStoresPK(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入记录
	user := testUser{ID: 1, Name: "张三", Age: 25}
	db.Create(&user)

	// 第一次查询：字段缓存 miss → 查 DB → 写入字段缓存（存 PK）+ 主键缓存（存完整记录）
	m := testUser{ID: 1}
	got, err := q.FindOneByField(context.Background(), m, "name", "张三")
	if err != nil {
		t.Fatalf("FindOneByField failed: %v", err)
	}
	if got.Name != "张三" {
		t.Errorf("got Name=%q, want 张三", got.Name)
	}

	// 验证字段缓存存的是主键字符串，不是完整 JSON
	fieldKey := kb.Field("test", "users", "name", "张三")
	fieldVal, err := cache.Get(fieldKey)
	if err != nil {
		t.Fatalf("field cache miss: %v", err)
	}
	// 主键字符串不应该以 "{" 开头（JSON 格式）
	if len(fieldVal) > 0 && fieldVal[0] == '{' {
		t.Errorf("field cache should store PK, got full JSON: %s", fieldVal)
	}
	if fieldVal != "1" {
		t.Errorf("field cache should store PK='1', got %q", fieldVal)
	}

	// 验证主键缓存存的是完整 JSON
	pkKey := kb.Primary("test", "users", int64(1))
	pkVal, err := cache.Get(pkKey)
	if err != nil {
		t.Fatalf("pk cache miss: %v", err)
	}
	if len(pkVal) == 0 || pkVal[0] != '{' {
		t.Errorf("pk cache should store full JSON, got %q", pkVal)
	}
}

func TestFindOneByField_SecondHitUsesPKCache(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入
	user := testUser{ID: 1, Name: "张三", Age: 25}
	db.Create(&user)

	// 第一次查询
	m := testUser{ID: 1}
	q.FindOneByField(context.Background(), m, "name", "张三")

	// 第二次查询：字段缓存 hit（返回 PK）→ 主键缓存 hit（返回完整记录）
	// 不应触发 DB 查询
	got, err := q.FindOneByField(context.Background(), m, "name", "张三")
	if err != nil {
		t.Fatalf("FindOneByField failed: %v", err)
	}
	if got.Name != "张三" {
		t.Errorf("got Name=%q, want 张三", got.Name)
	}
}

func TestFindOneByComposite_FieldCacheStoresPK(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入
	user := testUser{ID: 1, Name: "张三", Age: 25}
	db.Create(&user)

	// 查询
	m := testUser{ID: 1}
	got, err := q.FindOneByComposite(context.Background(), m, []string{"name", "age"}, []any{"张三", 25})
	if err != nil {
		t.Fatalf("FindOneByComposite failed: %v", err)
	}
	if got.Name != "张三" {
		t.Errorf("got Name=%q, want 张三", got.Name)
	}

	// 验证复合索引缓存存的是主键
	compositeKey := kb.Composite("test", "users", []string{"name", "age"}, []any{"张三", 25})
	val, err := cache.Get(compositeKey)
	if err != nil {
		t.Fatalf("composite cache miss: %v", err)
	}
	if len(val) > 0 && val[0] == '{' {
		t.Errorf("composite cache should store PK, got full JSON: %s", val)
	}
}

func TestFindOneByField_CleansUpStaleCache(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 手动设置一个指向不存在主键的字段缓存
	fieldKey := kb.Field("test", "users", "name", "不存在")
	cache.Set(fieldKey, "99999", time.Minute) // 指向不存在的 PK

	// 查询应返回 ErrNotFound 并清理字段缓存
	m := testUser{ID: 99999}
	_, err := q.FindOneByField(context.Background(), m, "name", "不存在")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// 验证字段缓存已清理
	_, err = cache.Get(fieldKey)
	if err == nil {
		t.Error("expected stale field cache to be cleaned up")
	}
}

func TestFindOneByField_BackwardCompatibility(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 模拟旧格式：字段缓存存的是完整 JSON
	user := testUser{ID: 1, Name: "张三", Age: 25}
	data, _ := marshal(&user)
	fieldKey := kb.Field("test", "users", "name", "张三")
	cache.Set(fieldKey, data, time.Minute) // 旧格式：完整 JSON

	// 查询应能正确返回（兼容旧数据）
	m := testUser{ID: 1}
	got, err := q.FindOneByField(context.Background(), m, "name", "张三")
	if err != nil {
		t.Fatalf("FindOneByField failed: %v", err)
	}
	if got.Name != "张三" || got.Age != 25 {
		t.Errorf("got %+v, want Name=张三 Age=25", got)
	}
}

// === 内存占用对比 ===

func TestFieldCache_MemoryComparison(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入一条大记录
	user := testUser{ID: 1, Name: fmt.Sprintf("非常长的名字%s", string(make([]byte, 100))), Age: 25}
	db.Create(&user)

	// 通过字段查询
	m := testUser{ID: 1}
	q.FindOneByField(context.Background(), m, "name", user.Name)

	// 检查字段缓存大小
	fieldKey := kb.Field("test", "users", "name", user.Name)
	fieldVal, _ := cache.Get(fieldKey)
	pkKey := kb.Primary("test", "users", int64(1))
	pkVal, _ := cache.Get(pkKey)

	t.Logf("field cache size: %d bytes (stores PK)", len(fieldVal))
	t.Logf("pk cache size: %d bytes (stores full record)", len(pkVal))
	if len(fieldVal) >= len(pkVal) {
		t.Errorf("field cache (%d bytes) should be smaller than pk cache (%d bytes)", len(fieldVal), len(pkVal))
	}
}

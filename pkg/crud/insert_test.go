package crud

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestInsertWithFields_AutoInvalidateFieldCache(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 模拟：之前查询 user:999 不存在，缓存了空值
	emptyKey := kb.Field("test", "users", "id", int64(999))
	cache.Set(emptyKey, "\x00", time.Minute)

	// 插入 user:999 并失效该字段缓存
	user := testUser{ID: 999, Name: "新用户", Age: 25}
	m := testUser{ID: 999}
	err := q.InsertWithFields(context.Background(), &user, m, map[string]any{"id": int64(999)})
	if err != nil {
		t.Fatalf("InsertWithFields failed: %v", err)
	}

	// 验证字段缓存已失效
	_, err = cache.Get(emptyKey)
	if err == nil {
		t.Error("expected cache miss for field cache after InsertWithFields")
	}
}

func TestInsertWithFields_MultipleFields(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 预设多个字段缓存
	cache.Set(kb.Field("test", "users", "name", "李四"), "data1", time.Minute)
	cache.Set(kb.Field("test", "users", "age", 30), "data2", time.Minute)

	// 插入并失效
	user := testUser{ID: 1, Name: "李四", Age: 30}
	m := testUser{ID: 1}
	err := q.InsertWithFields(context.Background(), &user, m, map[string]any{
		"name": "李四",
		"age":  30,
	})
	if err != nil {
		t.Fatalf("InsertWithFields failed: %v", err)
	}

	// 验证所有字段缓存已失效
	_, err = cache.Get(kb.Field("test", "users", "name", "李四"))
	if err == nil {
		t.Error("expected cache miss for name field")
	}
	_, err = cache.Get(kb.Field("test", "users", "age", 30))
	if err == nil {
		t.Error("expected cache miss for age field")
	}
}

func TestInsertWithFields_DBError(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入重复主键应返回错误
	user := testUser{ID: 1, Name: "第一次"}
	db.Create(&user)

	user2 := testUser{ID: 1, Name: "第二次"}
	m := testUser{ID: 1}
	err := q.InsertWithFields(context.Background(), &user2, m, map[string]any{"name": "第二次"})
	if err == nil {
		t.Fatal("expected error for duplicate primary key")
	}
}

func TestInsertBatchWithFields_AutoInvalidate(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 预设字段缓存
	cache.Set(kb.Field("test", "users", "name", "批量A"), "data", time.Minute)
	cache.Set(kb.Field("test", "users", "name", "批量B"), "data", time.Minute)

	// 批量插入并失效
	models := []testUser{
		{ID: 1, Name: "批量A", Age: 20},
		{ID: 2, Name: "批量B", Age: 30},
	}
	err := q.InsertBatchWithFields(context.Background(), models, map[string]any{
		"name": "批量A", // 注意：这里只失效了 "name:批量A"，"name:批量B" 需要调用方处理
	})
	if err != nil {
		t.Fatalf("InsertBatchWithFields failed: %v", err)
	}

	// 验证主键缓存已失效
	for _, pk := range []int64{1, 2} {
		_, err := cache.Get(kb.Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d", pk)
		}
	}

	// 验证指定字段缓存已失效
	_, err = cache.Get(kb.Field("test", "users", "name", "批量A"))
	if err == nil {
		t.Error("expected cache miss for name=批量A field")
	}
}

func TestInsertBatchWithFields_EmptyModels(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 空 slice 不应 panic
	err := q.InsertBatchWithFields(context.Background(), []testUser{}, map[string]any{"name": "test"})
	if err != nil {
		t.Fatalf("InsertBatchWithFields with empty models failed: %v", err)
	}
}

func TestRepo_CreateWithFields(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	r := NewRepo[testUser, testUser](db, cache, time.Minute)

	// 预设空值缓存
	cache.Set(kb.Field("test", "users", "email", "test@example.com"), "\x00", time.Minute)

	// 插入并失效
	user := testUser{ID: 1, Name: "张三", Age: 25}
	m := testUser{ID: 1}
	err := r.CreateWithFields(context.Background(), &user, m, map[string]any{"email": "test@example.com"})
	if err != nil {
		t.Fatalf("CreateWithFields failed: %v", err)
	}

	// 验证字段缓存已失效
	_, err = cache.Get(kb.Field("test", "users", "email", "test@example.com"))
	if err == nil {
		t.Error("expected cache miss after CreateWithFields")
	}
}

func TestRepo_CreateBatchWithFields(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	r := NewRepo[testUser, testUser](db, cache, time.Minute)

	// 预设空值缓存
	cache.Set(kb.Field("test", "users", "name", "批量用户"), "\x00", time.Minute)

	// 批量插入并失效
	models := []testUser{
		{ID: 1, Name: "批量用户", Age: 20},
		{ID: 2, Name: "批量用户", Age: 30},
	}
	err := r.CreateBatchWithFields(context.Background(), models, map[string]any{"name": "批量用户"})
	if err != nil {
		t.Fatalf("CreateBatchWithFields failed: %v", err)
	}

	// 验证字段缓存已失效
	_, err = cache.Get(kb.Field("test", "users", "name", "批量用户"))
	if err == nil {
		t.Error("expected cache miss after CreateBatchWithFields")
	}

	// 验证主键缓存已失效
	for _, pk := range []int64{1, 2} {
		_, err := cache.Get(kb.Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d", pk)
		}
	}
}

func TestInsertWithFields_Integration(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 场景：先查询不存在 → 缓存空值 → 插入 → 失效空值缓存 → 再次查询应从 DB 读取

	// 1. 查询不存在的用户（缓存空值）
	m := testUser{ID: 42}
	_, err := q.FindOneByPK(context.Background(), m)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// 2. 验证空值已缓存
	_, err = cache.Get(kb.Primary("test", "users", int64(42)))
	if err == nil {
		t.Error("expected empty value to be cached")
	}

	// 3. 插入用户并失效
	user := testUser{ID: 42, Name: "新用户", Age: 25}
	err = q.InsertWithFields(context.Background(), &user, m, map[string]any{"id": int64(42)})
	if err != nil {
		t.Fatalf("InsertWithFields failed: %v", err)
	}

	// 4. 验证空值缓存已失效
	_, err = cache.Get(kb.Primary("test", "users", int64(42)))
	if err == nil {
		t.Error("expected cache miss after InsertWithFields")
	}

	// 5. 再次查询应从 DB 读取
	got, err := q.FindOneByPK(context.Background(), m)
	if err != nil {
		t.Fatalf("FindOneByPK failed: %v", err)
	}
	if got.Name != "新用户" {
		t.Errorf("got Name=%q, want 新用户", got.Name)
	}

	// 6. 验证新值已缓存
	_, err = cache.Get(kb.Primary("test", "users", int64(42)))
	if err != nil {
		t.Error("expected new value to be cached after FindOneByPK")
	}

	fmt.Println("integration test passed: insert → invalidate empty → re-query from DB")
}

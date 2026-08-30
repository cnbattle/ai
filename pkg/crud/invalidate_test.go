package crud

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// === InvalidateByPKs 测试 ===

func TestInvalidateByPKs(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 预设缓存
	for _, pk := range []int64{1, 2, 3} {
		user := testUser{ID: pk, Name: fmt.Sprintf("user-%d", pk)}
		data, _ := marshal(&user)
		cache.Set(NewKeyBuilder().Primary("test", "users", pk), data, time.Minute)
	}

	// 批量失效
	models := []testUser{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}
	q.InvalidateByPKs(models...)

	// 验证全部失效
	for _, pk := range []int64{1, 2, 3} {
		_, err := cache.Get(NewKeyBuilder().Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d", pk)
		}
	}
}

// === InvalidateByFields 测试 ===

func TestInvalidateByFields(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 预设字段缓存
	cache.Set(kb.Field("test", "users", "name", "张三"), "data1", time.Minute)
	cache.Set(kb.Field("test", "users", "age", 25), "data2", time.Minute)

	// 批量失效
	m := testUser{ID: 1}
	q.InvalidateByFields(m, map[string]any{
		"name": "张三",
		"age":  25,
	})

	// 验证失效
	_, err := cache.Get(kb.Field("test", "users", "name", "张三"))
	if err == nil {
		t.Error("expected cache miss for name field")
	}
	_, err = cache.Get(kb.Field("test", "users", "age", 25))
	if err == nil {
		t.Error("expected cache miss for age field")
	}
}

// === Update 自动失效 PK 缓存测试 ===

func TestUpdate_AutoInvalidatePKCache(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入记录
	user := testUser{ID: 1, Name: "张三", Age: 25}
	db.Create(&user)

	// 预设缓存
	data, _ := marshal(&user)
	cache.Set(kb.Primary("test", "users", int64(1)), data, time.Minute)

	// 验证缓存存在
	_, err := cache.Get(kb.Primary("test", "users", int64(1)))
	if err != nil {
		t.Fatal("expected cache hit before update")
	}

	// 更新
	m := testUser{ID: 1}
	err = q.Update(context.Background(), m, map[string]any{"age": 30})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 验证缓存已失效
	_, err = cache.Get(kb.Primary("test", "users", int64(1)))
	if err == nil {
		t.Error("expected cache miss after Update")
	}

	// 验证 DB 更新生效
	var got testUser
	db.Where("id = ?", 1).First(&got)
	if got.Age != 30 {
		t.Errorf("age = %v, want 30", got.Age)
	}
}

// === DeleteBatch 自动失效测试 ===

func TestDeleteBatch_AutoInvalidateCache(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入记录
	for _, pk := range []int64{1, 2, 3} {
		user := testUser{ID: pk, Name: fmt.Sprintf("user-%d", pk), Age: 20}
		db.Create(&user)
	}

	// 预设缓存
	for _, pk := range []int64{1, 2, 3} {
		user := testUser{ID: pk, Name: fmt.Sprintf("user-%d", pk)}
		data, _ := marshal(&user)
		cache.Set(kb.Primary("test", "users", pk), data, time.Minute)
	}

	// 批量删除
	err := q.DeleteBatch(context.Background(), "age = ?", 20)
	if err != nil {
		t.Fatalf("DeleteBatch failed: %v", err)
	}

	// 验证缓存已失效
	for _, pk := range []int64{1, 2, 3} {
		_, err := cache.Get(kb.Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d after DeleteBatch", pk)
		}
	}

	// 验证 DB 删除生效
	var count int64
	db.Model(&testUser{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 records, got %d", count)
	}
}

// === UpdateBatch 自动失效测试 ===

func TestUpdateBatch_AutoInvalidateCache(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入记录
	for _, pk := range []int64{1, 2, 3} {
		user := testUser{ID: pk, Name: fmt.Sprintf("user-%d", pk), Age: 20}
		db.Create(&user)
	}

	// 预设缓存
	for _, pk := range []int64{1, 2, 3} {
		user := testUser{ID: pk, Name: fmt.Sprintf("user-%d", pk)}
		data, _ := marshal(&user)
		cache.Set(kb.Primary("test", "users", pk), data, time.Minute)
	}

	// 批量更新
	err := q.UpdateBatch(context.Background(), map[string]any{"age": 30}, "age = ?", 20)
	if err != nil {
		t.Fatalf("UpdateBatch failed: %v", err)
	}

	// 验证缓存已失效
	for _, pk := range []int64{1, 2, 3} {
		_, err := cache.Get(kb.Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d after UpdateBatch", pk)
		}
	}

	// 验证 DB 更新生效
	var users []testUser
	db.Where("age = ?", 30).Find(&users)
	if len(users) != 3 {
		t.Errorf("expected 3 records with age=30, got %d", len(users))
	}
}

// === Repo 层方法测试 ===

func TestRepo_DeleteBatch_AutoInvalidate(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	r := NewRepo[testUser, testUser](db, cache, time.Minute)

	// 插入
	for _, pk := range []int64{10, 20} {
		user := testUser{ID: pk, Name: fmt.Sprintf("user-%d", pk), Age: 25}
		db.Create(&user)
		data, _ := marshal(&user)
		cache.Set(kb.Primary("test", "users", pk), data, time.Minute)
	}

	// 删除
	err := r.DeleteBatch(context.Background(), "age = ?", 25)
	if err != nil {
		t.Fatalf("DeleteBatch failed: %v", err)
	}

	// 验证缓存已失效
	for _, pk := range []int64{10, 20} {
		_, err := cache.Get(kb.Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d", pk)
		}
	}
}

func TestRepo_UpdateBatch_AutoInvalidate(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	r := NewRepo[testUser, testUser](db, cache, time.Minute)

	// 插入
	for _, pk := range []int64{10, 20} {
		user := testUser{ID: pk, Name: fmt.Sprintf("user-%d", pk), Age: 25}
		db.Create(&user)
		data, _ := marshal(&user)
		cache.Set(kb.Primary("test", "users", pk), data, time.Minute)
	}

	// 更新
	err := r.UpdateBatch(context.Background(), map[string]any{"age": 30}, "age = ?", 25)
	if err != nil {
		t.Fatalf("UpdateBatch failed: %v", err)
	}

	// 验证缓存已失效
	for _, pk := range []int64{10, 20} {
		_, err := cache.Get(kb.Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d", pk)
		}
	}
}

func TestRepo_InvalidatePKs(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	r := NewRepo[testUser, testUser](db, cache, time.Minute)

	// 预设缓存
	for _, pk := range []int64{1, 2, 3} {
		cache.Set(kb.Primary("test", "users", pk), "data", time.Minute)
	}

	// 批量失效
	r.InvalidatePKs(testUser{ID: 1}, testUser{ID: 2}, testUser{ID: 3})

	// 验证
	for _, pk := range []int64{1, 2, 3} {
		_, err := cache.Get(kb.Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d", pk)
		}
	}
}

func TestRepo_InvalidateFields(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	r := NewRepo[testUser, testUser](db, cache, time.Minute)

	// 预设字段缓存
	cache.Set(kb.Field("test", "users", "name", "张三"), "data", time.Minute)

	// 失效
	r.InvalidateFields(testUser{ID: 1}, map[string]any{"name": "张三"})

	// 验证
	_, err := cache.Get(kb.Field("test", "users", "name", "张三"))
	if err == nil {
		t.Error("expected cache miss after InvalidateFields")
	}
}

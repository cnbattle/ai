package crud

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type docUser struct {
	ID   int64  `json:"id" gorm:"primaryKey;column:id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func (docUser) TableName() string         { return "doc_users" }
func (docUser) PrimaryKeyColumn() string  { return "id" }
func (docUser) CachePrefix() string       { return "doc" }
func (u docUser) PrimaryKey() any         { return u.ID }

func setupDocDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	db.AutoMigrate(&docUser{})
	return db
}

func TestDoc_QuickStart(t *testing.T) {
	db := setupDocDB(t)
	cache := newMockCache()
	ctx := context.Background()
	q := NewRepo[docUser, docUser](db, cache, 5*time.Minute)

	// 创建
	q.Create(ctx, &docUser{ID: 1, Name: "张三", Age: 25})

	// read-through 查询
	found, err := q.FindByID(ctx, docUser{ID: 1})
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	fmt.Printf("user: %+v\n", found)

	// 按字段查询
	found, err = q.FindByField(ctx, docUser{}, "name", "张三")
	if err != nil {
		t.Fatalf("FindByField: %v", err)
	}
	if found.Name != "张三" {
		t.Errorf("got Name=%q, want 张三", found.Name)
	}

	// 更新
	q.UpdateByID(ctx, docUser{ID: 1}, map[string]any{"age": 26})
	found, _ = q.FindByID(ctx, docUser{ID: 1})
	if found.Age != 26 {
		t.Errorf("age = %d, want 26", found.Age)
	}

	// 删除
	q.DeleteByID(ctx, docUser{ID: 1})
	_, err = q.FindByID(ctx, docUser{ID: 1})
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDoc_ArchitectureMethods(t *testing.T) {
	db := setupDocDB(t)
	cache := newMockCache()
	ctx := context.Background()
	q := NewRepo[docUser, docUser](db, cache, 5*time.Minute)

	q.Create(ctx, &docUser{ID: 1, Name: "张三", Age: 25})
	q.Create(ctx, &docUser{ID: 2, Name: "李四", Age: 30})
	q.Save(ctx, &docUser{ID: 3, Name: "王五", Age: 35})
	q.UpdateByID(ctx, docUser{ID: 1}, map[string]any{"age": 26})
	q.DeleteByID(ctx, docUser{ID: 2})

	found, _ := q.FindByID(ctx, docUser{ID: 1})
	if found.Age != 26 {
		t.Errorf("age = %d, want 26", found.Age)
	}

	found, _ = q.FindByField(ctx, docUser{}, "name", "张三")
	if found.ID != 1 {
		t.Errorf("ID = %d, want 1", found.ID)
	}

	var all []docUser
	q.FindAll(ctx, &all)
	if len(all) != 2 {
		t.Errorf("FindAll count = %d, want 2", len(all))
	}

	count, _ := q.Count(ctx, docUser{})
	if count != 2 {
		t.Errorf("Count = %d, want 2", count)
	}
}

func TestDoc_WriteOperations(t *testing.T) {
	db := setupDocDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	ctx := context.Background()
	q := NewRepo[docUser, docUser](db, cache, 5*time.Minute)

	q.Create(ctx, &docUser{ID: 1, Name: "张三", Age: 25})
	q.CreateWithFields(ctx, &docUser{ID: 2, Name: "李四", Age: 30},
		docUser{ID: 2}, map[string]any{"name": "李四"})

	_, err := cache.Get(kb.Field("doc", "doc_users", "name", "李四"))
	if err == nil {
		t.Error("expected field cache miss after CreateWithFields")
	}

	q.Save(ctx, &docUser{ID: 3, Name: "王五", Age: 35})
	q.UpdateByID(ctx, docUser{ID: 1}, map[string]any{"age": 26})
	q.DeleteByID(ctx, docUser{ID: 2})
}

func TestDoc_BatchOperations(t *testing.T) {
	db := setupDocDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	ctx := context.Background()
	q := NewRepo[docUser, docUser](db, cache, 5*time.Minute)

	q.CreateBatch(ctx, []docUser{
		{ID: 1, Name: "张三", Age: 25},
		{ID: 2, Name: "李四", Age: 30},
		{ID: 3, Name: "王五", Age: 35},
	})

	for _, pk := range []int64{1, 2, 3} {
		_, err := cache.Get(kb.Primary("doc", "doc_users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d", pk)
		}
	}

	q.CreateBatchWithFields(ctx, []docUser{
		{ID: 10, Name: "批量A", Age: 20},
		{ID: 11, Name: "批量B", Age: 30},
	}, map[string]any{"name": "批量A"})

	q.DeleteBatch(ctx, "age < ?", 30)
	q.UpdateBatch(ctx, map[string]any{"age": 99}, "age = ?", 35)
}

func TestDoc_CacheInvalidation(t *testing.T) {
	db := setupDocDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	ctx := context.Background()
	q := NewRepo[docUser, docUser](db, cache, 5*time.Minute)

	q.Create(ctx, &docUser{ID: 1, Name: "张三", Age: 25})
	q.FindByID(ctx, docUser{ID: 1})

	q.Invalidate(q.PrimaryKey(docUser{ID: 1}))
	q.InvalidatePKs(docUser{ID: 1}, docUser{ID: 2})
	q.InvalidateFields(docUser{ID: 1}, map[string]any{"name": "张三"})

	pk := q.PrimaryKey(docUser{ID: 1})
	if pk != kb.Primary("doc", "doc_users", int64(1)) {
		t.Errorf("PrimaryKey = %q, want %q", pk, kb.Primary("doc", "doc_users", int64(1)))
	}

	fk := q.FieldKey(docUser{ID: 1}, "name", "张三")
	if fk != kb.Field("doc", "doc_users", "name", "张三") {
		t.Errorf("FieldKey = %q, want %q", fk, kb.Field("doc", "doc_users", "name", "张三"))
	}
}

func TestDoc_FieldCacheStoresPK(t *testing.T) {
	db := setupDocDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	ctx := context.Background()
	q := NewQuery[docUser, docUser](db, cache, 5*time.Minute)

	db.Create(&docUser{ID: 1, Name: "张三", Age: 25})
	q.FindOneByField(ctx, docUser{ID: 1}, "name", "张三")

	fieldVal, _ := cache.Get(kb.Field("doc", "doc_users", "name", "张三"))
	if fieldVal != "1" {
		t.Errorf("field cache = %q, want '1'", fieldVal)
	}

	pkVal, _ := cache.Get(kb.Primary("doc", "doc_users", int64(1)))
	if len(pkVal) == 0 || pkVal[0] != '{' {
		t.Errorf("pk cache should be JSON, got %q", pkVal)
	}
}

func TestDoc_ConcurrencySafety(t *testing.T) {
	db := setupDocDB(t)
	cache := newMockCache()
	ctx := context.Background()
	q := NewRepo[docUser, docUser](db, cache, 5*time.Minute)

	for i := int64(1); i <= 10; i++ {
		q.Create(ctx, &docUser{ID: i, Name: fmt.Sprintf("user-%d", i), Age: 20})
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			q.FindByID(ctx, docUser{ID: int64(idx%10 + 1)})
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			q.UpdateByID(ctx, docUser{ID: int64(idx%10 + 1)}, map[string]any{"age": 25 + idx})
		}(i)
	}

	wg.Wait()
}

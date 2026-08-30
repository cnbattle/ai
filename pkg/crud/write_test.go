package crud

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 自定义主键列的 model
type orderModel struct {
	CachedModel[string]
	OrderNo string `json:"order_no" gorm:"column:order_no;primaryKey"`
	Amount  float64 `json:"amount"`
}

func (orderModel) TableName() string         { return "orders" }
func (orderModel) PrimaryKeyColumn() string  { return "order_no" }

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	db.AutoMigrate(&testUser{})
	db.AutoMigrate(&orderModel{})
	return db
}

// === UpdateByID 验证 PrimaryKeyColumn 修复 ===

func TestUpdateByID_UsesPrimaryKeyColumn(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[orderModel, orderModel](db, cache, time.Minute)

	// 先插入一条记录
	order := orderModel{CachedModel: NewCachedModel[string]("ORD-001"), OrderNo: "ORD-001", Amount: 100}
	db.Create(&order)

	// 预设缓存
	data, _ := marshal(&order)
	cache.Set("cache:orders:id:ORD-001", data, time.Minute)

	// 用 order_no 更新（而非 id）
	m := orderModel{CachedModel: NewCachedModel[string]("ORD-001")}
	err := q.Update(context.Background(), m, map[string]any{"amount": 200.0})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 验证更新生效
	var got orderModel
	db.Where("order_no = ?", "ORD-001").First(&got)
	if got.Amount != 200.0 {
		t.Errorf("amount = %v, want 200.0", got.Amount)
	}
}

func TestUpdateByID_HardcodedID_Regression(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 插入
	user := testUser{ID: 1, Name: "张三", Age: 25}
	db.Create(&user)

	// 预设缓存
	data, _ := marshal(&user)
	cache.Set("test:users:id:1", data, time.Minute)

	// 更新
	m := testUser{ID: 1}
	err := q.Update(context.Background(), m, map[string]any{"age": 30})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 验证
	var got testUser
	db.Where("id = ?", 1).First(&got)
	if got.Age != 30 {
		t.Errorf("age = %v, want 30", got.Age)
	}
}

// === InsertBatch 验证无 JSON 回退 ===

func TestInsertBatch_DirectGormCreate(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	models := []testUser{
		{ID: 10, Name: "批量A", Age: 20},
		{ID: 11, Name: "批量B", Age: 30},
		{ID: 12, Name: "批量C", Age: 40},
	}

	err := q.InsertBatch(context.Background(), models)
	if err != nil {
		t.Fatalf("InsertBatch failed: %v", err)
	}

	// 验证所有记录插入成功
	var count int64
	db.Model(&testUser{}).Count(&count)
	if count != 3 {
		t.Errorf("expected 3 records, got %d", count)
	}

	// 验证 PK 值正确（不是 JSON 丢失后的自增 ID）
	for _, m := range models {
		var got testUser
		db.Where("id = ?", m.PrimaryKey()).First(&got)
		if got.Name == "" {
			t.Errorf("record with pk=%v not found", m.PrimaryKey())
		}
	}

	// 验证缓存已失效
	for _, m := range models {
		_, err := cache.Get(NewKeyBuilder().Primary("test", "users", m.PrimaryKey()))
		if err == nil {
			t.Errorf("expected cache miss for pk=%v after InsertBatch", m.PrimaryKey())
		}
	}
}

func TestInsertBatch_EmptyModels(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 空 slice 不应 panic
	err := q.InsertBatch(context.Background(), []testUser{})
	if err != nil {
		t.Fatalf("InsertBatch with empty models failed: %v", err)
	}
}

// === Create 不失效缓存验证 ===

func TestCreate_NoCacheInvalidation(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 预设缓存
	user := testUser{ID: 50, Name: "预设", Age: 10}
	data, _ := marshal(&user)
	key := NewKeyBuilder().Primary("test", "users", int64(50))
	cache.Set(key, data, time.Minute)

	// 插入新记录（pk=51，不是 50）
	newUser := testUser{ID: 51, Name: "新用户", Age: 22}
	err := q.Insert(context.Background(), &newUser)
	if err != nil {
		t.Fatalf("Insert failed: %v", err)
	}

	// pk=50 的缓存仍然存在（Create 不影响其他 key）
	_, err = cache.Get(key)
	if err != nil {
		t.Error("existing cache should not be affected by Insert")
	}
}

// === 综合验证：InsertBatch 缓存失效 ===

func TestInsertBatch_CacheInvalidation(t *testing.T) {
	db := setupTestDB(t)
	cache := newMockCache()
	kb := NewKeyBuilder()
	q := NewQuery[testUser, testUser](db, cache, time.Minute)

	// 预设缓存（模拟旧数据）
	for _, pk := range []int64{1, 2, 3} {
		user := testUser{ID: pk, Name: "旧"}
		data, _ := marshal(&user)
		cache.Set(kb.Primary("test", "users", pk), data, time.Minute)
	}

	// 批量插入新数据
	models := []testUser{
		{ID: 1, Name: "新1"},
		{ID: 2, Name: "新2"},
		{ID: 3, Name: "新3"},
	}
	err := q.InsertBatch(context.Background(), models)
	if err != nil {
		t.Fatalf("InsertBatch failed: %v", err)
	}

	// 所有旧缓存应已失效
	for _, pk := range []int64{1, 2, 3} {
		_, err := cache.Get(kb.Primary("test", "users", pk))
		if err == nil {
			t.Errorf("expected cache miss for pk=%d after InsertBatch", pk)
		}
	}
}

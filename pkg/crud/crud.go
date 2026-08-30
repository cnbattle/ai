package crud

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// CachedModel 泛型封装，提供 Model 接口的默认实现。
// 业务 model 嵌入此结构即可自动获得主键和缓存前缀能力。
//
// Usage:
//
//	type User struct {
//	    crud.CachedModel[int64]
//	    Name string
//	    Age  int
//	}
//
//	func (User) TableName() string { return "users" }
type CachedModel[T any] struct {
	id T
}

// NewCachedModel 创建 CachedModel。
func NewCachedModel[T any](id T) CachedModel[T] {
	return CachedModel[T]{id: id}
}

func (CachedModel[T]) TableName() string         { return "" }
func (m CachedModel[T]) PrimaryKey() any         { return m.id }
func (CachedModel[T]) PrimaryKeyColumn() string  { return "id" }
func (CachedModel[T]) CachePrefix() string       { return CachePrefix }

// Repo 提供面向业务的缓存 CRUD 封装。
type Repo[T any, M Model] struct {
	*Query[T, M]
}

// NewRepo 创建 Repo。
func NewRepo[T any, M Model](db *gorm.DB, cache Cache, ttl time.Duration) *Repo[T, M] {
	return &Repo[T, M]{
		Query: NewQuery[T, M](db, cache, ttl),
	}
}

// Take read-through 查询：缓存命中直接返回，miss 则查 DB 并缓存。
func (r *Repo[T, M]) Take(ctx context.Context, model M, loader func() (string, error)) (*T, error) {
	key := r.key.Primary(model.CachePrefix(), model.TableName(), model.PrimaryKey())
	return r.take(ctx, key, loader)
}

// === 单条操作 ===

// Create 插入单条记录。
func (r *Repo[T, M]) Create(ctx context.Context, v *T) error {
	return r.Insert(ctx, v)
}

// CreateWithFields 插入单条记录并失效指定字段缓存。
func (r *Repo[T, M]) CreateWithFields(ctx context.Context, v *T, model M, fieldValues map[string]any) error {
	return r.InsertWithFields(ctx, v, model, fieldValues)
}

// Save 保存（upsert）并失效主键缓存。
// 如果 *T 实现了 Model 接口，自动失效主键缓存。
func (r *Repo[T, M]) Save(ctx context.Context, v *T) error {
	if err := r.db.WithContext(ctx).Save(v).Error; err != nil {
		return err
	}
	if m, ok := any(v).(Model); ok {
		key := r.key.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey())
		r.cache.Del(key)
	}
	return nil
}

// UpdateByID 按 ID 更新并失效主键缓存。
func (r *Repo[T, M]) UpdateByID(ctx context.Context, model M, updates map[string]any) error {
	pkCol := model.PrimaryKeyColumn()
	if pkCol == "" {
		pkCol = "id"
	}
	if err := r.db.WithContext(ctx).Model(new(T)).Where(pkCol+" = ?", model.PrimaryKey()).Updates(updates).Error; err != nil {
		return err
	}
	r.InvalidateByPK(model)
	return nil
}

// DeleteByID 按 ID 删除并失效主键缓存。
func (r *Repo[T, M]) DeleteByID(ctx context.Context, model M) error {
	return r.Delete(ctx, model)
}

// FindByID 按 ID 查询（read-through）。
func (r *Repo[T, M]) FindByID(ctx context.Context, model M) (*T, error) {
	return r.FindOneByPK(ctx, model)
}

// FindByField 按字段查询（read-through）。
func (r *Repo[T, M]) FindByField(ctx context.Context, model M, field string, value any) (*T, error) {
	return r.FindOneByField(ctx, model, field, value)
}

// FindAll 查询全部（不缓存）。
func (r *Repo[T, M]) FindAll(ctx context.Context, dest *[]T) error {
	return r.db.WithContext(ctx).Find(dest).Error
}

// Count 统计（不缓存）。
func (r *Repo[T, M]) Count(ctx context.Context, model M, conds ...any) (int64, error) {
	var count int64
	db := r.db.WithContext(ctx).Model(new(T))
	if len(conds) > 0 {
		db = db.Where(conds[0], conds[1:]...)
	}
	err := db.Count(&count).Error
	return count, err
}

// === 批量操作 ===

// CreateBatch 批量插入并自动失效所有主键缓存。
func (r *Repo[T, M]) CreateBatch(ctx context.Context, models []M) error {
	return r.InsertBatch(ctx, models)
}

// CreateBatchWithFields 批量插入并自动失效主键和指定字段缓存。
func (r *Repo[T, M]) CreateBatchWithFields(ctx context.Context, models []M, fieldValues map[string]any) error {
	return r.InsertBatchWithFields(ctx, models, fieldValues)
}

// DeleteBatch 按条件批量删除并自动失效受影响记录的缓存。
func (r *Repo[T, M]) DeleteBatch(ctx context.Context, conds ...any) error {
	return r.Query.DeleteBatch(ctx, conds...)
}

// UpdateBatch 按条件批量更新并自动失效受影响记录的缓存。
func (r *Repo[T, M]) UpdateBatch(ctx context.Context, updates map[string]any, conds ...any) error {
	return r.Query.UpdateBatch(ctx, updates, conds...)
}

// === 失效 ===

// Invalidate 失效指定缓存 key。
func (r *Repo[T, M]) Invalidate(keys ...string) {
	r.InvalidateKeys(keys...)
}

// InvalidatePKs 批量失效主键缓存。
func (r *Repo[T, M]) InvalidatePKs(models ...M) {
	r.InvalidateByPKs(models...)
}

// InvalidateFields 批量失效字段缓存。
func (r *Repo[T, M]) InvalidateFields(model M, fieldValues map[string]any) {
	r.InvalidateByFields(model, fieldValues)
}

// PrimaryKey 返回主键缓存 key。
func (r *Repo[T, M]) PrimaryKey(model M) string {
	return r.key.Primary(model.CachePrefix(), model.TableName(), model.PrimaryKey())
}

// FieldKey 返回字段缓存 key。
func (r *Repo[T, M]) FieldKey(model M, field string, value any) string {
	return r.key.Field(model.CachePrefix(), model.TableName(), field, value)
}

package crud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"cnbattle.com/ai/pkg/cache"
	"gorm.io/gorm"
)

// Query 提供带缓存的 CRUD 操作。
type Query[T any, M Model] struct {
	*CachedQuery
}

// NewQuery 创建 Query 实例。
func NewQuery[T any, M Model](db *gorm.DB, cache Cache, ttl time.Duration) *Query[T, M] {
	return &Query[T, M]{
		CachedQuery: NewCachedQuery(db, cache, ttl),
	}
}

// === Read-through ===

// FindOneByPK 按主键查询单条（read-through）。
// 缓存完整记录。
func (q *Query[T, M]) FindOneByPK(ctx context.Context, model M) (*T, error) {
	key := q.key.Primary(model.CachePrefix(), model.TableName(), model.PrimaryKey())
	pkCol := model.PrimaryKeyColumn()
	if pkCol == "" {
		pkCol = "id"
	}
	return q.take(ctx, key, func() (string, error) {
		var v T
		err := q.db.WithContext(ctx).Where(pkCol+" = ?", model.PrimaryKey()).First(&v).Error
		if err != nil {
			return "", err
		}
		return marshal(&v)
	})
}

// FindOneByField 按单字段索引查询（read-through）。
// 字段缓存只存主键，完整数据从主键缓存获取。
func (q *Query[T, M]) FindOneByField(ctx context.Context, model M, field string, value any) (*T, error) {
	fieldKey := q.key.Field(model.CachePrefix(), model.TableName(), field, value)
	return q.takeByField(ctx, fieldKey, model, func() (any, error) {
		var v T
		err := q.db.WithContext(ctx).Where(field+" = ?", value).First(&v).Error
		if err != nil {
			return nil, err
		}
		return &v, nil
	})
}

// FindOneByComposite 按复合索引查询（read-through）。
// 字段缓存只存主键，完整数据从主键缓存获取。
func (q *Query[T, M]) FindOneByComposite(ctx context.Context, model M, fields []string, values []any) (*T, error) {
	fieldKey := q.key.Composite(model.CachePrefix(), model.TableName(), fields, values)
	return q.takeByField(ctx, fieldKey, model, func() (any, error) {
		var v T
		where := ""
		for i, f := range fields {
			if i > 0 {
				where += " AND "
			}
			where += f + " = ?"
		}
		err := q.db.WithContext(ctx).Where(where, values...).First(&v).Error
		if err != nil {
			return nil, err
		}
		return &v, nil
	})
}

// Find 查询列表（不缓存）。
func (q *Query[T, M]) Find(ctx context.Context, dest *[]T, conds ...any) error {
	if len(conds) == 0 {
		return q.db.WithContext(ctx).Find(dest).Error
	}
	return q.db.WithContext(ctx).Where(conds[0], conds[1:]...).Find(dest).Error
}

// === 单条写入 ===

// Insert 插入单条记录。
func (q *Query[T, M]) Insert(ctx context.Context, v *T) error {
	return q.db.WithContext(ctx).Create(v).Error
}

// InsertWithFields 插入单条记录并失效主键和指定字段缓存。
// fieldValues 是需要失效的字段缓存，key 为字段名，value 为该字段的值。
func (q *Query[T, M]) InsertWithFields(ctx context.Context, v *T, model M, fieldValues map[string]any) error {
	if err := q.db.WithContext(ctx).Create(v).Error; err != nil {
		return err
	}
	q.InvalidateByPK(model)
	q.InvalidateByFields(model, fieldValues)
	return nil
}

// Update 更新记录并失效主键缓存。
func (q *Query[T, M]) Update(ctx context.Context, model M, v any) error {
	pkCol := model.PrimaryKeyColumn()
	if pkCol == "" {
		pkCol = "id"
	}
	if err := q.db.WithContext(ctx).Model(new(T)).Where(pkCol+" = ?", model.PrimaryKey()).Updates(v).Error; err != nil {
		return err
	}
	q.InvalidateByPK(model)
	return nil
}

// Delete 删除记录并失效主键缓存。
func (q *Query[T, M]) Delete(ctx context.Context, model M) error {
	pkCol := model.PrimaryKeyColumn()
	if pkCol == "" {
		pkCol = "id"
	}
	if err := q.db.WithContext(ctx).Where(pkCol+" = ?", model.PrimaryKey()).Delete(new(T)).Error; err != nil {
		return err
	}
	q.InvalidateByPK(model)
	return nil
}

// === 批量写入 ===

// InsertBatch 批量插入记录并失效主键缓存。
func (q *Query[T, M]) InsertBatch(ctx context.Context, models []M) error {
	if len(models) == 0 {
		return nil
	}
	if err := q.db.WithContext(ctx).Create(models).Error; err != nil {
		return err
	}
	keys := make([]string, len(models))
	for i, m := range models {
		keys[i] = q.key.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey())
	}
	q.InvalidateKeys(keys...)
	return nil
}

// InsertBatchWithFields 批量插入记录并失效主键和指定字段缓存。
func (q *Query[T, M]) InsertBatchWithFields(ctx context.Context, models []M, fieldValues map[string]any) error {
	if len(models) == 0 {
		return nil
	}
	if err := q.db.WithContext(ctx).Create(models).Error; err != nil {
		return err
	}
	keys := make([]string, 0, len(models)*(1+len(fieldValues)))
	for _, m := range models {
		keys = append(keys, q.key.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey()))
		for field, value := range fieldValues {
			keys = append(keys, q.key.Field(m.CachePrefix(), m.TableName(), field, value))
		}
	}
	q.InvalidateKeys(keys...)
	return nil
}

// DeleteBatch 按条件批量删除并自动失效受影响记录的缓存。
// 先查询符合条件的记录，收集主键缓存 key，删除后批量失效。
// 写入失败时不失效缓存。
func (q *Query[T, M]) DeleteBatch(ctx context.Context, conds ...any) error {
	// 查询受影响的记录
	var affected []M
	db := q.db.WithContext(ctx)
	if len(conds) > 0 {
		db = db.Where(conds[0], conds[1:]...)
	}
	if err := db.Find(&affected).Error; err != nil {
		return err
	}

	// 收集需要失效的 key
	keys := make([]string, 0, len(affected))
	for _, m := range affected {
		keys = append(keys, q.key.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey()))
	}

	// 执行删除（失败则不失效缓存）
	if err := db.Delete(new(T)).Error; err != nil {
		return err
	}

	// 删除成功后才失效缓存
	q.InvalidateKeys(keys...)
	return nil
}

// UpdateBatch 按条件批量更新并自动失效受影响记录的缓存。
// 更新失败时不失效缓存。
func (q *Query[T, M]) UpdateBatch(ctx context.Context, updates map[string]any, conds ...any) error {
	// 查询受影响的记录（用于收集缓存 key）
	var affected []M
	db := q.db.WithContext(ctx)
	if len(conds) > 0 {
		db = db.Where(conds[0], conds[1:]...)
	}
	if err := db.Find(&affected).Error; err != nil {
		return err
	}

	// 收集需要失效的 key
	keys := make([]string, 0, len(affected))
	for _, m := range affected {
		keys = append(keys, q.key.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey()))
	}

	// 执行更新（失败则不失效缓存）
	updateDB := q.db.WithContext(ctx).Model(new(T))
	if len(conds) > 0 {
		updateDB = updateDB.Where(conds[0], conds[1:]...)
	}
	if err := updateDB.Updates(updates).Error; err != nil {
		return err
	}

	// 更新成功后才失效缓存
	q.InvalidateKeys(keys...)
	return nil
}

// === 失效 ===

// InvalidateByPK 失效主键缓存。
func (q *Query[T, M]) InvalidateByPK(model M) {
	key := q.key.Primary(model.CachePrefix(), model.TableName(), model.PrimaryKey())
	q.cache.Del(key)
}

// InvalidateByPKs 批量失效主键缓存。
func (q *Query[T, M]) InvalidateByPKs(models ...M) {
	keys := make([]string, len(models))
	for i, m := range models {
		keys[i] = q.key.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey())
	}
	q.InvalidateKeys(keys...)
}

// InvalidateByField 失效字段缓存。
func (q *Query[T, M]) InvalidateByField(model M, field string, value any) {
	key := q.key.Field(model.CachePrefix(), model.TableName(), field, value)
	q.cache.Del(key)
}

// InvalidateByFields 批量失效字段缓存。
// fieldValues 是 field→value 的映射，为每个 model 的每个字段生成 key 并批量失效。
func (q *Query[T, M]) InvalidateByFields(model M, fieldValues map[string]any) {
	keys := make([]string, 0, len(fieldValues))
	for field, value := range fieldValues {
		keys = append(keys, q.key.Field(model.CachePrefix(), model.TableName(), field, value))
	}
	q.InvalidateKeys(keys...)
}

// InvalidateByComposite 失效复合索引缓存。
func (q *Query[T, M]) InvalidateByComposite(model M, fields []string, values []any) {
	key := q.key.Composite(model.CachePrefix(), model.TableName(), fields, values)
	q.cache.Del(key)
}

// InvalidateKeys 批量失效指定 key。
// 小批次（≤8）串行 Del，大批次并发 Del 提升吞吐，上限 50 并发。
func (q *Query[T, M]) InvalidateKeys(keys ...string) {
	if len(keys) == 0 {
		return
	}
	const threshold = 8
	const maxConc = 50
	if len(keys) <= threshold {
		for _, key := range keys {
			q.cache.Del(key)
		}
		return
	}

	limiter := make(chan struct{}, maxConc)
	var wg sync.WaitGroup
	wg.Add(len(keys))
	for _, key := range keys {
		limiter <- struct{}{}
		go func(k string) {
			defer wg.Done()
			defer func() { <-limiter }()
			defer func() { recover() }() // 防止 cache.Del panic 导致进程崩溃
			q.cache.Del(k)
		}(key)
	}
	wg.Wait()
}

// === 内部 ===

// take 从主键缓存读取完整记录（read-through）。
func (q *Query[T, M]) take(ctx context.Context, key string, loader func() (string, error)) (*T, error) {
	val, err := q.cache.GetOrLoad(ctx, key, q.ttl, cache.Loader(func(ctx context.Context) (string, error) {
		return loader()
	}))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, cache.ErrCacheEmpty) {
			q.cache.Del(key)
			return nil, ErrNotFound
		}
		return nil, err
	}

	var v T
	if err := json.Unmarshal([]byte(val), &v); err != nil {
		return nil, fmt.Errorf("unmarshal cache: %w", err)
	}
	return &v, nil
}

// takeByField 从字段缓存读取主键，再从主键缓存获取完整记录。
// 字段缓存只存主键字符串，减少内存占用。
func (q *Query[T, M]) takeByField(ctx context.Context, fieldKey string, model M, loader func() (any, error)) (*T, error) {
	// 1. 尝试从字段缓存读取主键
	pkVal, err := q.cache.GetOrLoad(ctx, fieldKey, q.ttl, cache.Loader(func(ctx context.Context) (string, error) {
		// 字段缓存 miss，查 DB 获取完整记录
		v, dbErr := loader()
		if dbErr != nil {
			return "", dbErr
		}
		// 提取主键
		if m, ok := v.(Model); ok {
			// 同时写入主键缓存
			pkKey := q.key.Primary(m.CachePrefix(), m.TableName(), m.PrimaryKey())
			data, marshalErr := marshal(v)
			if marshalErr != nil {
				return "", fmt.Errorf("marshal pk cache: %w", marshalErr)
			}
			q.cache.Set(pkKey, data, q.ttl)
			// 返回主键字符串给字段缓存
			return fmt.Sprintf("%v", m.PrimaryKey()), nil
		}
		// 不实现 Model 接口时，回退为存完整记录
		return marshal(v)
	}))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrNotFound) {
			q.cache.Del(fieldKey) // 清理无效的字段缓存
			return nil, ErrNotFound
		}
		if errors.Is(err, cache.ErrCacheEmpty) {
			q.cache.Del(fieldKey) // 清理空值缓存
			return nil, ErrNotFound
		}
		return nil, err
	}

	// 2. 检查字段缓存中是否存的是完整记录（兼容旧数据）还是主键
	var v T
	if err := json.Unmarshal([]byte(pkVal), &v); err == nil {
		// 是完整记录（旧格式兼容），直接返回
		return &v, nil
	}

	// 3. 是主键字符串，通过主键获取完整记录
	pkKey := q.key.Primary(model.CachePrefix(), model.TableName(), pkVal)
	pkData, err := q.cache.GetOrLoad(ctx, pkKey, q.ttl, cache.Loader(func(ctx context.Context) (string, error) {
		// 主键缓存 miss，查 DB
		pkCol := model.PrimaryKeyColumn()
		if pkCol == "" {
			pkCol = "id"
		}
		var v T
		if err := q.db.WithContext(ctx).Where(pkCol+" = ?", pkVal).First(&v).Error; err != nil {
			return "", err
		}
		return marshal(&v)
	}))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrNotFound) {
			q.cache.Del(fieldKey) // 主键不存在，清理字段缓存
			return nil, ErrNotFound
		}
		if errors.Is(err, cache.ErrCacheEmpty) {
			q.cache.Del(fieldKey)
			q.cache.Del(pkKey) // 同时清理主键缓存中的空值
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(pkData), &v); err != nil {
		return nil, fmt.Errorf("unmarshal cache: %w", err)
	}
	return &v, nil
}

func marshal(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

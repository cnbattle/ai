package crud

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// Model 定义需要缓存的模型必须实现的接口。
type Model interface {
	// TableName 返回表名。
	TableName() string
	// PrimaryKey 返回主键值。
	PrimaryKey() any
	// PrimaryKeyColumn 返回主键列名，默认 "id"。
	PrimaryKeyColumn() string
	// CachePrefix 返回缓存 key 前缀。
	CachePrefix() string
}

// ErrNotFound 缓存穿透时返回的空值错误（兼容 gorm.ErrRecordNotFound）。
var ErrNotFound = errors.New("record not found")

// CachePrefix 默认前缀。
const CachePrefix = "cache"

// CachedQuery 缓存查询的核心结构，封装 GORM DB 和 ProtectedCache。
type CachedQuery struct {
	db    *gorm.DB
	key   *KeyBuilder
	cache Cache
	ttl   time.Duration
}

// NewCachedQuery 创建 CachedQuery。
func NewCachedQuery(db *gorm.DB, cache Cache, defaultTTL time.Duration) *CachedQuery {
	return &CachedQuery{
		db:    db,
		key:   NewKeyBuilder(),
		cache: cache,
		ttl:   defaultTTL,
	}
}

// DB 返回底层 GORM DB。
func (q *CachedQuery) DB() *gorm.DB {
	return q.db
}

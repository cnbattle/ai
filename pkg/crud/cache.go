package crud

import (
	"context"
	"time"

	"cnbattle.com/ai/pkg/cache"
)

// Cache 缓存接口，兼容 cache.ProtectedCache。
type Cache interface {
	Set(key, value string, exp time.Duration) error
	Get(key string) (string, error)
	Del(key string) error
	DelMulti(keys ...string) error
	GetOrLoad(ctx context.Context, key string, exp time.Duration, loader cache.Loader) (string, error)
	DeleteEmpty(key string) error
}

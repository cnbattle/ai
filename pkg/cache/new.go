package cache

import (
	"context"
	"time"
)

const (
	Redis     = "Redis"
	FreeCache = "FreeCache"
	BigCache  = "BigCache"
)

func NewClient(provider, addr, password string, db, ext int, ctx context.Context) (Cache, error) {
	switch provider {
	case Redis:
		return NewRedisCache(addr, password, db, ctx), nil
	case FreeCache:
		return NewFreeCacheClient(ext), nil
	case BigCache:
		return NewBigCacheClient(time.Duration(ext), ctx), nil
	default:
		return NewRedisCache(addr, password, db, ctx), nil
	}
}

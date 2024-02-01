package cache

import (
	"context"
)

// Init cache
func Init(addr, password string, db int, ctx context.Context) Cache {
	return NewRedisCache(addr, password, db, ctx)
}

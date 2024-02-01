package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCache(addr, password string, db int, ctx context.Context) *RedisCache {
	return &RedisCache{client: redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	}), ctx: ctx}
}

func (r *RedisCache) Set(key, value string, exp time.Duration) error {
	return r.client.Set(r.ctx, key, value, exp).Err()
}

func (r *RedisCache) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

func (r *RedisCache) Del(key string) error {
	del := r.client.Del(r.ctx, key)
	return del.Err()
}

package ai

import (
	"context"
	"errors"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/coocood/freecache"
	"github.com/redis/go-redis/v9"
)

type CacheInterface interface {
	Set(key, value string, exp time.Duration) error
	Get(key string) (string, error)
	Del(key string) error
}

func NewCacheClient(provider, addr, password string, db, ext int, ctx context.Context) (CacheInterface, error) {
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

type BigCacheClient struct {
	client *bigcache.BigCache
}

func NewBigCacheClient(eviction time.Duration, ctx context.Context) *BigCacheClient {
	cache, err := bigcache.New(ctx, bigcache.DefaultConfig(eviction))
	if err != nil {
		panic(err)
	}
	return &BigCacheClient{client: cache}
}

func (r *BigCacheClient) Set(key, value string, _ time.Duration) error {
	return r.client.Set(key, []byte(value))
}

func (r *BigCacheClient) Get(key string) (string, error) {
	value, err := r.client.Get(key)
	return string(value), err
}

func (r *BigCacheClient) Del(key string) error {
	return r.client.Delete(key)
}

type FreeCacheClient struct {
	client *freecache.Cache
}

func NewFreeCacheClient(cacheSize int) *FreeCacheClient {
	if cacheSize == 0 {
		cacheSize = 100 * 1024 * 1024
	}
	cache := freecache.NewCache(cacheSize)
	return &FreeCacheClient{client: cache}
}

func (r *FreeCacheClient) Set(key, value string, exp time.Duration) error {
	return r.client.Set([]byte(key), []byte(value), int(exp))
}

func (r *FreeCacheClient) Get(key string) (string, error) {
	value, err := r.client.Get([]byte(key))
	return string(value), err
}

func (r *FreeCacheClient) Del(key string) error {
	affected := r.client.Del([]byte(key))
	if affected {
		return nil
	}
	return errors.New("cache del error")
}

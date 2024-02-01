package cache

import (
	"context"
	"time"

	"github.com/allegro/bigcache/v3"
)

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

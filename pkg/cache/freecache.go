package cache

import (
	"errors"
	"time"

	"github.com/coocood/freecache"
)

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

package ai

import (
	"context"
	"fmt"
	"time"

	"cnbattle.com/ai/pkg/cache"
)

var Cache *cache.ProtectedCache
var err error

//CACHE=true
//CACHE_PROVIDER=Redis or FreeCache or BigCache
//CACHE_HOST=127.0.0.1:6379
//CACHE_PASS=123456
//CACHE_DB=1
//CACHE_EXT=10
//CACHE_JITTER_RATIO=0.1
//CACHE_EMPTY_TTL=1s

func init() {
	if GetDefaultEnvToBool("CACHE", false) {
		LOG.Trace("auto initialization CACHE")
		c, e := cache.NewClient(GetDefaultEnv("CACHE_PROVIDER", "Redis"),
			GetEnv("CACHE_HOST"),
			GetEnv("CACHE_PASS"),
			GetDefaultEnvToInt("CACHE_DB", 1),
			GetDefaultEnvToInt("CACHE_EXT", 10),
			context.Background(),
		)
		if e != nil {
			panic(fmt.Sprintf("InitCache err:%v", e))
		}
		var opts []cache.Option
		if v := GetDefaultEnvToFloat64("CACHE_JITTER_RATIO", -1); v >= 0 {
			opts = append(opts, cache.WithJitterRatio(v))
		}
		if v := GetDefaultEnv("CACHE_EMPTY_TTL", ""); v != "" {
			d, e := time.ParseDuration(v)
			if e != nil {
				panic(fmt.Sprintf("InitCache CACHE_EMPTY_TTL parse err:%v", e))
			}
			opts = append(opts, cache.WithEmptyTTL(d))
		}
		Cache = cache.NewProtectedCache(c, opts...)
	}
}

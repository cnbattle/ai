package ai

import (
	"context"
	"fmt"

	"cnbattle.com/ai/pkg/cache"
)

var Cache cache.Cache
var err error

//CACHE=true
//CACHE_PROVIDER=Redis or FreeCache or BigCache
//CACHE_HOST=127.0.0.1:6379
//CACHE_PASS=123456
//CACHE_DB=1
//CACHE_EXT=10

func init() {
	if GetDefaultEnvToBool("CACHE", false) {
		LOG.Trace("auto initialization CACHE")
		Cache, err = cache.NewClient(GetDefaultEnv("CACHE_PROVIDER", "Redis"),
			GetEnv("CACHE_HOST"),
			GetEnv("CACHE_PASS"),
			GetDefaultEnvToInt("CACHE_DB", 1),
			GetDefaultEnvToInt("CACHE_EXT", 10),
			context.Background(),
		)
		if err != nil {
			panic(fmt.Sprintf("InitCache err:%v", err))
		}
	}
}

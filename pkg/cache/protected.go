package cache

import (
	"context"
	"math/rand/v2"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	emptyPrefix      = "\x00"
	defaultEmptyTTL  = time.Second
	defaultJitter    = 0.1
)

type Loader func(ctx context.Context) (string, error)

type Option func(*ProtectedCache)

func WithEmptyTTL(d time.Duration) Option {
	return func(p *ProtectedCache) { p.emptyTTL = d }
}

func WithJitterRatio(r float64) Option {
	return func(p *ProtectedCache) { p.jitterRatio = r }
}

type ProtectedCache struct {
	Cache
	sfg         singleflight.Group
	emptyTTL    time.Duration
	jitterRatio float64
}

func NewProtectedCache(c Cache, opts ...Option) *ProtectedCache {
	p := &ProtectedCache{
		Cache:       c,
		emptyTTL:    defaultEmptyTTL,
		jitterRatio: defaultJitter,
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func (p *ProtectedCache) SetWithJitter(ctx context.Context, key, value string, exp time.Duration) error {
	return p.Cache.Set(key, value, p.jitterTTL(exp))
}

func (p *ProtectedCache) GetOrLoad(ctx context.Context, key string, exp time.Duration, loader Loader) (string, error) {
	val, err := p.Cache.Get(key)
	if err == nil {
		if val == emptyPrefix {
			return "", ErrCacheEmpty
		}
		return val, nil
	}

	v, err, _ := p.sfg.Do(key, func() (interface{}, error) {
		val, err := p.Cache.Get(key)
		if err == nil {
			if val == emptyPrefix {
				return "", ErrCacheEmpty
			}
			return val, nil
		}

		val, err = loader(ctx)
		if err != nil {
			return "", err
		}

		if val == "" {
			p.Cache.Set(key, emptyPrefix, p.emptyTTL)
			return "", ErrCacheEmpty
		}

		p.Cache.Set(key, val, p.jitterTTL(exp))
		return val, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (p *ProtectedCache) DeleteEmpty(key string) error {
	return p.Cache.Del(key)
}

func (p *ProtectedCache) jitterTTL(exp time.Duration) time.Duration {
	jitter := time.Duration(float64(exp) * p.jitterRatio)
	if jitter <= 0 {
		return exp
	}
	return exp - jitter + time.Duration(rand.Int64N(int64(jitter*2)))
}

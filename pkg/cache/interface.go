package cache

import (
	"errors"
	"time"
)

var ErrCacheEmpty = errors.New("cache: empty value")

type Cache interface {
	Set(key, value string, exp time.Duration) error
	Get(key string) (string, error)
	Del(key string) error
}

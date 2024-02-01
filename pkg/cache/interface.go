package cache

import "time"

type Cache interface {
	Set(key, value string, exp time.Duration) error
	Get(key string) (string, error)
	Del(key string) error
}

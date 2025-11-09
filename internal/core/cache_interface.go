package core

import "time"

type CacheInterface interface {
	Set(key string, value []byte, expiration time.Duration) error
	Get(key string) ([]byte, bool)
	Del(key string) error
	Exists(key string) bool
	Keys() []string
	Flush() error
	TTL(key string) (time.Duration, bool)              // returns remaining TTL and whether the key exists
	Expire(key string, expiration time.Duration) error // updates the TTL of a key
}

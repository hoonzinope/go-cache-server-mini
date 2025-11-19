package router

import "time"

type DistributorInterface interface {
	Set(key string, value []byte, expiration time.Duration) error
	Get(key string) ([]byte, bool, error)
	Del(key string) error
	Exists(key string) (bool, error)
	Keys() ([]string, error)
	Flush() error
	TTL(key string) (time.Duration, bool, error)
	Expire(key string, expiration time.Duration) error
	Persist(key string) error
	Incr(key string) (int64, error)
	Decr(key string) (int64, error)
	SetNX(key string, value []byte, expiration time.Duration) (bool, error)
	GetSet(key string, value []byte) ([]byte, error)
	MGet(keys []string) (map[string][]byte, error)
	MSet(kv map[string][]byte, expiration time.Duration) error
}

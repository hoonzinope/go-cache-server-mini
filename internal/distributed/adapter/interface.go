package adapter

import (
	"time"
)

type AdapterInterface interface {
	SetItem(key string, value []byte, expiration time.Duration) error
	GetItem(key string) ([]byte, bool)
	DeleteItem(key string) error
	ExistsItem(key string) bool
	ListKeys() []string
	ClearCache() error
	GetTTL(key string) (time.Duration, bool)
	UpdateExpiration(key string, expiration time.Duration) error
	RemoveExpiration(key string) error
	Increment(key string) (int64, error)
	Decrement(key string) (int64, error)
	SetIfNotExists(key string, value []byte, expiration time.Duration) (bool, error)
	GetAndSet(key string, value []byte) ([]byte, error)
	GetMultiple(keys []string) map[string][]byte
	SetMultiple(kv map[string][]byte, expiration time.Duration) error
}

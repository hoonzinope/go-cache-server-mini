package adapter

import (
	"go-cache-server-mini/internal/core"
	"time"
)

type LocalAdapter struct {
	Cache core.CacheInterface
}

func NewLocalAdapter(cache core.CacheInterface) *LocalAdapter {
	return &LocalAdapter{
		Cache: cache,
	}
}

func (la *LocalAdapter) SetItem(key string, value []byte, expiration time.Duration) error {
	return la.Cache.Set(key, value, expiration)
}

func (la *LocalAdapter) GetItem(key string) ([]byte, bool) {
	return la.Cache.Get(key)
}

func (la *LocalAdapter) DeleteItem(key string) error {
	return la.Cache.Del(key)
}

func (la *LocalAdapter) ExistsItem(key string) bool {
	return la.Cache.Exists(key)
}

func (la *LocalAdapter) ListKeys() []string {
	return la.Cache.Keys()
}

func (la *LocalAdapter) ClearCache() error {
	return la.Cache.Flush()
}

func (la *LocalAdapter) GetTTL(key string) (time.Duration, bool) {
	return la.Cache.TTL(key)
}

func (la *LocalAdapter) UpdateExpiration(key string, expiration time.Duration) error {
	return la.Cache.Expire(key, expiration)
}

func (la *LocalAdapter) RemoveExpiration(key string) error {
	return la.Cache.Persist(key)
}

func (la *LocalAdapter) Increment(key string) (int64, error) {
	return la.Cache.Incr(key)
}

func (la *LocalAdapter) Decrement(key string) (int64, error) {
	return la.Cache.Decr(key)
}

func (la *LocalAdapter) SetIfNotExists(key string, value []byte, expiration time.Duration) (bool, error) {
	return la.Cache.SetNX(key, value, expiration)
}

func (la *LocalAdapter) GetAndSet(key string, value []byte) ([]byte, error) {
	return la.Cache.GetSet(key, value)
}

func (la *LocalAdapter) GetMultiple(keys []string) map[string][]byte {
	return la.Cache.MGet(keys)
}

func (la *LocalAdapter) SetMultiple(kv map[string][]byte, expiration time.Duration) error {
	return la.Cache.MSet(kv, expiration)
}

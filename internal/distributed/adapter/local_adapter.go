package adapter

import (
	"context"
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

func (la *LocalAdapter) SetItem(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return la.Cache.Set(key, value, expiration)
}

func (la *LocalAdapter) GetItem(ctx context.Context, key string) ([]byte, bool) {
	return la.Cache.Get(key)
}

func (la *LocalAdapter) DeleteItem(ctx context.Context, key string) error {
	return la.Cache.Del(key)
}

func (la *LocalAdapter) ExistsItem(ctx context.Context, key string) bool {
	return la.Cache.Exists(key)
}

func (la *LocalAdapter) ListKeys(ctx context.Context) []string {
	return la.Cache.Keys()
}

func (la *LocalAdapter) ClearCache(ctx context.Context) error {
	return la.Cache.Flush()
}

func (la *LocalAdapter) GetTTL(ctx context.Context, key string) (time.Duration, bool) {
	return la.Cache.TTL(key)
}

func (la *LocalAdapter) UpdateExpiration(ctx context.Context, key string, expiration time.Duration) error {
	return la.Cache.Expire(key, expiration)
}

func (la *LocalAdapter) RemoveExpiration(ctx context.Context, key string) error {
	return la.Cache.Persist(key)
}

func (la *LocalAdapter) Increment(ctx context.Context, key string) (int64, error) {
	return la.Cache.Incr(key)
}

func (la *LocalAdapter) Decrement(ctx context.Context, key string) (int64, error) {
	return la.Cache.Decr(key)
}

func (la *LocalAdapter) SetIfNotExists(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	return la.Cache.SetNX(key, value, expiration)
}

func (la *LocalAdapter) GetAndSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	return la.Cache.GetSet(key, value)
}

func (la *LocalAdapter) GetMultiple(ctx context.Context, keys []string) map[string][]byte {
	return la.Cache.MGet(keys)
}

func (la *LocalAdapter) SetMultiple(ctx context.Context, kv map[string][]byte, expiration time.Duration) error {
	return la.Cache.MSet(kv, expiration)
}

package router

import (
	"go-cache-server-mini/internal/distributed/adapter"
	"time"
)

type LocalDistributor struct {
	localAdapter adapter.AdapterInterface
}

func NewLocalDistributor(nodeRouter *NodeRouter) *LocalDistributor {
	localAdapter := nodeRouter.GetLocalAdapter()
	if localAdapter == nil {
		return nil
	}
	return &LocalDistributor{
		localAdapter: localAdapter,
	}
}

func (ld *LocalDistributor) Set(key string, value []byte, expiration time.Duration) error {
	return ld.localAdapter.SetItem(key, value, expiration)
}

func (ld *LocalDistributor) Get(key string) ([]byte, bool, error) {
	if value, found := ld.localAdapter.GetItem(key); found {
		return value, true, nil
	}
	return nil, false, nil
}

func (ld *LocalDistributor) Del(key string) error {
	return ld.localAdapter.DeleteItem(key)
}

func (ld *LocalDistributor) Exists(key string) (bool, error) {
	return ld.localAdapter.ExistsItem(key), nil
}

func (ld *LocalDistributor) Keys() ([]string, error) {
	keys := ld.localAdapter.ListKeys()
	return keys, nil
}

func (ld *LocalDistributor) Flush() error {
	return ld.localAdapter.ClearCache()
}

func (ld *LocalDistributor) TTL(key string) (time.Duration, bool, error) {
	ttl, found := ld.localAdapter.GetTTL(key)
	return ttl, found, nil
}

func (ld *LocalDistributor) Expire(key string, expiration time.Duration) error {
	return ld.localAdapter.UpdateExpiration(key, expiration)
}

func (ld *LocalDistributor) Persist(key string) error {
	return ld.localAdapter.RemoveExpiration(key)
}

func (ld *LocalDistributor) Incr(key string) (int64, error) {
	return ld.localAdapter.Increment(key)
}

func (ld *LocalDistributor) Decr(key string) (int64, error) {
	return ld.localAdapter.Decrement(key)
}

func (ld *LocalDistributor) SetNX(key string, value []byte, expiration time.Duration) (bool, error) {
	return ld.localAdapter.SetIfNotExists(key, value, expiration)
}

func (ld *LocalDistributor) GetSet(key string, value []byte) ([]byte, error) {
	return ld.localAdapter.GetAndSet(key, value)
}

func (ld *LocalDistributor) MGet(keys []string) (map[string][]byte, error) {
	result := ld.localAdapter.GetMultiple(keys)
	return result, nil
}

func (ld *LocalDistributor) MSet(items map[string][]byte, expiration time.Duration) error {
	return ld.localAdapter.SetMultiple(items, expiration)
}

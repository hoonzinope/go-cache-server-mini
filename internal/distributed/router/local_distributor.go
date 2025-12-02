package router

import (
	"context"
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

func (ld *LocalDistributor) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	return ld.localAdapter.SetItem(ctx, key, value, expiration)
}

func (ld *LocalDistributor) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if value, found := ld.localAdapter.GetItem(ctx, key); found {
		return value, true, nil
	}
	return nil, false, nil
}

func (ld *LocalDistributor) Del(ctx context.Context, key string) error {
	return ld.localAdapter.DeleteItem(ctx, key)
}

func (ld *LocalDistributor) Exists(ctx context.Context, key string) (bool, error) {
	return ld.localAdapter.ExistsItem(ctx, key), nil
}

func (ld *LocalDistributor) Keys(ctx context.Context) ([]string, error) {
	keys := ld.localAdapter.ListKeys(ctx)
	return keys, nil
}

func (ld *LocalDistributor) Flush(ctx context.Context) error {
	return ld.localAdapter.ClearCache(ctx)
}

func (ld *LocalDistributor) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	ttl, found := ld.localAdapter.GetTTL(ctx, key)
	return ttl, found, nil
}

func (ld *LocalDistributor) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return ld.localAdapter.UpdateExpiration(ctx, key, expiration)
}

func (ld *LocalDistributor) Persist(ctx context.Context, key string) error {
	return ld.localAdapter.RemoveExpiration(ctx, key)
}

func (ld *LocalDistributor) Incr(ctx context.Context, key string) (int64, error) {
	return ld.localAdapter.Increment(ctx, key)
}

func (ld *LocalDistributor) Decr(ctx context.Context, key string) (int64, error) {
	return ld.localAdapter.Decrement(ctx, key)
}

func (ld *LocalDistributor) SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	return ld.localAdapter.SetIfNotExists(ctx, key, value, expiration)
}

func (ld *LocalDistributor) GetSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	return ld.localAdapter.GetAndSet(ctx, key, value)
}

func (ld *LocalDistributor) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	result := ld.localAdapter.GetMultiple(ctx, keys)
	return result, nil
}

func (ld *LocalDistributor) MSet(ctx context.Context, items map[string][]byte, expiration time.Duration) error {
	return ld.localAdapter.SetMultiple(ctx, items, expiration)
}

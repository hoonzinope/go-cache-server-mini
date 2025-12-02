package router

import (
	"context"
	"time"
)

type DistributorInterface interface {
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Del(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Keys(ctx context.Context) ([]string, error)
	Flush(ctx context.Context) error
	TTL(ctx context.Context, key string) (time.Duration, bool, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Persist(ctx context.Context, key string) error
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error)
	GetSet(ctx context.Context, key string, value []byte) ([]byte, error)
	MGet(ctx context.Context, keys []string) (map[string][]byte, error)
	MSet(ctx context.Context, kv map[string][]byte, expiration time.Duration) error
}

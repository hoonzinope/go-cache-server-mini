package adapter

import (
	"context"
	"time"
)

type AdapterInterface interface {
	SetItem(ctx context.Context, key string, value []byte, expiration time.Duration) error
	GetItem(ctx context.Context, key string) ([]byte, bool)
	DeleteItem(ctx context.Context, key string) error
	ExistsItem(ctx context.Context, key string) bool
	ListKeys(ctx context.Context) []string
	ClearCache(ctx context.Context) error
	GetTTL(ctx context.Context, key string) (time.Duration, bool)
	UpdateExpiration(ctx context.Context, key string, expiration time.Duration) error
	RemoveExpiration(ctx context.Context, key string) error
	Increment(ctx context.Context, key string) (int64, error)
	Decrement(ctx context.Context, key string) (int64, error)
	SetIfNotExists(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error)
	GetAndSet(ctx context.Context, key string, value []byte) ([]byte, error)
	GetMultiple(ctx context.Context, keys []string) map[string][]byte
	SetMultiple(ctx context.Context, kv map[string][]byte, expiration time.Duration) error
}

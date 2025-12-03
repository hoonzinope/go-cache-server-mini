package adapter

import (
	"context"
	"go-cache-server-mini/internal/grpc/client"
	"log"
	"sync"
	"time"
)

type RemoteAdapter struct {
	remoteIp   string
	grpcClient *client.GRPCCacheClient
	mu         sync.RWMutex
	isClosed   bool
}

// NewRemoteAdapter creates a new instance of RemoteAdapter
func NewRemoteAdapter(remoteIp string, remotePort int) (*RemoteAdapter, error) {
	grpcClient, err := client.NewGRPCCacheClient(remoteIp, remotePort)
	if err != nil {
		log.Println("Failed to create GRPC client:", err)
		return nil, err
	}
	return &RemoteAdapter{
		remoteIp:   remoteIp,
		grpcClient: grpcClient,
		mu:         sync.RWMutex{},
		isClosed:   false,
	}, nil
}

func (ra *RemoteAdapter) GetNodeIP() string {
	return ra.remoteIp
}

func (ra *RemoteAdapter) Close() {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	if ra.isClosed {
		return
	}
	err := ra.grpcClient.Close()
	if err != nil {
		log.Println("Failed to close GRPC client:", err)
	}
	ra.isClosed = true
}

func (ra *RemoteAdapter) SetItem(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	ttlSeconds := int64(expiration.Seconds())
	if err := ra.grpcClient.Set(ctx, key, value, ttlSeconds); err != nil {
		log.Println("Failed to set item in remote cache:", err)
		return err
	}
	return nil
}

func (ra *RemoteAdapter) GetItem(ctx context.Context, key string) ([]byte, bool) {
	value, found, err := ra.grpcClient.Get(ctx, key)
	if err != nil {
		log.Println("Failed to get item from remote cache:", err)
		return nil, false
	}
	if found {
		return value, true
	}
	return nil, false
}

func (ra *RemoteAdapter) DeleteItem(ctx context.Context, key string) error {
	if err := ra.grpcClient.Del(ctx, key); err != nil {
		log.Println("Failed to delete item from remote cache:", err)
		return err
	}
	return nil
}

func (ra *RemoteAdapter) ExistsItem(ctx context.Context, key string) bool {
	exists, err := ra.grpcClient.Exists(ctx, key)
	if err != nil {
		log.Println("Failed to check existence of item in remote cache:", err)
		return false
	}
	return exists
}

func (ra *RemoteAdapter) ListKeys(ctx context.Context) []string {
	keys, err := ra.grpcClient.Keys(ctx)
	if err != nil {
		log.Println("Failed to list keys from remote cache:", err)
		return []string{}
	}
	return keys
}

func (ra *RemoteAdapter) ClearCache(ctx context.Context) error {
	if err := ra.grpcClient.Flush(ctx); err != nil {
		log.Println("Failed to flush remote cache:", err)
		return err
	}
	return nil
}

func (ra *RemoteAdapter) GetTTL(ctx context.Context, key string) (time.Duration, bool) {
	ttl, found, err := ra.grpcClient.TTL(ctx, key)
	if err != nil {
		log.Println("Failed to get TTL from remote cache:", err)
		return 0, false
	}
	return ttl, found
}

func (ra *RemoteAdapter) UpdateExpiration(ctx context.Context, key string, expiration time.Duration) error {
	ttlSeconds := int64(expiration.Seconds())
	if err := ra.grpcClient.Expire(ctx, key, ttlSeconds); err != nil {
		log.Println("Failed to update expiration in remote cache:", err)
		return err
	}
	return nil
}

func (ra *RemoteAdapter) RemoveExpiration(ctx context.Context, key string) error {
	if err := ra.grpcClient.Persist(ctx, key); err != nil {
		log.Println("Failed to persist key in remote cache:", err)
		return err
	}
	return nil
}

func (ra *RemoteAdapter) Increment(ctx context.Context, key string) (int64, error) {
	val, err := ra.grpcClient.Incr(ctx, key)
	if err != nil {
		log.Println("Failed to increment key in remote cache:", err)
		return 0, err
	}
	return val, nil
}

func (ra *RemoteAdapter) Decrement(ctx context.Context, key string) (int64, error) {
	val, err := ra.grpcClient.Decr(ctx, key)
	if err != nil {
		log.Println("Failed to decrement key in remote cache:", err)
		return 0, err
	}
	return val, nil
}

func (ra *RemoteAdapter) SetIfNotExists(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	ttlSeconds := int64(expiration.Seconds())
	success, err := ra.grpcClient.SetNX(ctx, key, value, ttlSeconds)
	if err != nil {
		log.Println("Failed to setnx in remote cache:", err)
		return false, err
	}
	return success, nil
}

func (ra *RemoteAdapter) GetAndSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	oldValue, found, err := ra.grpcClient.GetSet(ctx, key, value)
	if err != nil {
		log.Println("Failed to getset in remote cache:", err)
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return oldValue, nil
}

func (ra *RemoteAdapter) GetMultiple(ctx context.Context, keys []string) map[string][]byte {
	result, err := ra.grpcClient.MGet(ctx, keys)
	if err != nil {
		log.Println("Failed to mget from remote cache:", err)
		return nil
	}
	return result
}

func (ra *RemoteAdapter) SetMultiple(ctx context.Context, kv map[string][]byte, expiration time.Duration) error {
	if err := ra.grpcClient.MSet(ctx, kv, int64(expiration.Seconds())); err != nil {
		log.Println("Failed to mset in remote cache:", err)
		return err
	}
	return nil
}

package adapter

import (
	"context"
	"go-cache-server-mini/internal/grpc/client"
	"log"
	"sync"
	"time"
)

type RemoteAdapter struct {
	ctx        context.Context
	remoteIp   string
	grpcClient *client.GRPCCacheClient
	mu         sync.RWMutex
	isClosed   bool
}

// NewRemoteAdapter creates a new instance of RemoteAdapter
func NewRemoteAdapter(ctx context.Context, remoteIp string, remotePort int) (*RemoteAdapter, error) {
	grpcClient, err := client.NewGRPCCacheClient(ctx, remoteIp, remotePort)
	if err != nil {
		log.Println("Failed to create GRPC client:", err)
		return nil, err
	}
	return &RemoteAdapter{
		ctx:        ctx,
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

func (ra *RemoteAdapter) SetItem(key string, value []byte, expiration time.Duration) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	err := ra.grpcClient.Set(ra.ctx, key, value, int64(expiration.Seconds()))
	if err != nil {
		log.Println("Failed to set item in remote cache:", err)
		return err
	}
	return nil
}

func (ra *RemoteAdapter) GetItem(key string) ([]byte, bool) {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	value, found, err := ra.grpcClient.Get(ra.ctx, key)
	if err != nil {
		log.Println("Failed to get item from remote cache:", err)
		return nil, false
	}
	if found {
		return value, true
	}
	log.Println("Item not found in remote cache for key:", key)
	return nil, false
}

func (ra *RemoteAdapter) DeleteItem(key string) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	delErr := ra.grpcClient.Del(ra.ctx, key)
	if delErr != nil {
		log.Println("Failed to delete item from remote cache:", delErr)
		return delErr
	}
	return nil
}

func (ra *RemoteAdapter) ExistsItem(key string) bool {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	_, found, err := ra.grpcClient.Get(ra.ctx, key)
	if err != nil {
		log.Println("Failed to check existence of item in remote cache:", err)
		return false
	}
	if found {
		return true
	}
	log.Println("Item does not exist in remote cache for key:", key)
	return false
}

func (ra *RemoteAdapter) ListKeys() []string {
	// no implementation for listing keys in remote cache
	return []string{}
}

func (ra *RemoteAdapter) ClearCache() error {
	// no implementation for clearing remote cache
	return nil
}

func (ra *RemoteAdapter) GetTTL(key string) (time.Duration, bool) {
	return 0, false
}

func (ra *RemoteAdapter) UpdateExpiration(key string, expiration time.Duration) error {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	value, found, err := ra.grpcClient.Get(ra.ctx, key)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	err = ra.grpcClient.Set(ra.ctx, key, value, int64(expiration.Seconds()))
	if err != nil {
		return err
	}

	return nil
}

func (ra *RemoteAdapter) RemoveExpiration(key string) error {
	// no implementation for removing expiration in remote cache
	return nil
}

func (ra *RemoteAdapter) Increment(key string) (int64, error) {
	// no implementation for incrementing item in remote cache
	return 0, nil
}

func (ra *RemoteAdapter) Decrement(key string) (int64, error) {
	// no implementation for decrementing item in remote cache
	return 0, nil
}

func (ra *RemoteAdapter) SetIfNotExists(key string, value []byte, expiration time.Duration) (bool, error) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	exists := ra.ExistsItem(key)
	if !exists {
		err := ra.grpcClient.Set(ra.ctx, key, value, int64(expiration.Seconds()))
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (ra *RemoteAdapter) GetAndSet(key string, value []byte) ([]byte, error) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	oldValue, found, err := ra.grpcClient.Get(ra.ctx, key)
	if err != nil {
		return nil, err
	}
	err = ra.grpcClient.Set(ra.ctx, key, value, 0)
	if err != nil {
		return nil, err
	}
	if found {
		return oldValue, nil
	}
	return nil, nil
}

func (ra *RemoteAdapter) GetMultiple(keys []string) map[string][]byte {
	// no implementation for getting multiple items from remote cache
	return nil
}

func (ra *RemoteAdapter) SetMultiple(kv map[string][]byte, expiration time.Duration) error {
	// no implementation for setting multiple items in remote cache
	return nil
}

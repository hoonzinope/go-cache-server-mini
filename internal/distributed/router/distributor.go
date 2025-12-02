package router

import (
	"context"
	"go-cache-server-mini/internal/distributed/adapter"
	"log"
	"time"
)

type Distributor struct {
	nodeRouter   *NodeRouter
	localAdapter adapter.AdapterInterface
}

func NewDistributor(nodeRouter *NodeRouter) *Distributor {
	localAdapter := nodeRouter.GetLocalAdapter()
	if localAdapter == nil {
		log.Println("Local adapter not found in node router")
		return nil
	}
	return &Distributor{
		nodeRouter:   nodeRouter,
		localAdapter: localAdapter,
	}
}

func (d *Distributor) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	localAdapter := d.localAdapter
	if err := localAdapter.SetItem(ctx, key, value, expiration); err != nil {
		return err
	}

	// Create a context that cannot be canceled
	ctxwoCancel := context.WithoutCancel(ctx)
	adapters, err := d.nodeRouter.GetAdapters(key)
	if err != nil {
		return err
	}
	go func() {
		for _, adapter := range adapters {
			if err := adapter.SetItem(ctxwoCancel, key, value, expiration); err != nil {
				log.Printf("Failed to replicate set item for key %s: %v", key, err)
			}
		}
	}()
	return nil
}

func (d *Distributor) Get(ctx context.Context, key string) ([]byte, bool, error) {
	localAdapter := d.localAdapter
	if value, found := localAdapter.GetItem(ctx, key); found {
		return value, true, nil
	}

	// TODO: Optimize by getting from only relevant adapters
	adapters, err := d.nodeRouter.GetAdapters(key)
	if err != nil {
		return nil, false, err
	}
	for _, adapter := range adapters {
		if value, found := adapter.GetItem(ctx, key); found {
			return value, true, nil
		}
	}
	return nil, false, nil
}

func (d *Distributor) Del(ctx context.Context, key string) error {
	localAdapter := d.localAdapter
	if err := localAdapter.DeleteItem(ctx, key); err != nil {
		return err
	}

	// TODO: Optimize by deleting from only relevant adapters
	adapters, err := d.nodeRouter.GetAdapters(key)
	if err != nil {
		return err
	}
	go func() {
		for _, adapter := range adapters {
			adapter.DeleteItem(ctx, key)
		}
	}()
	return nil
}

func (d *Distributor) Exists(ctx context.Context, key string) (bool, error) {
	localAdapter := d.localAdapter
	if localAdapter.ExistsItem(ctx, key) {
		return true, nil
	}

	// TODO: Optimize by checking only relevant adapters
	adapters, err := d.nodeRouter.GetAdapters(key)
	if err != nil {
		return false, err
	}
	for _, adapter := range adapters {
		if adapter.ExistsItem(ctx, key) {
			return true, nil
		}
	}
	return false, nil
}

func (d *Distributor) Keys(ctx context.Context) ([]string, error) {
	var allKeys []string
	localAdapter := d.localAdapter
	if localAdapter != nil {
		keys := localAdapter.ListKeys(ctx)
		allKeys = append(allKeys, keys...)
	}
	// TODO: Consider fetching keys from other adapters if needed
	return allKeys, nil
}

func (d *Distributor) Flush(ctx context.Context) error {
	localAdapter := d.localAdapter
	// TODO: Consider flushing other adapters if needed
	return localAdapter.ClearCache(ctx)
}

func (d *Distributor) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	localAdapter := d.localAdapter
	if ttl, found := localAdapter.GetTTL(ctx, key); found {
		return ttl, true, nil
	}
	// TODO: Optimize by getting from only relevant adapters
	// adapters, err := d.nodeRouter.GetAdapters(key)
	// if err != nil {
	// 	return 0, false
	// }
	// for _, adapter := range adapters {
	// 	if ttl, found := adapter.GetTTL(key); found {
	// 		return ttl, true, nil
	// 	}
	// }
	return 0, false, nil
}

func (d *Distributor) Expire(ctx context.Context, key string, expiration time.Duration) error {
	localAdapter := d.localAdapter
	if err := localAdapter.UpdateExpiration(ctx, key, expiration); err != nil {
		return err
	}
	// TODO: Optimize by updating only on relevant adapters
	adapters, err := d.nodeRouter.GetAdapters(key)
	if err != nil {
		return err
	}
	go func() {
		for _, adapter := range adapters {
			if err := adapter.UpdateExpiration(ctx, key, expiration); err != nil {
				return
			}
		}
	}()
	return nil
}

func (d *Distributor) Persist(ctx context.Context, key string) error {
	localAdapter := d.localAdapter
	if err := localAdapter.RemoveExpiration(ctx, key); err != nil {
		return err
	}
	adapters, err := d.nodeRouter.GetAdapters(key)
	if err != nil {
		return err
	}
	go func() {
		for _, adapter := range adapters {
			if err := adapter.RemoveExpiration(ctx, key); err != nil {
				return
			}
		}
	}()
	return nil
}

func (d *Distributor) Incr(ctx context.Context, key string) (int64, error) {
	localAdapter := d.localAdapter
	val, err := localAdapter.Increment(ctx, key)
	if err != nil {
		return 0, err
	}

	// TODO: Optimize by incrementing only on relevant adapters
	// adapters, err := d.nodeRouter.GetAdapters(key)
	// if err != nil {
	// 	return 0, err
	// }
	// var result int64
	// for _, adapter := range adapters {
	// 	val, err := adapter.Increment(key)
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// 	result = val
	// }
	return val, nil
}

func (d *Distributor) Decr(ctx context.Context, key string) (int64, error) {
	localAdapter := d.localAdapter
	val, err := localAdapter.Decrement(ctx, key)
	if err != nil {
		return 0, err
	}
	// TODO: Optimize by decrementing only on relevant adapters
	// adapters, err := d.nodeRouter.GetAdapters(key)
	// if err != nil {
	// 	return 0, err
	// }
	// var result int64
	// for _, adapter := range adapters {
	// 	val, err := adapter.Decrement(key)
	// 	if err != nil {
	// 		return 0, err
	// 	}
	// 	result = val
	// }
	return val, nil
}

func (d *Distributor) SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	localAdapter := d.localAdapter
	success, err := localAdapter.SetIfNotExists(ctx, key, value, expiration)
	if err != nil {
		return false, err
	}
	if success {
		return true, nil
	}
	// TODO: Optimize by setting only on relevant adapters
	adapters, err := d.nodeRouter.GetAdapters(key)
	if err != nil {
		return false, err
	}
	var setSuccess bool
	for _, adapter := range adapters {
		success, err := adapter.SetIfNotExists(ctx, key, value, expiration)
		if err != nil {
			return false, err
		}
		if success {
			setSuccess = true
		}
	}
	return setSuccess, nil
}

func (d *Distributor) GetSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	localAdapter := d.localAdapter
	oldValue, err := localAdapter.GetAndSet(ctx, key, value)
	if err != nil {
		return nil, err
	}
	if oldValue != nil {
		return oldValue, nil
	}

	// TODO : Optimize by getting and setting on only relevant adapters
	adapters, err := d.nodeRouter.GetAdapters(key)
	if err != nil {
		return nil, err
	}
	for _, adapter := range adapters {
		val, err := adapter.GetAndSet(ctx, key, value)
		if err != nil {
			return nil, err
		}
		oldValue = val
	}
	return oldValue, nil
}

func (d *Distributor) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	localAdapter := d.localAdapter
	result := localAdapter.GetMultiple(ctx, keys)
	// TODO: Optimize by getting from only relevant adapters
	return result, nil
}

func (d *Distributor) MSet(ctx context.Context, kv map[string][]byte, expiration time.Duration) error {
	localAdapter := d.localAdapter
	if err := localAdapter.SetMultiple(ctx, kv, expiration); err != nil {
		return err
	}

	ctxwoCancel := context.WithoutCancel(ctx)
	for key := range kv {
		value := kv[key]
		adapters, err := d.nodeRouter.GetAdapters(key)
		if err != nil {
			return err
		}
		go func() {
			for _, adapter := range adapters {
				if err := adapter.SetItem(ctxwoCancel, key, value, expiration); err != nil {
					log.Printf("Failed to replicate set item for key %s: %v", key, err)
				}
			}
		}()
	}
	return nil
}

package core

import (
	"context"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/util"
	"sync"
	"time"
)

type cacheItem struct {
	value      []byte
	expiration time.Time
	persistent bool
}

type Cache struct {
	Lock       sync.RWMutex
	KVMap      map[string]cacheItem
	defaultTTL int64
	maxTTL     int64
}

func NewCache(ctx context.Context, defaultTTL, maxTTL int64) *Cache {
	cache := &Cache{
		KVMap:      make(map[string]cacheItem),
		defaultTTL: defaultTTL,
		maxTTL:     maxTTL,
	}
	go cache.expireWorker(ctx)
	return cache
}

func (c *Cache) expireWorker(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			c.Lock.Lock()
			for key, item := range c.KVMap {
				if isExpired(item) {
					delete(c.KVMap, key)
				}
			}
			c.Lock.Unlock()
		}
	}
}

func (c *Cache) Set(key string, value []byte, expiration time.Duration) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	c.KVMap[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
		persistent: persistent,
	}
	return nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	item, exists := c.KVMap[key]
	if exists {
		if isExpired(item) {
			return nil, false
		}
		return item.value, true
	}
	return nil, false
}

func (c *Cache) Del(key string) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	delete(c.KVMap, key)
	return nil
}

func (c *Cache) Exists(key string) bool {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	item, exists := c.KVMap[key]
	if exists {
		return !isExpired(item)
	}
	return false
}

func (c *Cache) Keys() []string {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	var keys []string
	for key, item := range c.KVMap {
		if !isExpired(item) {
			keys = append(keys, key)
		}
	}
	return keys
}

func (c *Cache) Flush() error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	c.KVMap = make(map[string]cacheItem)
	return nil
}

func (c *Cache) TTL(key string) (time.Duration, bool) {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	item, exists := c.KVMap[key]
	if !exists {
		return 0, false
	}
	if item.persistent {
		return -1, true
	}
	remaining := time.Until(item.expiration)
	if remaining <= 0 {
		return 0, false
	}
	return remaining, true
}

func (c *Cache) Expire(key string, expiration time.Duration) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	if expiration <= 0 {
		delete(c.KVMap, key)
		return nil
	}

	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	item, exists := c.KVMap[key]
	if !exists {
		return internal.ErrNotFound
	}
	if isExpired(item) {
		return internal.ErrNotFound
	}
	c.KVMap[key] = cacheItem{
		value:      item.value,
		expiration: time.Now().Add(expiration),
		persistent: persistent,
	}
	return nil
}

func (c *Cache) Persist(key string) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	item, exists := c.KVMap[key]
	if !exists || isExpired(item) {
		return internal.ErrNotFound
	}
	c.KVMap[key] = cacheItem{
		value:      item.value,
		expiration: time.Time{},
		persistent: true,
	}
	return nil
}

func (c *Cache) Incr(key string) (int64, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()

	item, exists := c.KVMap[key]
	if !exists || isExpired(item) {
		return 0, internal.ErrNotFound
	}
	value, err := util.BytesToInt64(item.value)
	if err != nil {
		return 0, err
	}
	value++
	c.KVMap[key] = cacheItem{
		value:      util.Int64ToBytes(value),
		expiration: item.expiration,
		persistent: item.persistent,
	}
	return value, nil
}

func (c *Cache) Decr(key string) (int64, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	item, exists := c.KVMap[key]
	if !exists || isExpired(item) {
		return 0, internal.ErrNotFound
	}
	value, err := util.BytesToInt64(item.value)
	if err != nil {
		return 0, err
	}
	value--
	c.KVMap[key] = cacheItem{
		value:      util.Int64ToBytes(value),
		expiration: item.expiration,
		persistent: item.persistent,
	}
	return value, nil
}

func (c *Cache) SetNX(key string, value []byte, expiration time.Duration) (bool, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	item, exists := c.KVMap[key]
	if exists && !isExpired(item) {
		return false, nil
	}
	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	c.KVMap[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
		persistent: persistent,
	}
	return true, nil
}

func (c *Cache) GetSet(key string, value []byte) ([]byte, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	item, exists := c.KVMap[key]
	var oldValue []byte
	var expiration time.Time = time.Now().Add(time.Duration(c.defaultTTL) * time.Second)
	var persistent bool = false
	if exists && !isExpired(item) {
		oldValue = item.value
		persistent = item.persistent
		expiration = item.expiration
	}
	c.KVMap[key] = cacheItem{
		value:      value,
		expiration: expiration,
		persistent: persistent,
	}
	return oldValue, nil
}

func (c *Cache) MGet(keys []string) map[string][]byte {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	result := make(map[string][]byte)
	for _, key := range keys {
		item, exists := c.KVMap[key]
		if exists {
			if isExpired(item) {
				continue
			}
			result[key] = item.value
		}
	}
	return result
}

func (c *Cache) MSet(kv map[string][]byte, expiration time.Duration) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	for key, value := range kv {
		c.KVMap[key] = cacheItem{
			value:      value,
			expiration: time.Now().Add(expiration),
			persistent: persistent,
		}
	}
	return nil
}

func isExpired(item cacheItem) bool {
	if time.Now().After(item.expiration) && !item.persistent {
		return true
	}
	return false
}

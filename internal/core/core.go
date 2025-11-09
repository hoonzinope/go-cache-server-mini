package core

import (
	"context"
	"go-cache-server-mini/internal"
	"sync"
	"time"
)

type cacheItem struct {
	value      []byte
	expiration time.Time
}

type Cache struct {
	KVMap      *sync.Map
	defaultTTL int64
	maxTTL     int64
}

func NewCache(ctx context.Context, defaultTTL, maxTTL int64) *Cache {
	cache := &Cache{
		KVMap:      &sync.Map{},
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
			now := time.Now()
			c.KVMap.Range(func(key, value any) bool {
				if now.After(value.(cacheItem).expiration) {
					c.Del(key.(string))
				}
				return true
			})
		}
	}
}

func (c *Cache) Set(key string, value []byte, expiration time.Duration) error {
	expiration = setExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	c.KVMap.Store(key, cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
	})
	return nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	item, exists := c.KVMap.Load(key)
	if exists {
		if time.Now().After(item.(cacheItem).expiration) {
			c.Del(key)
			return nil, false
		}
		return item.(cacheItem).value, true
	}
	return nil, false
}

func (c *Cache) Del(key string) error {
	c.KVMap.Delete(key)
	return nil
}

func (c *Cache) Exists(key string) bool {
	item, exists := c.KVMap.Load(key)
	if exists {
		if time.Now().After(item.(cacheItem).expiration) {
			c.Del(key)
			return false
		}
		return true
	}
	return false
}

func (c *Cache) Keys() []string {
	var keys []string
	c.KVMap.Range(func(key, value any) bool {
		if time.Now().Before(value.(cacheItem).expiration) {
			keys = append(keys, key.(string))
		} else {
			c.Del(key.(string))
		}
		return true
	})
	return keys
}

func (c *Cache) Flush() error {
	c.KVMap.Range(func(key, value any) bool {
		c.KVMap.Delete(key)
		return true
	})
	return nil
}

func (c *Cache) TTL(key string) (time.Duration, bool) {
	item, exists := c.KVMap.Load(key)
	if !exists {
		return 0, false
	}
	remaining := time.Until(item.(cacheItem).expiration)
	if remaining <= 0 {
		c.Del(key)
		return 0, false
	}
	return remaining, true
}

func (c *Cache) Expire(key string, expiration time.Duration) error {
	if expiration <= 0 {
		c.Del(key)
		return nil
	}

	expiration = setExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	item, exists := c.KVMap.Load(key)
	if !exists {
		return internal.ErrNotFound
	}
	c.KVMap.Store(key, cacheItem{
		value:      item.(cacheItem).value,
		expiration: time.Now().Add(expiration),
	})
	return nil
}

func setExpiration(defaultTTL, maxTTL int64, reqTTL int64) time.Duration {
	var ttl int64
	if reqTTL <= 0 {
		ttl = defaultTTL
	} else if reqTTL > maxTTL {
		ttl = maxTTL
	} else {
		ttl = reqTTL
	}
	return time.Duration(ttl) * time.Second
}

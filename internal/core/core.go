package core

import (
	"context"
	"go-cache-server-mini/internal"
	"sync"
	"time"
)

type Cache struct {
	KVMap      *sync.Map
	KExpired   *sync.Map
	FlushMutex sync.Mutex
	defaultTTL int64
	maxTTL     int64
}

func NewCache(ctx context.Context, defaultTTL, maxTTL int64) *Cache {
	cache := &Cache{
		KVMap:      &sync.Map{},
		KExpired:   &sync.Map{},
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
			c.KExpired.Range(func(key, value any) bool {
				if now.After(value.(time.Time)) {
					c.Del(key.(string))
				}
				return true
			})
		}
	}
}

func (c *Cache) Set(key string, value []byte, expiration time.Duration) error {
	c.KVMap.Store(key, value)
	expiration = setExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	c.KExpired.Store(key, time.Now().Add(expiration))
	return nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	value, ok := c.KVMap.Load(key)
	if !ok {
		return nil, false
	}
	if exp, exists := c.KExpired.Load(key); exists {
		if time.Now().After(exp.(time.Time)) {
			c.Del(key)
			return nil, false
		}
	}
	return value.([]byte), true
}

func (c *Cache) Del(key string) error {
	c.KVMap.Delete(key)
	c.KExpired.Delete(key)
	return nil
}

func (c *Cache) Exists(key string) bool {
	if exp, exists := c.KExpired.Load(key); exists {
		if time.Now().After(exp.(time.Time)) {
			c.Del(key)
			return false
		} else {
			return true
		}
	} else {
		return false
	}
}

func (c *Cache) Keys() []string {
	var keys []string
	c.KExpired.Range(func(key, value any) bool {
		if time.Now().Before(value.(time.Time)) {
			keys = append(keys, key.(string))
		} else {
			c.Del(key.(string))
		}
		return true
	})
	return keys
}

func (c *Cache) Flush() error {
	c.FlushMutex.Lock()
	defer c.FlushMutex.Unlock()
	c.KVMap = &sync.Map{}
	c.KExpired = &sync.Map{}
	return nil
}

func (c *Cache) TTL(key string) (time.Duration, bool) {
	expiration, ok := c.KExpired.Load(key)
	if !ok {
		return 0, false
	}
	remaining := time.Until(expiration.(time.Time))
	if remaining < 0 {
		return 0, false
	}
	return remaining, true
}

func (c *Cache) Expire(key string, expiration time.Duration) error {
	_, ok := c.KVMap.Load(key)
	if !ok {
		return internal.ErrNotFound
	}
	expiration = setExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	c.KExpired.Store(key, time.Now().Add(expiration))
	return nil
}

func setExpiration(defaultTTL, maxTTL int64, reqTTL int64) time.Duration {
	var ttl int64
	if reqTTL < 0 {
		ttl = defaultTTL
	} else if reqTTL > maxTTL {
		ttl = maxTTL
	} else {
		ttl = reqTTL
	}
	return time.Duration(ttl) * time.Second
}

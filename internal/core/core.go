package core

import (
	"sync"
	"time"
)

type Cache struct {
	KVMap       *sync.Map
	KExpired    *sync.Map
	KTimers     *sync.Map
	TTL_default int64
	TTL_max     int64
}

func NewCache(TTL_default, TTL_max int64) *Cache {
	return &Cache{
		KVMap:       &sync.Map{},
		KExpired:    &sync.Map{},
		KTimers:     &sync.Map{},
		TTL_default: TTL_default,
		TTL_max:     TTL_max,
	}
}

func (c *Cache) Set(key string, value []byte, expiration time.Duration) error {
	c.KVMap.Store(key, value)

	if expiration <= 0 {
		expiration = time.Duration(c.TTL_default) * time.Second
	} else if expiration > time.Duration(c.TTL_max)*time.Second {
		expiration = time.Duration(c.TTL_max) * time.Second
	}

	c.KExpired.Store(key, time.Now().Add(expiration))
	c.KTimers.Store(key, time.AfterFunc(expiration, func() {
		c.KVMap.Delete(key)
	}))
	return nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	value, ok := c.KVMap.Load(key)
	if !ok {
		return nil, false
	}
	return value.([]byte), true
}

func (c *Cache) Del(key string) error {
	c.KVMap.Delete(key)
	c.KExpired.Delete(key)
	if timer, exists := c.KTimers.Load(key); exists {
		timer.(*time.Timer).Stop()
		c.KTimers.Delete(key)
	}
	return nil
}

func (c *Cache) Exists(key string) bool {
	_, ok := c.KVMap.Load(key)
	return ok
}

func (c *Cache) Keys() []string {
	var keys []string
	c.KVMap.Range(func(key, value any) bool {
		keys = append(keys, key.(string))
		return true
	})
	return keys
}

func (c *Cache) Flush() error {
	c.KVMap = &sync.Map{}
	c.KExpired = &sync.Map{}
	c.KTimers = &sync.Map{}
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
		return nil // key does not exist
	}

	if expiration <= 0 {
		expiration = time.Duration(c.TTL_default) * time.Second
	} else if expiration > time.Duration(c.TTL_max)*time.Second {
		expiration = time.Duration(c.TTL_max) * time.Second
	}

	c.KExpired.Store(key, time.Now().Add(expiration))
	if timer, exists := c.KTimers.Load(key); exists {
		timer.(*time.Timer).Stop()
	}
	c.KTimers.Store(key, time.AfterFunc(expiration, func() {
		c.KVMap.Delete(key)
	}))
	return nil
}

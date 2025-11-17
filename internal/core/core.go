package core

import (
	"context"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/core/data"
	"go-cache-server-mini/internal/core/persistentLogger"
	"go-cache-server-mini/internal/util"
	"sync"
	"time"
)

const sampleDeleteKeyCount = 20 // randomly check 20 keys for expiration each second

type Cache struct {
	Lock             sync.RWMutex
	KVMap            map[string]data.CacheItem
	defaultTTL       int64
	maxTTL           int64
	persistentLogger *persistentLogger.PersistentLogger
	persistentType   string
}

func NewCache(ctx context.Context, config *config.Config) (*Cache, error) {

	cache := &Cache{
		KVMap:          make(map[string]data.CacheItem),
		defaultTTL:     config.TTL.Default,
		maxTTL:         config.TTL.Max,
		persistentType: config.Persistent.Type,
	}

	if config.Persistent.Type == "file" {
		cache.persistentLogger = persistentLogger.NewPersistentLogger(ctx, config)
		if err := cache.Load(); err != nil { // load existing data from SNAP/AOF files
			cache.persistentLogger.Close()
			return nil, err
		}
	}
	go cache.daemon(ctx)
	return cache, nil
}

func (c *Cache) Load() error {
	var loadErr error
	c.KVMap, loadErr = c.persistentLogger.Load(c.KVMap)
	return loadErr
}

func (c *Cache) daemon(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var snapTickerChan <-chan time.Time
	if c.persistentType == "file" {
		snapTicker := time.NewTicker(60 * time.Second)
		snapTickerChan = snapTicker.C
		defer snapTicker.Stop()
	}
	for {
		select {
		case <-ctx.Done():
			if c.persistentType == "file" {
				c.persistentLogger.Close()
			}
			return
		case <-ticker.C:
			c.Lock.Lock()
			checkCount := 0 // randomly check and delete expired keys
			for key, item := range c.KVMap {
				if isExpired(item) {
					delete(c.KVMap, key)
					c.delItemLog(key)
				}
				checkCount++
				if checkCount >= sampleDeleteKeyCount {
					break
				}
			}
			c.Lock.Unlock()
		case <-snapTickerChan: // trigger snapshot every 60 seconds
			if c.persistentLogger != nil {
				c.persistentLogger.TriggerSnap(c.KVMap, &c.Lock)
			}
		}
	}
}

func (c *Cache) Set(key string, value []byte, expiration time.Duration) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	c.KVMap[key] = data.CacheItem{
		Value:      value,
		Expiration: time.Now().Add(expiration),
		Persistent: persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.KVMap[key])
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
		return item.Value, true
	}
	return nil, false
}

func (c *Cache) Del(key string) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	delete(c.KVMap, key)
	// Write to AOF
	c.delItemLog(key)
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
	// Write to AOF
	for key := range c.KVMap {
		c.delItemLog(key)
	}
	c.KVMap = make(map[string]data.CacheItem)
	return nil
}

func (c *Cache) TTL(key string) (time.Duration, bool) {
	c.Lock.RLock()
	defer c.Lock.RUnlock()
	item, exists := c.KVMap[key]
	if !exists {
		return 0, false
	}
	if item.Persistent {
		return -1, true
	}
	remaining := time.Until(item.Expiration)
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
		// Write to AOF
		c.delItemLog(key)
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
	c.KVMap[key] = data.CacheItem{
		Value:      item.Value,
		Expiration: time.Now().Add(expiration),
		Persistent: persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.KVMap[key])
	return nil
}

func (c *Cache) Persist(key string) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	item, exists := c.KVMap[key]
	if !exists || isExpired(item) {
		return internal.ErrNotFound
	}
	c.KVMap[key] = data.CacheItem{
		Value:      item.Value,
		Expiration: time.Time{},
		Persistent: true,
	}
	// Write to AOF
	c.setItemLog(key, c.KVMap[key])
	return nil
}

func (c *Cache) Incr(key string) (int64, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	item, exists := c.KVMap[key]
	if !exists || isExpired(item) {
		return 0, internal.ErrNotFound
	}
	value, err := util.BytesToInt64(item.Value)
	if err != nil {
		return 0, err
	}
	value++
	c.KVMap[key] = data.CacheItem{
		Value:      util.Int64ToBytes(value),
		Expiration: item.Expiration,
		Persistent: item.Persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.KVMap[key])
	return value, nil
}

func (c *Cache) Decr(key string) (int64, error) {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	item, exists := c.KVMap[key]
	if !exists || isExpired(item) {
		return 0, internal.ErrNotFound
	}
	value, err := util.BytesToInt64(item.Value)
	if err != nil {
		return 0, err
	}
	value--
	c.KVMap[key] = data.CacheItem{
		Value:      util.Int64ToBytes(value),
		Expiration: item.Expiration,
		Persistent: item.Persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.KVMap[key])
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
	c.KVMap[key] = data.CacheItem{
		Value:      value,
		Expiration: time.Now().Add(expiration),
		Persistent: persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.KVMap[key])
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
		oldValue = item.Value
		persistent = item.Persistent
		expiration = item.Expiration
	}
	c.KVMap[key] = data.CacheItem{
		Value:      value,
		Expiration: expiration,
		Persistent: persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.KVMap[key])
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
			result[key] = item.Value
		}
	}
	return result
}

func (c *Cache) MSet(kv map[string][]byte, expiration time.Duration) error {
	c.Lock.Lock()
	defer c.Lock.Unlock()
	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	expirationTime := time.Now().Add(expiration)
	for key, value := range kv {
		c.KVMap[key] = data.CacheItem{
			Value:      value,
			Expiration: expirationTime,
			Persistent: persistent,
		}
		// Write to AOF
		c.setItemLog(key, c.KVMap[key])
	}
	return nil
}

func isExpired(item data.CacheItem) bool {
	if time.Now().After(item.Expiration) && !item.Persistent {
		return true
	}
	return false
}

func (c *Cache) setItemLog(key string, item data.CacheItem) {
	if c.persistentType == "file" {
		// Write to AOF
		c.persistentLogger.WriteAOF(persistentLogger.Command{
			Action: "SET",
			Key:    key,
			Item:   item,
		})
	}
}

func (c *Cache) delItemLog(key string) {
	if c.persistentType == "file" {
		// Write to AOF
		c.persistentLogger.WriteAOF(persistentLogger.Command{
			Action: "DEL",
			Key:    key,
		})
	}
}

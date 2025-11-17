package core

import (
	"context"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/core/data"
	"go-cache-server-mini/internal/core/persistentLogger"
	"go-cache-server-mini/internal/util"
	"slices"
	"sync"
	"time"
)

const shardCount = 256          // number of shards for sharded locks
const sampleDeleteKeyCount = 20 // randomly check 20 keys for expiration each second

type Cache struct {
	KVMap            map[string]data.CacheItem // for snapshot purpose
	defaultTTL       int64
	maxTTL           int64
	persistentLogger *persistentLogger.PersistentLogger
	persistentType   string
	shardedMap       [shardCount]*struct { // sharded map for concurrent access
		lock  sync.RWMutex
		kvmap map[string]data.CacheItem
	}
}

func NewCache(ctx context.Context, config *config.Config) (*Cache, error) {

	// Initialize sharded map
	shardedMap := initShardedMap()

	cache := &Cache{
		KVMap:          make(map[string]data.CacheItem),
		defaultTTL:     config.TTL.Default,
		maxTTL:         config.TTL.Max,
		persistentType: config.Persistent.Type,
		shardedMap:     shardedMap,
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

func initShardedMap() [shardCount]*struct {
	lock  sync.RWMutex
	kvmap map[string]data.CacheItem
} {
	shardedMap := [shardCount]*struct {
		lock  sync.RWMutex
		kvmap map[string]data.CacheItem
	}{}
	for i := 0; i < shardCount; i++ {
		shardedMap[i] = &struct {
			lock  sync.RWMutex
			kvmap map[string]data.CacheItem
		}{
			lock:  sync.RWMutex{},
			kvmap: make(map[string]data.CacheItem),
		}
	}
	return shardedMap
}

func (c *Cache) getShardedIndex(key string) int {
	hash := util.Fnv32aHash(key)
	return int(hash % uint32(shardCount))
}

func (c *Cache) Load() error {
	var loadErr error
	c.KVMap, loadErr = c.persistentLogger.Load(c.KVMap)
	// Load data into shardedMap
	for key, item := range c.KVMap {
		index := c.getShardedIndex(key)
		c.shardedMap[index].lock.Lock()
		c.shardedMap[index].kvmap[key] = item
		c.shardedMap[index].lock.Unlock()
	}
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
			c.expireSampling()
		case <-snapTickerChan: // trigger snapshot every 60 seconds
			if c.persistentLogger != nil {
				c.snapMap()
			}
		}
	}
}

func (c *Cache) Set(key string, value []byte, expiration time.Duration) error {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.Lock()
	defer c.shardedMap[index].lock.Unlock()

	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	c.shardedMap[index].kvmap[key] = data.CacheItem{
		Value:      value,
		Expiration: time.Now().Add(expiration),
		Persistent: persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.shardedMap[index].kvmap[key])
	return nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.RLock()
	defer c.shardedMap[index].lock.RUnlock()

	item, exists := c.shardedMap[index].kvmap[key]
	if exists {
		if isExpired(item) {
			return nil, false
		}
		return item.Value, true
	}
	return nil, false
}

func (c *Cache) Del(key string) error {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.Lock()
	defer c.shardedMap[index].lock.Unlock()
	delete(c.shardedMap[index].kvmap, key)
	// Write to AOF
	c.delItemLog(key)
	return nil
}

func (c *Cache) Exists(key string) bool {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.RLock()
	defer c.shardedMap[index].lock.RUnlock()
	item, exists := c.shardedMap[index].kvmap[key]
	if exists {
		return !isExpired(item)
	}
	return false
}

func (c *Cache) Keys() []string {
	for i := 0; i < shardCount; i++ {
		c.shardedMap[i].lock.RLock()
	}

	var keys []string
	for i := 0; i < shardCount; i++ {
		for key, item := range c.shardedMap[i].kvmap {
			if isExpired(item) {
				continue
			}
			keys = append(keys, key)
		}
	}

	for i := shardCount - 1; i >= 0; i-- {
		c.shardedMap[i].lock.RUnlock()
	}

	return keys
}

func (c *Cache) Flush() error {
	for i := 0; i < shardCount; i++ {
		c.shardedMap[i].lock.Lock()
	}
	for i := 0; i < shardCount; i++ {
		for key := range c.shardedMap[i].kvmap {
			// Write to AOF
			c.delItemLog(key)
		}
		c.shardedMap[i].kvmap = make(map[string]data.CacheItem)
	}
	for i := shardCount - 1; i >= 0; i-- {
		c.shardedMap[i].lock.Unlock()
	}
	return nil
}

func (c *Cache) TTL(key string) (time.Duration, bool) {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.RLock()
	defer c.shardedMap[index].lock.RUnlock()
	item, exists := c.shardedMap[index].kvmap[key]
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
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.Lock()
	defer c.shardedMap[index].lock.Unlock()
	if expiration <= 0 {
		delete(c.shardedMap[index].kvmap, key)
		// Write to AOF
		c.delItemLog(key)
		return nil
	}

	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	item, exists := c.shardedMap[index].kvmap[key]
	if !exists {
		return internal.ErrNotFound
	}
	if isExpired(item) {
		return internal.ErrNotFound
	}
	c.shardedMap[index].kvmap[key] = data.CacheItem{
		Value:      item.Value,
		Expiration: time.Now().Add(expiration),
		Persistent: persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.shardedMap[index].kvmap[key])
	return nil
}

func (c *Cache) Persist(key string) error {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.Lock()
	defer c.shardedMap[index].lock.Unlock()
	item, exists := c.shardedMap[index].kvmap[key]
	if !exists || isExpired(item) {
		return internal.ErrNotFound
	}
	c.shardedMap[index].kvmap[key] = data.CacheItem{
		Value:      item.Value,
		Expiration: time.Time{},
		Persistent: true,
	}
	// Write to AOF
	c.setItemLog(key, c.shardedMap[index].kvmap[key])
	return nil
}

func (c *Cache) Incr(key string) (int64, error) {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.Lock()
	defer c.shardedMap[index].lock.Unlock()
	item, exists := c.shardedMap[index].kvmap[key]
	if !exists || isExpired(item) {
		return 0, internal.ErrNotFound
	}
	value, err := util.BytesToInt64(item.Value)
	if err != nil {
		return 0, err
	}
	value++
	c.shardedMap[index].kvmap[key] = data.CacheItem{
		Value:      util.Int64ToBytes(value),
		Expiration: item.Expiration,
		Persistent: item.Persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.shardedMap[index].kvmap[key])
	return value, nil
}

func (c *Cache) Decr(key string) (int64, error) {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.Lock()
	defer c.shardedMap[index].lock.Unlock()
	item, exists := c.shardedMap[index].kvmap[key]
	if !exists || isExpired(item) {
		return 0, internal.ErrNotFound
	}
	value, err := util.BytesToInt64(item.Value)
	if err != nil {
		return 0, err
	}
	value--
	c.shardedMap[index].kvmap[key] = data.CacheItem{
		Value:      util.Int64ToBytes(value),
		Expiration: item.Expiration,
		Persistent: item.Persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.shardedMap[index].kvmap[key])
	return value, nil
}

func (c *Cache) SetNX(key string, value []byte, expiration time.Duration) (bool, error) {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.Lock()
	defer c.shardedMap[index].lock.Unlock()
	item, exists := c.shardedMap[index].kvmap[key]
	if exists && !isExpired(item) {
		return false, nil
	}
	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	c.shardedMap[index].kvmap[key] = data.CacheItem{
		Value:      value,
		Expiration: time.Now().Add(expiration),
		Persistent: persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.shardedMap[index].kvmap[key])
	return true, nil
}

func (c *Cache) GetSet(key string, value []byte) ([]byte, error) {
	index := c.getShardedIndex(key)
	c.shardedMap[index].lock.Lock()
	defer c.shardedMap[index].lock.Unlock()
	item, exists := c.shardedMap[index].kvmap[key]
	var oldValue []byte
	var expiration time.Time = time.Now().Add(time.Duration(c.defaultTTL) * time.Second)
	var persistent bool = false
	if exists && !isExpired(item) {
		oldValue = item.Value
		persistent = item.Persistent
		expiration = item.Expiration
	}
	c.shardedMap[index].kvmap[key] = data.CacheItem{
		Value:      value,
		Expiration: expiration,
		Persistent: persistent,
	}
	// Write to AOF
	c.setItemLog(key, c.shardedMap[index].kvmap[key])
	return oldValue, nil
}

func (c *Cache) MGet(keys []string) map[string][]byte {
	indexList := make([]int, 0, len(keys))
	for _, key := range keys {
		indexList = append(indexList, c.getShardedIndex(key))
	}
	indexList = slices.Compact(indexList)
	slices.Sort(indexList)
	for _, index := range indexList {
		c.shardedMap[index].lock.RLock()
	}
	result := make(map[string][]byte)
	for _, key := range keys {
		index := c.getShardedIndex(key)
		item, exists := c.shardedMap[index].kvmap[key]
		if exists && !isExpired(item) {
			result[key] = item.Value
		}
	}
	for j := len(indexList) - 1; j >= 0; j-- {
		c.shardedMap[indexList[j]].lock.RUnlock()
	}
	return result
}

func (c *Cache) MSet(kv map[string][]byte, expiration time.Duration) error {
	indexList := make([]int, 0, len(kv))
	for key := range kv {
		indexList = append(indexList, c.getShardedIndex(key))
	}
	indexList = slices.Compact(indexList)
	slices.Sort(indexList)
	for _, index := range indexList {
		c.shardedMap[index].lock.Lock()
	}
	expiration, persistent := util.SetExpiration(c.defaultTTL, c.maxTTL, int64(expiration.Seconds()))
	expirationTime := time.Now().Add(expiration)
	for key, value := range kv {
		index := c.getShardedIndex(key)
		c.shardedMap[index].kvmap[key] = data.CacheItem{
			Value:      value,
			Expiration: expirationTime,
			Persistent: persistent,
		}
		// Write to AOF
		c.setItemLog(key, c.shardedMap[index].kvmap[key])
	}
	for j := len(indexList) - 1; j >= 0; j-- {
		c.shardedMap[indexList[j]].lock.Unlock()
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

func (c *Cache) expireSampling() {
	indexList := util.GetRandomShardIndex(shardCount, sampleDeleteKeyCount)
	for _, index := range indexList {
		c.shardedMap[index].lock.Lock()
	}
	checkCount := 0
	breakFlag := false
	for _, i := range indexList {
		for key, item := range c.shardedMap[i].kvmap {
			if isExpired(item) {
				delete(c.shardedMap[i].kvmap, key)
				c.delItemLog(key)
			}
			checkCount++
			if checkCount >= sampleDeleteKeyCount {
				breakFlag = true
				break
			}
		}
		if breakFlag {
			break
		}
	}
	for j := len(indexList) - 1; j >= 0; j-- {
		c.shardedMap[indexList[j]].lock.Unlock()
	}
}

func (c *Cache) snapMap() {
	for i := 0; i < shardCount; i++ {
		c.shardedMap[i].lock.Lock()
	}
	c.KVMap = make(map[string]data.CacheItem)
	for i := 0; i < shardCount; i++ {
		for key, item := range c.shardedMap[i].kvmap {
			c.KVMap[key] = item
		}
	}
	for i := shardCount - 1; i >= 0; i-- {
		c.shardedMap[i].lock.Unlock()
	}
	c.persistentLogger.TriggerSnap(c.KVMap)
}

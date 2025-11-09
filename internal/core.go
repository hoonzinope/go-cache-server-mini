package internal

import (
	"sync"
)

type Cache struct {
	KVMap *sync.Map
}

func NewCache() *Cache {
	return &Cache{
		KVMap: &sync.Map{},
	}
}

func (c *Cache) Set(key string, value []byte) error {
	c.KVMap.Store(key, value)
	return nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	value, ok := c.KVMap.Load(key)
	if !ok {
		return nil, false
	}
	return value.([]byte), true
}

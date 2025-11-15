package data

import "time"

type CacheItem struct {
	Value      []byte
	Expiration time.Time
	Persistent bool
}

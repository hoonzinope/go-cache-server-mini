package core

import "time"

type CacheInterface interface {
	Set(key string, value []byte, expiration time.Duration) error           // expiration of -1 means no expiration
	Get(key string) ([]byte, bool)                                          // returns value and whether the key exists
	Del(key string) error                                                   // deletes a key
	Exists(key string) bool                                                 // checks if a key exists
	Keys() []string                                                         // returns all keys
	Flush() error                                                           // clears the cache
	TTL(key string) (time.Duration, bool)                                   // returns remaining TTL and whether the key exists
	Expire(key string, expiration time.Duration) error                      // updates the TTL of a key
	Persist(key string) error                                               // removes the expiration from a key
	Incr(key string) (int64, error)                                         // increments an integer value plus one
	Decr(key string) (int64, error)                                         // decrements an integer value minus one
	SetNX(key string, value []byte, expiration time.Duration) (bool, error) // sets the value only if the key does not exist
	GetSet(key string, value []byte) ([]byte, error)                        // sets a new value and returns the old value
	MGet(keys []string) map[string][]byte                                   // retrieves multiple keys at once
	MSet(kv map[string][]byte, expiration time.Duration) error              // sets multiple key-value pairs at once
}

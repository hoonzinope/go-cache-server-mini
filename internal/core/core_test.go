package core

import (
	"context"
	"os"
	"testing"
	"time"

	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/config"
)

func newTestCache(t *testing.T) *Cache {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	config := config.LoadTestConfig()
	t.Cleanup(func() {
		os.RemoveAll(config.Persistent.Path)
		cancel()
	})
	cache, err := NewCache(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	return cache
}

func TestCacheBasicOperations(t *testing.T) {
	cache := newTestCache(t)

	if err := cache.Set("foo", []byte("bar"), 5*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	value, ok := cache.Get("foo")
	if !ok || string(value) != "bar" {
		t.Fatalf("Get returned unexpected result, ok=%v value=%s", ok, value)
	}

	if err := cache.Del("foo"); err != nil {
		t.Fatalf("Del returned error: %v", err)
	}

	if _, ok := cache.Get("foo"); ok {
		t.Fatalf("expected key to be deleted")
	}
}

func TestCacheExistsKeysAndFlush(t *testing.T) {
	cache := newTestCache(t)

	if cache.Exists("missing") {
		t.Fatalf("missing key should not exist")
	}

	if err := cache.Set("a", []byte("1"), 5*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := cache.Set("b", []byte("2"), 5*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	if !cache.Exists("a") || !cache.Exists("b") {
		t.Fatalf("expected keys to exist")
	}

	keys := cache.Keys()
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}

	if err := cache.Flush(); err != nil {
		t.Fatalf("Flush returned error: %v", err)
	}

	if cache.Exists("a") || cache.Exists("b") {
		t.Fatalf("keys should be removed after flush")
	}
}

func TestCacheTTLExpireAndPersist(t *testing.T) {
	cache := newTestCache(t)
	if err := cache.Set("ttl", []byte("value"), 2*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	ttl, ok := cache.TTL("ttl")
	if !ok || ttl <= 0 {
		t.Fatalf("TTL should return positive value, got %v (ok=%v)", ttl, ok)
	}

	if err := cache.Expire("ttl", 5*time.Second); err != nil {
		t.Fatalf("Expire returned error: %v", err)
	}

	ttl, ok = cache.TTL("ttl")
	if !ok || ttl <= 0 || ttl > 5*time.Second {
		t.Fatalf("Expire should update TTL close to requested duration, got %v", ttl)
	}

	if err := cache.Persist("ttl"); err != nil {
		t.Fatalf("Persist returned error: %v", err)
	}

	ttl, ok = cache.TTL("ttl")
	if !ok || ttl != -1 {
		t.Fatalf("Persisted key should have ttl -1, got %v", ttl)
	}

	if err := cache.Del("ttl"); err != nil {
		t.Fatalf("Del returned error: %v", err)
	}
	if err := cache.Expire("ttl", time.Second); err != internal.ErrNotFound {
		t.Fatalf("Expire should return ErrNotFound for missing key, got %v", err)
	}
}

func TestCacheIncrDecr(t *testing.T) {
	cache := newTestCache(t)

	if err := cache.Set("counter", []byte("1"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	if value, err := cache.Incr("counter"); err != nil || value != 2 {
		t.Fatalf("Incr expected 2, got value=%d err=%v", value, err)
	}

	if value, err := cache.Decr("counter"); err != nil || value != 1 {
		t.Fatalf("Decr expected 1, got value=%d err=%v", value, err)
	}

	if _, err := cache.Incr("missing"); err != internal.ErrNotFound {
		t.Fatalf("Incr should fail with ErrNotFound for missing key, got %v", err)
	}
}

func TestCacheSetNX(t *testing.T) {
	cache := newTestCache(t)

	ok, err := cache.SetNX("nx", []byte("first"), 0)
	if err != nil || !ok {
		t.Fatalf("SetNX first call expected true, got ok=%v err=%v", ok, err)
	}

	ok, err = cache.SetNX("nx", []byte("second"), 0)
	if err != nil {
		t.Fatalf("SetNX second call returned error: %v", err)
	}
	if ok {
		t.Fatalf("SetNX should return false when key exists")
	}

	val, exists := cache.Get("nx")
	if !exists || string(val) != "first" {
		t.Fatalf("SetNX should not overwrite existing value, got %s", val)
	}
}

func TestCacheMGetMSet(t *testing.T) {
	cache := newTestCache(t)
	payload := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
	}

	if err := cache.MSet(payload, 0); err != nil {
		t.Fatalf("MSet returned error: %v", err)
	}

	result := cache.MGet([]string{"a", "b", "c"})
	if len(result) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(result))
	}
	if string(result["a"]) != "1" || string(result["b"]) != "2" {
		t.Fatalf("unexpected MGet results: %v", result)
	}
	if _, exists := result["c"]; exists {
		t.Fatalf("missing key should not be present in MGet result")
	}
}

func TestCacheGetSetPreservesTTL(t *testing.T) {
	cache := newTestCache(t)

	if err := cache.Set("ttl-key", []byte("v1"), 2*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	before, ok := cache.TTL("ttl-key")
	if !ok {
		t.Fatalf("TTL reported missing key before GetSet")
	}
	if before <= 0 || before > 2*time.Second {
		t.Fatalf("unexpected TTL before GetSet: %v", before)
	}

	if _, err := cache.GetSet("ttl-key", []byte("v2")); err != nil {
		t.Fatalf("GetSet returned error: %v", err)
	}

	after, ok := cache.TTL("ttl-key")
	if !ok {
		t.Fatalf("TTL reported missing key after GetSet")
	}
	if after <= 0 {
		t.Fatalf("TTL after GetSet should remain positive, got %v", after)
	}
	if after > before {
		t.Fatalf("TTL increased after GetSet, expected to preserve expiration (before=%v after=%v)", before, after)
	}
}

func TestCacheGetSetRespectsPersistence(t *testing.T) {
	cache := newTestCache(t)

	if err := cache.Set("persist-key", []byte("v1"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := cache.Persist("persist-key"); err != nil {
		t.Fatalf("Persist returned error: %v", err)
	}

	if _, err := cache.GetSet("persist-key", []byte("v2")); err != nil {
		t.Fatalf("GetSet returned error: %v", err)
	}

	ttl, ok := cache.TTL("persist-key")
	if !ok {
		t.Fatalf("TTL reported missing persistent key")
	}
	if ttl != -1 {
		t.Fatalf("persistent key TTL should be -1, got %v", ttl)
	}
}

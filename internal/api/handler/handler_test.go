package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"go-cache-server-mini/internal/config"
	"go-cache-server-mini/internal/core"
	"go-cache-server-mini/internal/distributed/adapter"
	"go-cache-server-mini/internal/distributed/router"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newHandlerTestCache(t *testing.T) router.DistributorInterface {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	config := config.LoadTestConfig()
	t.Cleanup(func() {
		os.RemoveAll(config.Persistent.Path)
		cancel()
	})
	core, err := core.NewCache(ctx, config)
	localAdapter := adapter.NewLocalAdapter(core)
	nodeRouter := router.NewNodeRouter(ctx, localAdapter)
	cacheDistributor := router.NewDistributor(nodeRouter)
	if err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}
	return cacheDistributor
}

func newTestContext(method, target string, body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	return c, w
}

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	return data
}

func TestSetHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	handler := SetHandler{Cache: cache}

	body := mustJSON(t, map[string]any{"key": "foo", "value": "bar", "ttl": 1})
	c, w := newTestContext(http.MethodPost, "/set", body)
	handler.Set(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	_, ok, err := cache.Get("foo")
	if !ok {
		t.Fatalf("expected key to be set in cache: %v", err)
	}
}

func TestGetHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("foo", []byte(`"bar"`), 5*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := GetHandler{Cache: cache}
	c, w := newTestContext(http.MethodGet, "/get?key=foo", nil)
	handler.Get(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	var actual string
	if err := json.Unmarshal(resp.Value, &actual); err != nil {
		t.Fatalf("failed to parse value: %v", err)
	}
	if actual != "bar" {
		t.Fatalf("expected value bar, got %s", actual)
	}
}

func TestDelHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("foo", []byte("1"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := DelHandler{Cache: cache}
	c, w := newTestContext(http.MethodDelete, "/del?key=foo", nil)
	handler.Del(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	exists, err := cache.Exists("foo")
	if exists || err != nil {
		t.Fatalf("expected key to be deleted")
	}
}

func TestExistsHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("foo", []byte("1"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := ExistsHandler{Cache: cache}
	c, w := newTestContext(http.MethodGet, "/exists?key=foo", nil)
	handler.Exists(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Exists bool `json:"exists"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if !resp.Exists {
		t.Fatalf("expected exists=true")
	}
}

func TestKeysHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("a", []byte("1"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := cache.Set("b", []byte("2"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := KeysHandler{Cache: cache}
	c, w := newTestContext(http.MethodGet, "/keys", nil)
	handler.Keys(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Keys []string `json:"keys"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	slices.Sort(resp.Keys)
	expected := []string{"a", "b"}
	if !slices.Equal(resp.Keys, expected) {
		t.Fatalf("expected keys %v, got %v", expected, resp.Keys)
	}
}

func TestFlushHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("a", []byte("1"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := FlushHandler{Cache: cache}
	c, w := newTestContext(http.MethodPost, "/flush", nil)
	handler.Flush(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	keys, err := cache.Keys()
	if err != nil {
		t.Fatalf("Keys returned error: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected cache to be empty after flush")
	}
}

func TestExpireHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("ttl", []byte("1"), 5*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := ExpireHandler{Cache: cache}
	body := mustJSON(t, map[string]any{"key": "ttl", "ttl": 1})
	c, w := newTestContext(http.MethodPost, "/expire", body)
	handler.Expire(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	ttl, ok, err := cache.TTL("ttl")
	if err != nil {
		t.Fatalf("TTL returned error: %v", err)
	}
	if !ok || ttl <= 0 || ttl > time.Second {
		t.Fatalf("expected ttl to be updated to ~1s, got %v", ttl)
	}
}

func TestTTLHandlerReturnsPersistentTTL(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("persist-key", []byte("1"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := cache.Persist("persist-key"); err != nil {
		t.Fatalf("Persist returned error: %v", err)
	}

	handler := TTLHandler{Cache: cache}
	c, w := newTestContext(http.MethodGet, "/ttl?key=persist-key", nil)
	handler.TTL(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp["ttl"] != -1 {
		t.Fatalf("expected ttl -1 for persistent key, got %v", resp["ttl"])
	}
}

func TestPersistHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("persist", []byte("1"), 5*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := PersistHandler{Cache: cache}
	c, w := newTestContext(http.MethodPost, "/persist?key=persist", nil)
	handler.Persist(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	ttl, ok, err := cache.TTL("persist")
	if err != nil {
		t.Fatalf("TTL returned error: %v", err)
	}
	if !ok || ttl != -1 {
		t.Fatalf("expected ttl -1 after persist, got %v", ttl)
	}
}

func TestIncrHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("count", []byte("1"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := IncrHandler{Cache: cache}
	c, w := newTestContext(http.MethodPost, "/incr?key=count", nil)
	handler.Incr(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	var actual int
	if err := json.Unmarshal(resp.Value, &actual); err != nil {
		t.Fatalf("failed to parse value: %v", err)
	}
	if actual != 2 {
		t.Fatalf("expected value 2, got %d", actual)
	}
}

func TestDecrHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("count", []byte("2"), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := DecrHandler{Cache: cache}
	c, w := newTestContext(http.MethodPost, "/decr?key=count", nil)
	handler.Decr(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	var actual int
	if err := json.Unmarshal(resp.Value, &actual); err != nil {
		t.Fatalf("failed to parse value: %v", err)
	}
	if actual != 1 {
		t.Fatalf("expected value 1, got %d", actual)
	}
}

func TestSetNXHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	handler := SetNXHandler{Cache: cache}

	first := mustJSON(t, map[string]any{"key": "nx", "value": 1})
	c, w := newTestContext(http.MethodPost, "/setnx", first)
	handler.SetNX(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	var resp map[string]bool
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp["success"] {
		t.Fatalf("expected success true on first setnx")
	}

	second := mustJSON(t, map[string]any{"key": "nx", "value": 2})
	c, w = newTestContext(http.MethodPost, "/setnx", second)
	handler.SetNX(c)
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["success"] {
		t.Fatalf("expected success false when key exists")
	}
}

func TestGetSetHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("swap", []byte(`"old"`), 0); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := GetSetHandler{Cache: cache}
	body := mustJSON(t, map[string]any{"key": "swap", "value": "new"})
	c, w := newTestContext(http.MethodPost, "/getset", body)
	handler.GetSet(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		Value json.RawMessage `json:"value"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	var old string
	json.Unmarshal(resp.Value, &old)
	if old != "old" {
		t.Fatalf("expected old value old, got %s", old)
	}

	value, _, err := cache.Get("swap")
	if err != nil {
		t.Fatalf("failed to get value from cache: %v", err)
	}
	if string(value) != `"new"` {
		t.Fatalf("expected cache to hold new value, got %s", value)
	}
}

func TestMGetHandlerReturnsValues(t *testing.T) {
	cache := newHandlerTestCache(t)
	if err := cache.Set("a", []byte("1"), 5*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}
	if err := cache.Set("b", []byte("2"), 5*time.Second); err != nil {
		t.Fatalf("Set returned error: %v", err)
	}

	handler := MGetHandler{Cache: cache}
	body := mustJSON(t, map[string]any{"keys": []string{"a", "b", "missing"}})
	c, w := newTestContext(http.MethodPost, "/mget", body)
	handler.MGet(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp struct {
		KV map[string]json.RawMessage `json:"kv"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.KV) != 2 {
		t.Fatalf("expected 2 keys in response, got %d", len(resp.KV))
	}
	if string(resp.KV["a"]) != "1" || string(resp.KV["b"]) != "2" {
		t.Fatalf("unexpected kv payload: %v", resp.KV)
	}
}

func TestMSetHandler(t *testing.T) {
	cache := newHandlerTestCache(t)
	handler := MSetHandler{Cache: cache}

	body := mustJSON(t, map[string]any{
		"kv": map[string]any{
			"a": "1",
			"b": "2",
		},
		"ttl": 2,
	})
	c, w := newTestContext(http.MethodPost, "/mset", body)
	handler.MSet(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if value, ok, err := cache.Get("a"); !ok || err != nil || string(value) != `"1"` {
		t.Fatalf("expected key a to be set, got %s ok=%v err=%v", value, ok, err)
	}
	if value, ok, err := cache.Get("b"); !ok || err != nil || string(value) != `"2"` {
		t.Fatalf("expected key b to be set, got %s ok=%v err=%v", value, ok, err)
	}
}

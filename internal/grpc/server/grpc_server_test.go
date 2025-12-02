package server

import (
	"context"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"go-cache-server-mini/internal/distributed/router"
	"go-cache-server-mini/internal/grpc/client"
	"go-cache-server-mini/internal/grpc/pb"

	"google.golang.org/grpc"
)

func TestGRPCCacheServer_SetGetDel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stubDistributor := newStubDistributor()
	grpcServer := grpc.NewServer()
	pb.RegisterCacheServiceServer(grpcServer, NewGRPCCacheServer(stubDistributor))

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	go func() {
		if serveErr := grpcServer.Serve(lis); serveErr != nil {
			t.Logf("grpc server exited: %v", serveErr)
		}
	}()
	defer grpcServer.Stop()

	host, portStr, err := net.SplitHostPort(lis.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host/port: %v", err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatalf("failed to parse port: %v", err)
	}

	cacheClient, err := client.NewGRPCCacheClient(host, port)
	if err != nil {
		t.Fatalf("failed to create grpc client: %v", err)
	}
	defer cacheClient.Close()

	// set
	if err := cacheClient.Set(ctx, "foo", []byte("bar"), int64(time.Second.Seconds())); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// get hit
	value, found, err := cacheClient.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatalf("expected key to be found")
	}
	if string(value) != "bar" {
		t.Fatalf("expected value 'bar', got %q", string(value))
	}

	// del
	if err := cacheClient.Del(ctx, "foo"); err != nil {
		t.Fatalf("Del failed: %v", err)
	}

	// get miss
	_, found, err = cacheClient.Get(ctx, "foo")
	if err != nil {
		t.Fatalf("Get after delete returned error: %v", err)
	}
	if found {
		t.Fatalf("expected key to be missing after delete")
	}
}

// stubDistributor implements router.DistributorInterface for tests.
type stubDistributor struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newStubDistributor() router.DistributorInterface {
	return &stubDistributor{
		m: make(map[string][]byte),
	}
}

func (d *stubDistributor) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.m[key] = value
	return nil
}

func (d *stubDistributor) Get(ctx context.Context, key string) ([]byte, bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	val, ok := d.m[key]
	return val, ok, nil
}

func (d *stubDistributor) Del(ctx context.Context, key string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.m, key)
	return nil
}

func (d *stubDistributor) Exists(ctx context.Context, key string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.m[key]
	return ok, nil
}

func (d *stubDistributor) Keys(ctx context.Context) ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	keys := make([]string, 0, len(d.m))
	for k := range d.m {
		keys = append(keys, k)
	}
	return keys, nil
}

func (d *stubDistributor) Flush(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.m = make(map[string][]byte)
	return nil
}

func (d *stubDistributor) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	return 0, false, nil
}

func (d *stubDistributor) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}

func (d *stubDistributor) Persist(ctx context.Context, key string) error {
	return nil
}

func (d *stubDistributor) Incr(ctx context.Context, key string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	val, ok := d.m[key]
	var num int64
	if ok {
		num = int64(len(val))
	}
	num++
	d.m[key] = []byte(strconv.FormatInt(num, 10))
	return num, nil
}

func (d *stubDistributor) Decr(ctx context.Context, key string) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	val, ok := d.m[key]
	var num int64
	if ok {
		num = int64(len(val))
	}
	num--
	d.m[key] = []byte(strconv.FormatInt(num, 10))
	return num, nil
}

func (d *stubDistributor) SetNX(ctx context.Context, key string, value []byte, expiration time.Duration) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.m[key]; ok {
		return false, nil
	}
	d.m[key] = value
	return true, nil
}

func (d *stubDistributor) GetSet(ctx context.Context, key string, value []byte) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	old := d.m[key]
	d.m[key] = value
	return old, nil
}

func (d *stubDistributor) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	result := make(map[string][]byte, len(keys))
	for _, k := range keys {
		if val, ok := d.m[k]; ok {
			result[k] = val
		}
	}
	return result, nil
}

func (d *stubDistributor) MSet(ctx context.Context, kv map[string][]byte, expiration time.Duration) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for k, v := range kv {
		d.m[k] = v
	}
	return nil
}

// Ensure stubDistributor implements the interface at compile time.
var _ router.DistributorInterface = (*stubDistributor)(nil)

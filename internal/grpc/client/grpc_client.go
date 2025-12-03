package client

import (
	"context"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/grpc/pb"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type GRPCCacheClient struct {
	pb.CacheServiceClient
	remote_node_ip string
	conn           *grpc.ClientConn
}

func NewGRPCCacheClient(remote_node_ip string, port int) (*GRPCCacheClient, error) {
	url := remote_node_ip + ":" + strconv.Itoa(port)
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCCacheClient{
		CacheServiceClient: pb.NewCacheServiceClient(conn),
		remote_node_ip:     remote_node_ip,
		conn:               conn,
	}, nil
}

func (c *GRPCCacheClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCCacheClient) Get(ctx context.Context, key string) ([]byte, bool, error) {
	req := &pb.GetRequest{
		Key: key,
	}
	res, err := c.CacheServiceClient.Get(ctx, req)
	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok && statusErr.Code() == codes.NotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	if res == nil {
		return nil, false, nil
	}
	return res.Value, true, nil
}

func (c *GRPCCacheClient) Set(ctx context.Context, key string, value []byte, expiration int64) error {
	req := &pb.SetRequest{
		Key:   key,
		Value: value,
		Ttl:   expiration,
	}
	_, err := c.CacheServiceClient.Set(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *GRPCCacheClient) Del(ctx context.Context, key string) error {
	req := &pb.DelRequest{
		Key: key,
	}
	_, err := c.CacheServiceClient.Del(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *GRPCCacheClient) Exists(ctx context.Context, key string) (bool, error) {
	req := &pb.ExistsRequest{Key: key}
	res, err := c.CacheServiceClient.Exists(ctx, req)
	if err != nil {
		return false, err
	}
	return res.GetExists(), nil
}

func (c *GRPCCacheClient) Keys(ctx context.Context) ([]string, error) {
	res, err := c.CacheServiceClient.Keys(ctx, &pb.KeysRequest{})
	if err != nil {
		return nil, err
	}
	return res.GetKeys(), nil
}

func (c *GRPCCacheClient) Flush(ctx context.Context) error {
	_, err := c.CacheServiceClient.Flush(ctx, &pb.FlushRequest{})
	return err
}

func (c *GRPCCacheClient) TTL(ctx context.Context, key string) (time.Duration, bool, error) {
	res, err := c.CacheServiceClient.TTL(ctx, &pb.TTLRequest{Key: key})
	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok && statusErr.Code() == codes.NotFound {
			return 0, false, nil
		}
		return 0, false, err
	}
	if res.GetTtlSeconds() < 0 {
		return -1, true, nil
	}
	return time.Duration(res.GetTtlSeconds()) * time.Second, res.GetFound(), nil
}

func (c *GRPCCacheClient) Expire(ctx context.Context, key string, ttlSeconds int64) error {
	_, err := c.CacheServiceClient.Expire(ctx, &pb.ExpireRequest{Key: key, Ttl: ttlSeconds})
	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok && statusErr.Code() == codes.NotFound {
			return internal.ErrNotFound
		}
		return err
	}
	return nil
}

func (c *GRPCCacheClient) Persist(ctx context.Context, key string) error {
	_, err := c.CacheServiceClient.Persist(ctx, &pb.PersistRequest{Key: key})
	if err != nil {
		statusErr, ok := status.FromError(err)
		if ok && statusErr.Code() == codes.NotFound {
			return internal.ErrNotFound
		}
		return err
	}
	return nil
}

func (c *GRPCCacheClient) Incr(ctx context.Context, key string) (int64, error) {
	res, err := c.CacheServiceClient.Incr(ctx, &pb.IncrRequest{Key: key})
	if err != nil {
		return 0, err
	}
	return res.GetValue(), nil
}

func (c *GRPCCacheClient) Decr(ctx context.Context, key string) (int64, error) {
	res, err := c.CacheServiceClient.Decr(ctx, &pb.DecrRequest{Key: key})
	if err != nil {
		return 0, err
	}
	return res.GetValue(), nil
}

func (c *GRPCCacheClient) SetNX(ctx context.Context, key string, value []byte, ttlSeconds int64) (bool, error) {
	res, err := c.CacheServiceClient.SetNX(ctx, &pb.SetNXRequest{
		Key:   key,
		Value: value,
		Ttl:   ttlSeconds,
	})
	if err != nil {
		return false, err
	}
	return res.GetSuccess(), nil
}

func (c *GRPCCacheClient) GetSet(ctx context.Context, key string, value []byte) ([]byte, bool, error) {
	res, err := c.CacheServiceClient.GetSet(ctx, &pb.GetSetRequest{
		Key:   key,
		Value: value,
	})
	if err != nil {
		return nil, false, err
	}
	return res.GetOldValue(), res.GetFound(), nil
}

func (c *GRPCCacheClient) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	res, err := c.CacheServiceClient.MGet(ctx, &pb.MGetRequest{Keys: keys})
	if err != nil {
		return nil, err
	}
	return res.GetKv(), nil
}

func (c *GRPCCacheClient) MSet(ctx context.Context, kv map[string][]byte, ttlSeconds int64) error {
	_, err := c.CacheServiceClient.MSet(ctx, &pb.MSetRequest{
		Kv:  kv,
		Ttl: ttlSeconds,
	})
	return err
}

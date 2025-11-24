package client

import (
	"context"
	"go-cache-server-mini/internal/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCCacheClient struct {
	pb.CacheServiceClient
	remote_node_ip string
	ctx            context.Context
	conn           *grpc.ClientConn
}

func NewGRPCCacheClient(ctx context.Context, remote_node_ip string) (*GRPCCacheClient, error) {
	url := remote_node_ip + ":50051"
	conn, err := grpc.NewClient(url, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &GRPCCacheClient{
		CacheServiceClient: pb.NewCacheServiceClient(conn),
		remote_node_ip:     remote_node_ip,
		ctx:                ctx,
		conn:               conn,
	}, nil
}

func (c *GRPCCacheClient) Close() error {
	return c.conn.Close()
}

func (c *GRPCCacheClient) Get(key string) ([]byte, bool, error) {
	req := &pb.GetRequest{
		Key: key,
	}
	res, err := c.CacheServiceClient.Get(c.ctx, req)
	if err != nil {
		return nil, false, err
	}
	if res == nil {
		return nil, false, nil
	}
	return res.Value, true, nil
}

func (c *GRPCCacheClient) Set(key string, value []byte, expiration int64) error {
	req := &pb.SetRequest{
		Key:   key,
		Value: value,
		Ttl:   expiration,
	}
	_, err := c.CacheServiceClient.Set(c.ctx, req)
	if err != nil {
		return err
	}
	return nil
}

func (c *GRPCCacheClient) Del(key string) error {
	req := &pb.DelRequest{
		Key: key,
	}
	_, err := c.CacheServiceClient.Del(c.ctx, req)
	if err != nil {
		return err
	}
	return nil
}

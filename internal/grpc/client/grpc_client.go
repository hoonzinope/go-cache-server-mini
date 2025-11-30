package client

import (
	"context"
	"go-cache-server-mini/internal/grpc/pb"
	"strconv"

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

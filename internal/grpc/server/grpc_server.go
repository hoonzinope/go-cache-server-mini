package server

import (
	"context"
	"errors"
	"go-cache-server-mini/internal"
	"go-cache-server-mini/internal/distributed/router"
	"go-cache-server-mini/internal/grpc/pb"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// another node input cache ops to local node via gRPC

type GRPCCacheServer struct {
	pb.UnimplementedCacheServiceServer
	distributor router.DistributorInterface
}

func StartGRPCCacheServer(ctx context.Context, addr string, distributor router.DistributorInterface) error {

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer()
	grpcCacheServer := NewGRPCCacheServer(distributor)
	pb.RegisterCacheServiceServer(grpcServer, grpcCacheServer)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	return grpcServer.Serve(lis)
}

func NewGRPCCacheServer(distributor router.DistributorInterface) *GRPCCacheServer {
	return &GRPCCacheServer{
		distributor: distributor,
	}
}

func (s *GRPCCacheServer) Get(ctx context.Context, req *pb.GetRequest) (res *pb.GetResponse, err error) {
	value, found, err := s.distributor.Get(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if !found {
		return &pb.GetResponse{}, status.Error(codes.NotFound, "key not found")
	}
	resp := &pb.GetResponse{
		Value: value,
	}
	return resp, nil
}

func (s *GRPCCacheServer) Set(ctx context.Context, req *pb.SetRequest) (res *pb.SetResponse, err error) {
	err = s.distributor.Set(ctx, req.Key, req.Value, time.Duration(req.Ttl)*time.Second)
	if err != nil {
		return nil, err
	}
	return &pb.SetResponse{
		Status: true,
	}, nil
}

func (s *GRPCCacheServer) Del(ctx context.Context, req *pb.DelRequest) (res *pb.DelResponse, err error) {
	err = s.distributor.Del(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	return &pb.DelResponse{
		Status: true,
	}, nil
}

func (s *GRPCCacheServer) Exists(ctx context.Context, req *pb.ExistsRequest) (res *pb.ExistsResponse, err error) {
	exists, err := s.distributor.Exists(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	return &pb.ExistsResponse{Exists: exists}, nil
}

func (s *GRPCCacheServer) Keys(ctx context.Context, _ *pb.KeysRequest) (res *pb.KeysResponse, err error) {
	keys, err := s.distributor.Keys(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.KeysResponse{Keys: keys}, nil
}

func (s *GRPCCacheServer) Flush(ctx context.Context, _ *pb.FlushRequest) (res *pb.FlushResponse, err error) {
	if err := s.distributor.Flush(ctx); err != nil {
		return nil, err
	}
	return &pb.FlushResponse{Status: true}, nil
}

func (s *GRPCCacheServer) TTL(ctx context.Context, req *pb.TTLRequest) (res *pb.TTLResponse, err error) {
	ttl, found, err := s.distributor.TTL(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, status.Error(codes.NotFound, "key not found")
	}
	if ttl < 0 {
		return &pb.TTLResponse{TtlSeconds: -1, Found: true}, nil
	}
	return &pb.TTLResponse{TtlSeconds: int64(ttl.Seconds()), Found: true}, nil
}

func (s *GRPCCacheServer) Expire(ctx context.Context, req *pb.ExpireRequest) (res *pb.ExpireResponse, err error) {
	if err := s.distributor.Expire(ctx, req.Key, time.Duration(req.Ttl)*time.Second); err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "key not found")
		}
		return nil, err
	}
	return &pb.ExpireResponse{Status: true}, nil
}

func (s *GRPCCacheServer) Persist(ctx context.Context, req *pb.PersistRequest) (res *pb.PersistResponse, err error) {
	if err := s.distributor.Persist(ctx, req.Key); err != nil {
		if errors.Is(err, internal.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "key not found")
		}
		return nil, err
	}
	return &pb.PersistResponse{Status: true}, nil
}

func (s *GRPCCacheServer) Incr(ctx context.Context, req *pb.IncrRequest) (res *pb.IncrResponse, err error) {
	val, err := s.distributor.Incr(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	return &pb.IncrResponse{Value: val}, nil
}

func (s *GRPCCacheServer) Decr(ctx context.Context, req *pb.DecrRequest) (res *pb.DecrResponse, err error) {
	val, err := s.distributor.Decr(ctx, req.Key)
	if err != nil {
		return nil, err
	}
	return &pb.DecrResponse{Value: val}, nil
}

func (s *GRPCCacheServer) SetNX(ctx context.Context, req *pb.SetNXRequest) (res *pb.SetNXResponse, err error) {
	success, err := s.distributor.SetNX(ctx, req.Key, req.Value, time.Duration(req.Ttl)*time.Second)
	if err != nil {
		return nil, err
	}
	return &pb.SetNXResponse{Success: success}, nil
}

func (s *GRPCCacheServer) GetSet(ctx context.Context, req *pb.GetSetRequest) (res *pb.GetSetResponse, err error) {
	oldValue, err := s.distributor.GetSet(ctx, req.Key, req.Value)
	if err != nil {
		return nil, err
	}
	found := oldValue != nil
	return &pb.GetSetResponse{OldValue: oldValue, Found: found}, nil
}

func (s *GRPCCacheServer) MGet(ctx context.Context, req *pb.MGetRequest) (res *pb.MGetResponse, err error) {
	result, err := s.distributor.MGet(ctx, req.Keys)
	if err != nil {
		return nil, err
	}
	return &pb.MGetResponse{Kv: result}, nil
}

func (s *GRPCCacheServer) MSet(ctx context.Context, req *pb.MSetRequest) (res *pb.MSetResponse, err error) {
	if err := s.distributor.MSet(ctx, req.Kv, time.Duration(req.Ttl)*time.Second); err != nil {
		return nil, err
	}
	return &pb.MSetResponse{Status: true}, nil
}

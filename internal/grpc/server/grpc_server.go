package server

import (
	"context"
	"go-cache-server-mini/internal/distributed/router"
	"go-cache-server-mini/internal/grpc/pb"
	"time"

	"google.golang.org/grpc"
)

// another node input cache ops to local node via gRPC

type GRPCCacheServer struct {
	pb.UnimplementedCacheServiceServer
	distributor router.DistributorInterface
}

func StartGRPCCacheServer(ctx context.Context, distributor router.DistributorInterface) *GRPCCacheServer {
	grpcServer := grpc.NewServer()
	grpcCacheServer := NewGRPCCacheServer(distributor)
	pb.RegisterCacheServiceServer(grpcServer, grpcCacheServer)
	return grpcCacheServer
}

func NewGRPCCacheServer(distributor router.DistributorInterface) *GRPCCacheServer {
	return &GRPCCacheServer{
		distributor: distributor,
	}
}

func (s *GRPCCacheServer) Get(ctx context.Context, req *pb.GetRequest) (res *pb.GetResponse, err error) {
	value, found, err := s.distributor.Get(req.Key)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	resp := &pb.GetResponse{
		Value: value,
	}
	return resp, nil
}

func (s *GRPCCacheServer) Set(ctx context.Context, req *pb.SetRequest) (res *pb.SetResponse, err error) {
	err = s.distributor.Set(req.Key, req.Value, time.Duration(req.Ttl))
	if err != nil {
		return nil, err
	}
	return &pb.SetResponse{}, nil
}

func (s *GRPCCacheServer) Del(ctx context.Context, req *pb.DelRequest) (res *pb.DelResponse, err error) {
	err = s.distributor.Del(req.Key)
	if err != nil {
		return nil, err
	}
	return &pb.DelResponse{}, nil
}

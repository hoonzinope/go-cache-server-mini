package server

import (
	"context"
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
	value, found, err := s.distributor.Get(req.Key)
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
	err = s.distributor.Set(req.Key, req.Value, time.Duration(req.Ttl)*time.Second)
	if err != nil {
		return nil, err
	}
	return &pb.SetResponse{
		Status: true,
	}, nil
}

func (s *GRPCCacheServer) Del(ctx context.Context, req *pb.DelRequest) (res *pb.DelResponse, err error) {
	err = s.distributor.Del(req.Key)
	if err != nil {
		return nil, err
	}
	return &pb.DelResponse{
		Status: true,
	}, nil
}

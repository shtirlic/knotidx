package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"syscall"

	"github.com/shtirlic/knotidx/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var (
	grpcServer *grpc.Server
)

type grpcKnotidxServer struct {
	pb.UnimplementedKnotidxServer
}

func (s *grpcKnotidxServer) Reload(context.Context, *pb.EmptyRequest) (*pb.EmptyResponse, error) {
	syscall.Kill(syscall.Getpid(), syscall.SIGHUP)
	return &pb.EmptyResponse{}, nil
}

func (s *grpcKnotidxServer) Shutdown(context.Context, *pb.EmptyRequest) (*pb.EmptyResponse, error) {
	syscall.Kill(syscall.Getpid(), syscall.SIGQUIT)
	return &pb.EmptyResponse{}, nil
}

func (s *grpcKnotidxServer) GetKeys(ctx context.Context, sr *pb.SearchRequest) (*pb.SearchResponse, error) {
	slog.Debug("search request", "text", sr.Query)
	keys := gStore.Keys(sr.Query)

	var results []*pb.SearchItemResponse
	for _, key := range keys {
		results = append(results, &pb.SearchItemResponse{Key: key})
	}
	sre := &pb.SearchResponse{Results: results, Count: int32(len(results))}
	return sre, nil
}

func NewGrpcServer() *grpcKnotidxServer {
	s := &grpcKnotidxServer{}
	return s
}

func startGrpcServer() error {
	port := 12345

	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		slog.Debug("failed to listen: %v", "err", err)
		return err
	}
	var opts []grpc.ServerOption
	grpcServer = grpc.NewServer(opts...)

	pb.RegisterKnotidxServer(grpcServer, NewGrpcServer())
	reflection.Register(grpcServer)
	grpcServer.Serve(lis)

	return nil
}

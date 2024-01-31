package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"syscall"

	"github.com/shtirlic/knotidx/internal/config"
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

func (s *grpcKnotidxServer) ResetScheduler(context.Context, *pb.EmptyRequest) (*pb.EmptyResponse, error) {
	resetScheduler(gConf.Interval)
	return &pb.EmptyResponse{}, nil
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
	keys := gStore.Keys("", sr.Query, 100)

	var results []*pb.SearchItemResponse
	for _, key := range keys {
		results = append(results, &pb.SearchItemResponse{Key: key})
	}
	sre := &pb.SearchResponse{Results: results, Count: int32(len(results))}
	return sre, nil
}

func NewGRPCServer() *grpcKnotidxServer {
	return &grpcKnotidxServer{}
}

func stopGRPCServer() {
	slog.Info("Stopping GRPC Server")
	if grpcServer != nil {
		grpcServer.GracefulStop()
	}
}

func startGRPCServer(c config.GRPCConfig) {
	if !c.Server {
		return
	}

	var network, address string
	network = string(c.Type)

	if c.Type == config.GrpcServerUnixType {
		address = c.Path
	}
	if c.Type == config.GrpcServerTcpType {
		host := c.Host
		address = fmt.Sprintf("%s:%d", host, c.Port)
	}

	slog.Info("Starting GRPC Server", "address", address, "network", network)

	lis, err := net.Listen(network, address)
	if err != nil {
		slog.Debug("failed to listen: %v", "err", err)
		return
	}
	var opts []grpc.ServerOption
	grpcServer = grpc.NewServer(opts...)

	pb.RegisterKnotidxServer(grpcServer, NewGRPCServer())
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		slog.Error("Can't start GRPC Server", "err", err)
		grpcServer = nil
	}

}

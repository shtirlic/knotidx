package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/pb"
	"github.com/shtirlic/knotidx/internal/store"
	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPServer struct {
	server *grpc.Server
	store  store.Store
	config config.Config
	pb.UnimplementedKnotidxServer
}

func (s *GRPServer) ResetScheduler(context.Context, *pb.EmptyRequest) (*pb.EmptyResponse, error) {
	//  resetScheduler(daemonConf.Interval) //todo fix daemon
	return &pb.EmptyResponse{}, nil
}

func (s *GRPServer) Reload(context.Context, *pb.EmptyRequest) (*pb.EmptyResponse, error) {
	unix.Kill(unix.Getpid(), unix.SIGHUP)
	return &pb.EmptyResponse{}, nil
}

func (s *GRPServer) Shutdown(context.Context, *pb.EmptyRequest) (*pb.EmptyResponse, error) {
	unix.Kill(unix.Getpid(), unix.SIGQUIT)
	return &pb.EmptyResponse{}, nil
}

func (s *GRPServer) GetKeys(ctx context.Context, sr *pb.SearchRequest) (*pb.SearchResponse, error) {
	keys := s.store.Keys("", sr.Query, 100)

	sre := &pb.SearchResponse{}
	for _, key := range keys {
		sre.Results = append(sre.Results, &pb.SearchItemResponse{Key: key})
	}
	sre.Count = int32(len(sre.Results))

	slog.Debug("GRPC Search request", "text", sr.Query, "results", sre.Count)
	return sre, nil
}

func NewGRPCServer(c config.Config, s store.Store) *GRPServer {
	return &GRPServer{
		config: c,
		store:  s,
	}
}

func (s *GRPServer) Enabled() bool {
	return s.config.GRPC.Server
}

func (s *GRPServer) Stop() {
	slog.Info("Stopping GRPC Server")
	if s.server != nil {
		s.server.GracefulStop()
	}
}

func (s *GRPServer) Start() {
	if !s.Enabled() {
		return
	}

	var network, address string
	network = string(s.config.GRPC.Type)

	if s.config.GRPC.Type == config.GrpcServerUnixType {
		address = s.config.GRPC.Path
	}
	if s.config.GRPC.Type == config.GrpcServerTcpType {
		host := s.config.GRPC.Host
		address = fmt.Sprintf("%s:%d", host, s.config.GRPC.Port)
	}

	slog.Info("Starting GRPC Server", "address", address, "network", network)

	lis, err := net.Listen(network, address)
	if err != nil {
		slog.Debug("failed to listen: %v", "err", err)
		return
	}
	var opts []grpc.ServerOption
	s.server = grpc.NewServer(opts...)

	pb.RegisterKnotidxServer(s.server, s)
	reflection.Register(s.server)

	if err := s.server.Serve(lis); err != nil {
		slog.Error("Can't start GRPC Server", "err", err)
		s.server = nil
	}
}

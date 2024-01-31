package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/shtirlic/knotidx/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func idxClient() error {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	var address string
	c := gConf.GRPC

	if c.Type == "unix" {
		address = fmt.Sprintf("unix://%s", c.Path)
	}
	if c.Type == "tcp" {
		host := "localhost"
		address = fmt.Sprintf("%s:%d", host, c.Port)
	}

	slog.Info("GRPC Client Connect", "address", address)
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	grpcClient := pb.NewKnotidxClient(conn)
	s := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter query: ")
	for s.Scan() {
		text := s.Text()
		res, err := grpcClient.GetKeys(context.Background(), &pb.SearchRequest{Query: text})
		if err != nil {
			return err
		}

		jr, err := json.MarshalIndent(res.Results, "", "\t")
		if err != nil {
			return err
		}

		fmt.Println("Results", "results")
		fmt.Println(string(jr))
		fmt.Print("Enter query: ")
	}
	programExitCode = 0
	return nil
}

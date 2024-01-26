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

func searchClient() error {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	var address string

	c := gConf.Grpc

	if c.Type == "unix" {
		address = "unix://" + c.Path
	}
	if c.Type == "tcp" {
		host := "localhost"
		address = fmt.Sprintf("%s:%d", host, c.Port)
	}

	slog.Info("GRPC Client Connect to:", "address", address)
	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	grpcClient := pb.NewKnotidxClient(conn)
	s := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter text: ")
	for s.Scan() {
		text := s.Text()
		res, err := grpcClient.GetKeys(context.Background(), &pb.SearchRequest{Query: text})
		if err != nil {
			return err
		}

		jr, _ := json.MarshalIndent(res.Results, "", "\t")
		fmt.Println("Search", "results")
		fmt.Println(string(jr))
		fmt.Print("Enter query: ")
	}
	programExitCode = 0
	return nil
}

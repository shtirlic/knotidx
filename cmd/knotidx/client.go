//go:generate protoc --go_out=../../. --go_opt=paths=import --go-grpc_out=../../. --go-grpc_opt=paths=import --proto_path=../../proto  knotidx.proto
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/shtirlic/knotidx/internal/config"
	"github.com/shtirlic/knotidx/internal/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func idxClient(c config.GRPCConfig) error {

	var opts []grpc.DialOption
	var err error
	var jr []byte
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	var address string

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

	if !*jsonCmd {
		fmt.Print("Enter query: ")
	}
	for s.Scan() {
		text := s.Text()
		res, err := grpcClient.GetKeys(context.Background(), &pb.SearchRequest{Query: text})
		if err != nil {
			return err
		}

		// if *jsonCmd {
		// jr, err = json.Marshal(res.Results)
		// } else {
		jr, err = json.MarshalIndent(res.Results, "", "\t")
		// }

		if err != nil {
			return err
		}
		if *jsonCmd {
			fmt.Println(string(jr))
			programExitCode = 0
			return nil
		} else {
			fmt.Println("Results", "results")
			fmt.Println(string(jr))
			fmt.Print("Enter query: ")
		}
	}
	programExitCode = 0
	return nil
}

package main

import (
	"context"
	"flag"
	"log"

	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	addr := flag.String("address", "localhost:8080", "Server address in the format host:port")
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	laptopClient := protoc.NewLaptopServiceClient(conn)

	randLap := sample.NewLaptop()
	req := &protoc.CreateLaptopRequest{Laptop: randLap}

	res, err := laptopClient.CreateLaptop(context.Background(), req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Printf("Laptop with ID %s already exists", randLap.GetId())
		} else {
			log.Fatalf("Failed to create laptop: %v", err)
		}

		return
	}

	log.Printf("Laptop created: %s", res.GetId())
}

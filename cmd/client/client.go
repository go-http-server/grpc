package main

import (
	"context"
	"flag"
	"io"
	"log"
	"time"

	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func createLaptop(laptopClient protoc.LaptopServiceClient) {
	randLap := sample.NewLaptop()
	req := &protoc.CreateLaptopRequest{Laptop: randLap}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := laptopClient.CreateLaptop(ctx, req)
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

func searchLaptop(laptopClient protoc.LaptopServiceClient, filter *protoc.Filter) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &protoc.SearchLaptopRequest{Filter: filter}
	stream, err := laptopClient.SearchLaptop(ctx, req)
	if err != nil {
		log.Fatalf("Failed to search laptops: %v", err)
	}

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			return
		}

		if err != nil {
			log.Fatalf("Failed to receive laptop: %v", err)
		}

		laptop := res.GetLaptop()
		log.Printf("Laptop found: %s", laptop.GetId())
		log.Printf("  Name: %s", laptop.GetName())
		log.Printf("  Brand: %s", laptop.GetBrand())
		log.Printf("  Price: $%.2f", laptop.GetPriceUsd())
		log.Printf("  CPU Cores: %d", laptop.GetCpu().GetNumCores())
		log.Printf("  CPU GHz: %.2f", laptop.GetCpu().GetMinGhz())
		log.Printf("  Memory: %d%s", laptop.GetRam().GetValue(), laptop.GetRam().GetUnit().String())
	}
}

func main() {
	addr := flag.String("address", "localhost:8080", "Server address in the format host:port")
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	laptopClient := protoc.NewLaptopServiceClient(conn)
	for range 10 {
		createLaptop(laptopClient)
	}

	filter := &protoc.Filter{
		MaxPriceUsd: 3000,
		MinCpuCores: 2,
		MinCpuGhz:   1.5,
		MinMemory: &protoc.Memory{
			Value: 4,
			Unit:  protoc.Memory_GIGABYTE,
		},
	}
	searchLaptop(laptopClient, filter)
}

package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-http-server/grpc/client"
	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func testCreateLaptop(laptopClient *client.LaptopClient) {
	laptopClient.CreateLaptop(sample.NewLaptop()) // Create a sample laptop
}

func testSearchLaptop(laptopClient *client.LaptopClient) {
	for range 10 {
		laptopClient.CreateLaptop(sample.NewLaptop()) // Create a sample laptop
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
	laptopClient.SearchLaptop(filter)
}

func testUploadImage(laptopClient *client.LaptopClient) {
	laptop := sample.NewLaptop()
	laptopClient.CreateLaptop(laptop)
	laptopClient.UploadImage(laptop.GetId(), "./tmp/image.jpg")
}

func testRateLaptop(laptopClient *client.LaptopClient) {
	n := 3
	laptopIDs := make([]string, n)
	for i := range n {
		laptop := sample.NewLaptop()
		laptopIDs[i] = laptop.GetId()
		laptopClient.CreateLaptop(laptop)
	}

	scores := make([]float64, n)
	for {
		fmt.Print("create laptop: y/n?")
		var answer string
		fmt.Scan(&answer)

		if strings.ToLower(answer) != "y" {
			break
		}

		for i := range n {
			scores[i] = sample.RandomLaptopScore()
		}

		err := laptopClient.RateLaptop(laptopIDs, scores)
		if err != nil {
			log.Fatalf("Failed to rate laptops: %v", err)
		}
	}
}

func authMethods() map[string]bool {
	const laptopServiceMethod = "/LaptopService/"
	return map[string]bool{
		laptopServiceMethod + "CreateLaptop": true,
		laptopServiceMethod + "SearchLaptop": false,
		laptopServiceMethod + "RateLaptop":   true,
		laptopServiceMethod + "UploadImage":  true,
	}
}

func main() {
	addr := flag.String("address", "localhost:8080", "Server address in the format host:port")
	flag.Parse()

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	authClient := client.NewAuthClient(conn, "user", "password")
	interceptor, err := client.NewAuthInterceptor(authClient, authMethods(), 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create auth interceptor: %v", err)
	}

	connAuth, err := grpc.NewClient(*addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	laptopClient := client.NewLaptopClient(connAuth)
	testRateLaptop(laptopClient) // Test rating laptops
}

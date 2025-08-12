package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-http-server/grpc/client"
	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	const routeGuideServiceMethod = "/RouteGuide/"
	return map[string]bool{
		laptopServiceMethod + "CreateLaptop":   true,
		laptopServiceMethod + "SearchLaptop":   false,
		laptopServiceMethod + "RateLaptop":     true,
		laptopServiceMethod + "UploadImage":    true,
		routeGuideServiceMethod + "GetFeature": true,
	}
}

func loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load TLS credentials from a file or other source
	// For simplicity, we are returning insecure credentials here.
	// In a real application, you would load your TLS certs and keys.
	pemServerCA, err := os.ReadFile("certs/ca.crt")
	if err != nil {
		return nil, nil
	}

	clientCert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pemServerCA) {
		return nil, fmt.Errorf("failed to append server CA certificate")
	}

	// Create credentials from the loaded certs
	config := &tls.Config{
		RootCAs:      certPool,
		Certificates: []tls.Certificate{clientCert},
	}
	return credentials.NewTLS(config), nil
}

func main() {
	addr := flag.String("address", "localhost:8080", "Server address in the format host:port")
	enableTLS := flag.Bool("tls", false, "Enable TLS for the connection")
	flag.Parse()

	transportOpts := grpc.WithTransportCredentials(insecure.NewCredentials())
	if *enableTLS {
		tlsCredentials, err := loadTLSCredentials()
		if err != nil {
			log.Fatalf("Failed to load TLS credentials: %v", err)
		}
		transportOpts = grpc.WithTransportCredentials(tlsCredentials)
	}

	conn, err := grpc.NewClient(*addr, transportOpts)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	authClient := client.NewAuthClient(conn, "admin", "password")
	interceptor, err := client.NewAuthInterceptor(authClient, authMethods(), 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create auth interceptor: %v", err)
	}

	connAuth, err := grpc.NewClient(*addr,
		transportOpts,
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	log.Printf("Connected to server at %s, with transport option TLS: %t", *addr, *enableTLS)

	// laptopClient := client.NewLaptopClient(connAuth)
	routeGuideClient := client.NewRouteGuideClient(connAuth)
	feature, err := routeGuideClient.GetFeature(&protoc.Point{Latitude: 409146138, Longitude: -746188906})
	if err != nil {
		log.Fatalf("Failed to get feature: %v", err)
	}
	log.Printf("Feature: %+v", feature)
}

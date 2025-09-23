package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/go-http-server/grpc/client"
	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const retryPolicy = `
{
  "methodConfig": [
    {
      "name": [
        {
          "service": "/LaptopService/CreateLaptop"
        }
      ],
      "retryPolicy": {
        "MaxAttempts": 4,
        "InitialBackoff": ".01s",
        "MaxBackoff": ".01s",
        "BackoffMultiplier": 1,
        "RetryableStatusCodes": [
          "UNAVAILABLE"
        ]
      }
    }
  ]
}`

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
		laptopServiceMethod + "CreateLaptop":     true,
		laptopServiceMethod + "SearchLaptop":     false,
		laptopServiceMethod + "RateLaptop":       true,
		laptopServiceMethod + "UploadImage":      true,
		routeGuideServiceMethod + "GetFeature":   true,
		routeGuideServiceMethod + "ListFeatures": true,
		routeGuideServiceMethod + "RecordRoute":  true,
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

func randomPoint() *protoc.Point {
	lat := (rand.Int32N(180) - 90) * 1e7
	long := (rand.Int32N(360) - 180) * 1e7
	return &protoc.Point{Latitude: lat, Longitude: long}
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

	// keepalive option in connection
	kacp := keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: true,             // send pings even without active streams
	}
	kac := grpc.WithKeepaliveParams(kacp)

	// serviceConfig includes: retry policy
	sConf := grpc.WithDefaultServiceConfig(retryPolicy)

	conn, err := grpc.NewClient(*addr, transportOpts, kac, sConf)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer conn.Close()

	authClient := client.NewAuthClient(conn, "admin_valid", "password")
	interceptor, err := client.NewAuthInterceptor(authClient, authMethods(), 5*time.Second)
	if err != nil {
		log.Fatalf("Failed to create auth interceptor: %v", err)
	}

	connAuth, err := grpc.NewClient(*addr,
		transportOpts,
		kac,
		grpc.WithUnaryInterceptor(interceptor.Unary()),
		grpc.WithStreamInterceptor(interceptor.Stream()),
		sConf,
	)
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}
	defer connAuth.Close()

	log.Printf("Connected to server at %s, with transport option TLS: %t", *addr, *enableTLS)

	laptopClient := client.NewLaptopClient(connAuth)
	routeGuideClient := client.NewRouteGuideClient(connAuth)
	feature, err := routeGuideClient.GetFeature(&protoc.Point{Latitude: 409146138, Longitude: -746188906})
	if err != nil {
		log.Fatalf("Failed to get feature: %v", err)
	}
	log.Printf("Feature: %+v", feature)

	feature, err = routeGuideClient.GetFeature(&protoc.Point{Latitude: 0, Longitude: 0})
	if err != nil {
		log.Fatalf("Failed to get feature: %v", err)
	}
	log.Printf("Feature: %+v", feature)

	// server streaming
	err = routeGuideClient.ListFeatures(&protoc.Rectangle{
		Lo: &protoc.Point{Latitude: 400000000, Longitude: -750000000},
		Hi: &protoc.Point{Latitude: 420000000, Longitude: -730000000},
	})
	if err != nil {
		log.Fatalf("Failed to list features: %v", err)
	}

	// Create a random number of random points
	pointCount := int(rand.Int32N(100)) + 2 // Traverse at least two points
	var points []*protoc.Point
	for range pointCount {
		points = append(points, randomPoint())
	}
	err = routeGuideClient.RecordRoute(points)
	if err != nil {
		log.Fatalf("Failed to record route: %v", err)
	}

	// bidirectional streaming
	notes := []*protoc.RouteNote{
		{Location: &protoc.Point{Latitude: 0, Longitude: 1}, Message: "First message"},
		{Location: &protoc.Point{Latitude: 0, Longitude: 2}, Message: "Second message"},
		{Location: &protoc.Point{Latitude: 0, Longitude: 3}, Message: "Third message"},
		{Location: &protoc.Point{Latitude: 0, Longitude: 1}, Message: "Fourth message"},
		{Location: &protoc.Point{Latitude: 0, Longitude: 2}, Message: "Fifth message"},
		{Location: &protoc.Point{Latitude: 0, Longitude: 3}, Message: "Sixth message"},
	}
	err = routeGuideClient.RouteChat(notes)
	if err != nil {
		log.Fatalf("Failed to chat on route: %v", err)
	}
	testCreateLaptop(laptopClient)
	testSearchLaptop(laptopClient)
}

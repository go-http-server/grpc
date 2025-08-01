package service_test

import (
	"context"
	"testing"

	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"github.com/go-http-server/grpc/service"
)

func TestServerCreateLaptop(t *testing.T) {
	// Initialize the server and store
	store := service.NewInMemoryLaptopStore()
	server := service.NewLaptopServer(store)

	// Create a new laptop request
	laptop := sample.NewLaptop()
	req := &protoc.CreateLaptopRequest{Laptop: laptop}

	// Call the CreateLaptop method
	res, err := server.CreateLaptop(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateLaptop failed: %v", err)
	}

	// Check if the response contains the correct ID
	if res.Id != laptop.Id {
		t.Errorf("Expected ID %s, got %s", laptop.Id, res.Id)
	}
}

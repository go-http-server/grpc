// Package service provides the gRPC server implementation for the LaptopService.
package service

import (
	"context"
	"errors"
	"log"

	"github.com/go-http-server/grpc/protoc"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LaptopServer is the server API for LaptopService service.
type LaptopServer struct {
	protoc.UnimplementedLaptopServiceServer
	Store LaptopStore
}

// NewLaptopServer creates a new instance of LaptopServer.
func NewLaptopServer(store LaptopStore) *LaptopServer {
	return &LaptopServer{Store: store}
}

// CreateLaptop handles the creation of a new laptop.
func (s *LaptopServer) CreateLaptop(ctx context.Context, req *protoc.CreateLaptopRequest) (*protoc.CreateLaptopResponse, error) {
	laptopReq := req.GetLaptop()
	log.Printf("Received request to create laptop: %s", laptopReq.GetId())

	if len(laptopReq.Id) > 0 {
		_, err := uuid.Parse(laptopReq.GetId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "laptop id is invalid: %s", err)
		}

	} else {
		// generate a new UUID for the laptop
		id, err := uuid.NewRandom()
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate laptop id: %s", err)
		}

		laptopReq.Id = id.String()
	}

	// save laptop to database
	err := s.Store.Save(laptopReq)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "laptop with id %s already exists", laptopReq.GetId())
		}

		return nil, status.Errorf(codes.Internal, "failed to save laptop: %s", err)
	}

	res := &protoc.CreateLaptopResponse{
		Id: laptopReq.Id, // Return the ID of the created laptop
	}
	return res, nil
}

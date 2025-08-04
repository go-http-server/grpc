// Package service provides the gRPC server implementation for the LaptopService.
package service

import (
	"context"
	"errors"
	"log"

	"github.com/go-http-server/grpc/protoc"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// LaptopServer is the server API for LaptopService service.
type LaptopServer struct {
	protoc.UnimplementedLaptopServiceServer
	LaptopStore LaptopStore
	ImgStore    ImageStore
}

// NewLaptopServer creates a new instance of LaptopServer.
func NewLaptopServer(store LaptopStore) *LaptopServer {
	return &LaptopServer{LaptopStore: store}
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

	if errors.Is(ctx.Err(), context.Canceled) {
		return nil, status.Errorf(codes.Canceled, "context was canceled")
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, status.Errorf(codes.DeadlineExceeded, "deadline context exceed")
	}

	// save laptop to database
	err := s.LaptopStore.Save(laptopReq)
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

// SearchLaptop handles the search for laptops based on filter criteria.
func (s *LaptopServer) SearchLaptop(req *protoc.SearchLaptopRequest, streaming grpc.ServerStreamingServer[protoc.SearchLaptopResponse]) error {
	filter := req.GetFilter()

	log.Printf("Received request to search laptops with filter: %+v", filter)

	err := s.LaptopStore.Search(streaming.Context(), filter, func(laptop *protoc.Laptop) error {
		res := &protoc.SearchLaptopResponse{Laptop: laptop}

		// stream the laptop response back to the client
		err := streaming.Send(res)
		if err != nil {
			return err
		}

		log.Printf("Sent laptop: %s", laptop.GetId())

		return nil
	})
	if err != nil {
		return status.Errorf(codes.Internal, "failed to search laptops: %s", err)
	}
	return nil
}

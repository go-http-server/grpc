// Package service provides the gRPC server implementation for the LaptopService.
package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"time"

	"github.com/go-http-server/grpc/protoc"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	maxImageSize = 10 << 20
)

// LaptopServer is the server API for LaptopService service.
type LaptopServer struct {
	protoc.UnimplementedLaptopServiceServer
	LaptopStore LaptopStore
	ImgStore    ImageStore
	RateStore   RatingStore
}

// NewLaptopServer creates a new instance of LaptopServer.
func NewLaptopServer(store LaptopStore, imgStore ImageStore, rateStore RatingStore) *LaptopServer {
	return &LaptopServer{LaptopStore: store, ImgStore: imgStore, RateStore: rateStore}
}

// CreateLaptop handles the creation of a new laptop.
func (s *LaptopServer) CreateLaptop(ctx context.Context, req *protoc.CreateLaptopRequest) (*protoc.CreateLaptopResponse, error) {
	defer func() {
		trailer := metadata.Pairs("timestamp", time.Now().Format(time.DateOnly))
		grpc.SetTrailer(ctx, trailer)
	}()
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

	if err := contextError(ctx); err != nil {
		return nil, err
	}

	// save laptop to database
	err := s.LaptopStore.Save(laptopReq)
	if err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "laptop with id %s already exists", laptopReq.GetId())
		}

		return nil, status.Errorf(codes.Internal, "failed to save laptop: %s", err)
	}

	// Create and send header.
	header := metadata.New(map[string]string{"location": "MTV", "timestamp": time.Now().Format(time.DateOnly)})
	grpc.SendHeader(ctx, header)

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

func (s *LaptopServer) UploadImage(clientStreaming grpc.ClientStreamingServer[protoc.UploadImageRequest, protoc.UploadImageResponse]) error {
	// listen first streaming request to receive information of image upload
	req, err := clientStreaming.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot receive image info req: %s", err)
	}

	laptopID := req.GetInfo().GetLaptopId()
	imageType := req.GetInfo().GetImageType()
	log.Printf("Received request to upload image for laptop: %s, type: %s", laptopID, imageType)

	laptop, err := s.LaptopStore.Find(laptopID)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot find laptop with id %s: %s", laptopID, err)
	}
	if laptop == nil {
		return status.Errorf(codes.InvalidArgument, "laptop not found")
	}

	imageData := bytes.Buffer{}
	imageSize := 0

	// loop to receive streaming request to get chunk data image from client streaming
	for {
		if err := contextError(clientStreaming.Context()); err != nil {
			return err
		}
		req, err := clientStreaming.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive chunk image data: %s", err)
		}

		chunk := req.GetChunkData()
		size := len(chunk)
		imageSize += size

		if imageSize > maxImageSize {
			return status.Errorf(codes.InvalidArgument, "image size exceeds the limit of %d bytes", maxImageSize)
		}

		_, err = imageData.Write(chunk)
		if err != nil {
			return status.Errorf(codes.Internal, "cannot write image data: %s", err)
		}
	}

	imageID, err := s.ImgStore.Save(laptopID, imageType, imageData)
	if err != nil {
		return status.Errorf(codes.Internal, "cannot save image: %s", err)
	}

	res := &protoc.UploadImageResponse{Id: imageID, Size: uint32(imageSize)}
	err = clientStreaming.SendAndClose(res)
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot send response to client")
	}

	log.Printf("Image uploaded successfully for laptop %s, image ID: %s, size: %d bytes", laptopID, imageID, imageSize)

	return nil
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return status.Errorf(codes.Canceled, "context was canceled")
	case context.DeadlineExceeded:
		return status.Errorf(codes.DeadlineExceeded, "deadline context exceed")
	default:
		return nil
	}
}

// RateLaptop handles the rating of laptops through a bidirectional streaming RPC.
func (s *LaptopServer) RateLaptop(stream grpc.BidiStreamingServer[protoc.RateLaptopRequest, protoc.RateLaptopResponse]) error {
	for {
		err := contextError(stream.Context())
		if err != nil {
			return err
		}

		req, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return status.Errorf(codes.Unknown, "cannot receive rate laptop request: %s", err)
		}

		laptopID := req.GetLaptopId()
		score := req.GetScore()
		log.Printf("Received rating for laptop %s with score %.2f", laptopID, score)

		found, err := s.LaptopStore.Find(laptopID)
		if err != nil {
			return status.Errorf(codes.Internal, "cannot find laptop with id %s: %s", laptopID, err)
		}

		if found == nil {
			return status.Errorf(codes.NotFound, "laptop with id %s not found", laptopID)
		}

		rating, err := s.RateStore.AddRating(laptopID, score)
		if err != nil {
			return status.Errorf(codes.Internal, "cannot add rating for laptop %s: %s", laptopID, err)
		}

		res := &protoc.RateLaptopResponse{
			LaptopId:     laptopID,
			RatedCount:   rating.Count,
			AverageScore: rating.Sum / float64(rating.Count),
		}

		err = stream.Send(res)
		if err != nil {
			return status.Errorf(codes.Unknown, "cannot send response to client: %s", err)
		}
	}

	return nil
}

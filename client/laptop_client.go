package client

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-http-server/grpc/protoc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
)

// LaptopClient is a client for interacting with the laptop service.
type LaptopClient struct {
	service protoc.LaptopServiceClient
}

// NewLaptopClient creates a new LaptopClient instance.
func NewLaptopClient(cc *grpc.ClientConn) *LaptopClient {
	service := protoc.NewLaptopServiceClient(cc)
	return &LaptopClient{service: service}
}

// CreateLaptop sends a request to create a new laptop in the service.
func (laptopClient *LaptopClient) CreateLaptop(laptop *protoc.Laptop) {
	req := &protoc.CreateLaptopRequest{Laptop: laptop}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := laptopClient.service.CreateLaptop(ctx, req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Printf("Laptop with ID %s already exists", laptop.GetId())
		} else {
			log.Fatalf("Failed to create laptop: %v", err)
		}

		return
	}

	log.Printf("Laptop created: %s", res.GetId())
}

// SearchLaptop sends a request to search for laptops based on the provided filter.
func (laptopClient *LaptopClient) SearchLaptop(filter *protoc.Filter) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req := &protoc.SearchLaptopRequest{Filter: filter}
	stream, err := laptopClient.service.SearchLaptop(ctx, req, grpc.UseCompressor(gzip.Name))
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

// UploadImage uploads an image for a laptop identified by laptopID.
func (laptopClient *LaptopClient) UploadImage(laptopID string, imagePath string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.service.UploadImage(ctx, grpc.UseCompressor(gzip.Name))
	if err != nil {
		log.Fatal("Failed to upload image: ", err, stream.RecvMsg(nil))
	}

	req := &protoc.UploadImageRequest{Data: &protoc.UploadImageRequest_Info{
		Info: &protoc.ImageInfo{
			LaptopId:  laptopID,
			ImageType: filepath.Ext(imagePath), // Assuming JPEG for simplicity
		},
	}}
	err = stream.Send(req)
	if err != nil {
		log.Fatal("Failed to send image info: ", err, stream.RecvMsg(nil))
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024) // 1KB buffer
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("cannot read chunk to buffer: %v", err)
		}

		req := &protoc.UploadImageRequest{
			Data: &protoc.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		if err != nil {
			log.Fatalf("Failed to send image chunk: %v", err)
		}
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Failed to receive upload image response: %v", err)
	}

	log.Printf("Image uploaded successfully for laptop %s, image ID: %s, size: %d", laptopID, res.GetId(), res.GetSize())
}

// RateLaptop sends a request to rate multiple laptops with their respective scores.
func (laptopClient *LaptopClient) RateLaptop(laptopIDs []string, scores []float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.service.RateLaptop(ctx, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return err
	}

	waitResponse := make(chan error)
	go func() {
		for {
			res, err := stream.Recv()
			if err == io.EOF {
				waitResponse <- nil
				return
			}
			if err != nil {
				waitResponse <- err
				return
			}

			log.Printf("Received response for laptop %s: RatedCount=%d, AverageScore=%.2f", res.GetLaptopId(), res.GetRatedCount(), res.GetAverageScore())
		}
	}()

	// send request rating laptop
	for i, laptopID := range laptopIDs {
		req := &protoc.RateLaptopRequest{
			LaptopId: laptopID,
			Score:    scores[i],
		}

		err := stream.Send(req)
		if err != nil {
			return fmt.Errorf("failed to send rate laptop request: %v, %v", err, stream.RecvMsg(nil))
		}

		log.Printf("Sent rating for laptop %s with score %.2f", laptopID, scores[i])
	}

	err = stream.CloseSend()
	if err != nil {
		return fmt.Errorf("failed to close stream: %v, %v", err, stream.RecvMsg(nil))
	}

	err = <-waitResponse
	return err
}

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func createLaptop(laptopClient protoc.LaptopServiceClient, laptop *protoc.Laptop) {
	req := &protoc.CreateLaptopRequest{Laptop: laptop}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := laptopClient.CreateLaptop(ctx, req)
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

func testCreateLaptop(laptopClient protoc.LaptopServiceClient) {
	createLaptop(laptopClient, sample.NewLaptop()) // Create a sample laptop
}

func testSearchLaptop(laptopClient protoc.LaptopServiceClient) {
	for range 10 {
		createLaptop(laptopClient, sample.NewLaptop()) // Create a sample laptop
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

func uploadImage(laptopClient protoc.LaptopServiceClient, laptopID string, imagePath string) {
	file, err := os.Open(imagePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := laptopClient.UploadImage(ctx)
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

func testUploadImage(laptopClient protoc.LaptopServiceClient) {
	laptop := sample.NewLaptop()
	createLaptop(laptopClient, laptop)
	uploadImage(laptopClient, laptop.GetId(), "./tmp/image.jpg")
}

func rateLaptop(client protoc.LaptopServiceClient, laptopIDs []string, scores []float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.RateLaptop(ctx)
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

func testRateLaptop(laptopClient protoc.LaptopServiceClient) {
	n := 3
	laptopIDs := make([]string, n)
	for i := range n {
		laptop := sample.NewLaptop()
		laptopIDs[i] = laptop.GetId()
		createLaptop(laptopClient, laptop)
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

		err := rateLaptop(laptopClient, laptopIDs, scores)
		if err != nil {
			log.Fatalf("Failed to rate laptops: %v", err)
		}
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
	testRateLaptop(laptopClient) // Test rating laptops
}

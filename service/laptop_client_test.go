package service_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"github.com/go-http-server/grpc/serializer"
	"github.com/go-http-server/grpc/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestClientCreateLaptop(t *testing.T) {
	t.Parallel()

	laptopStore := service.NewInMemoryLaptopStore()
	serverAddr := startTestLaptopServer(t, laptopStore, nil, nil)

	conn := newClientConnection(t, serverAddr)
	defer conn.Close()
	laptopClient := protoc.NewLaptopServiceClient(conn)

	laptop := sample.NewLaptop()
	expectedID := laptop.GetId()
	req := &protoc.CreateLaptopRequest{Laptop: laptop}

	res, err := laptopClient.CreateLaptop(t.Context(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, expectedID, res.GetId())

	laptopFound, err := laptopStore.Find(expectedID)
	require.NoError(t, err)
	require.NotNil(t, laptopFound)
	require.Equal(t, expectedID, laptopFound.GetId())

	requireSameLaptop(t, laptop, laptopFound)
}

func TestClientSearchLaptop(t *testing.T) {
	t.Parallel()

	filter := &protoc.Filter{
		MaxPriceUsd: 3000,
		MinCpuCores: 2,
		MinCpuGhz:   1.5,
		MinMemory: &protoc.Memory{
			Value: 4,
			Unit:  protoc.Memory_GIGABYTE,
		},
	}

	store := service.NewInMemoryLaptopStore()
	expectedIDs := make(map[string]bool)

	for i := range 6 {
		laptop := sample.NewLaptop()

		switch i {
		case 0:
			laptop.PriceUsd = 3500
		case 1:
			laptop.Cpu.NumCores = 1
		case 2:
			laptop.Cpu.MinGhz = 1.0
		case 3:
			laptop.Ram = &protoc.Memory{Value: 4096, Unit: protoc.Memory_MEGABYTE}
		case 4:
			laptop.PriceUsd = 2500
			laptop.Cpu.NumCores = 4
			laptop.Cpu.MinGhz = 2.5
			laptop.Cpu.MaxGhz = 3.5
			laptop.Ram = &protoc.Memory{Value: 16, Unit: protoc.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		case 5:
			laptop.PriceUsd = 3000
			laptop.Cpu.NumCores = 4
			laptop.Cpu.MinGhz = 2.5
			laptop.Cpu.MaxGhz = 3.5
			laptop.Ram = &protoc.Memory{Value: 24, Unit: protoc.Memory_GIGABYTE}
			expectedIDs[laptop.Id] = true
		}

		err := store.Save(laptop)
		require.NoError(t, err)
	}

	serverAddr := startTestLaptopServer(t, store, nil, nil)
	conn := newClientConnection(t, serverAddr)
	defer conn.Close()
	laptopClient := protoc.NewLaptopServiceClient(conn)

	req := &protoc.SearchLaptopRequest{Filter: filter}
	stream, err := laptopClient.SearchLaptop(t.Context(), req)
	require.NoError(t, err)

	found := 0

	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		require.Contains(t, expectedIDs, res.GetLaptop().GetId())
		found++
	}

	require.Equal(t, len(expectedIDs), found)
}

func startTestLaptopServer(t *testing.T, laptopStore service.LaptopStore, imgStore service.ImageStore, ratingStore service.RatingStore) string {
	t.Helper()
	laptopServer := service.NewLaptopServer(laptopStore, imgStore, ratingStore)

	grpcServer := grpc.NewServer()
	protoc.RegisterLaptopServiceServer(grpcServer, laptopServer)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go grpcServer.Serve(listener)

	return listener.Addr().String()
}

func newClientConnection(t *testing.T, serverAddr string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	require.NotNil(t, conn)

	return conn
}

func requireSameLaptop(t *testing.T, expected, actual *protoc.Laptop) {
	expectedJSON, err := serializer.ProtobufToJSON(expected)
	require.NoError(t, err)
	require.NotNil(t, expectedJSON)
	require.NotEmpty(t, expectedJSON)

	actualJSON, err := serializer.ProtobufToJSON(actual)
	require.NoError(t, err)
	require.NotNil(t, actualJSON)
	require.NotEmpty(t, actualJSON)

	require.Equal(t, expectedJSON, actualJSON, "Expected and actual laptops do not match")
}

func TestClientUploadImage(t *testing.T) {
	testImagePath := "../tmp/image.jpg"

	laptopStore := service.NewInMemoryLaptopStore()
	imageStore := service.NewDiskImageStore("../images")

	laptop := sample.NewLaptop()
	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	serverAddr := startTestLaptopServer(t, laptopStore, imageStore, nil)
	conn := newClientConnection(t, serverAddr)
	defer conn.Close()
	laptopClient := protoc.NewLaptopServiceClient(conn)

	file, err := os.Open(testImagePath)
	require.NoError(t, err)
	defer file.Close()

	stream, err := laptopClient.UploadImage(t.Context())
	require.NoError(t, err)

	imageType := filepath.Ext(testImagePath)

	req := &protoc.UploadImageRequest{Data: &protoc.UploadImageRequest_Info{
		Info: &protoc.ImageInfo{
			LaptopId:  laptop.GetId(),
			ImageType: imageType, // Assuming JPEG for simplicity
		},
	}}
	err = stream.Send(req)
	require.NoError(t, err)

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024) // 1KB buffer
	size := 0

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}

		require.NoError(t, err)
		size += n

		req := &protoc.UploadImageRequest{
			Data: &protoc.UploadImageRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		}

		err = stream.Send(req)
		require.NoError(t, err)
	}

	res, err := stream.CloseAndRecv()
	require.NoError(t, err)
	require.NotZero(t, size)
	require.NotZero(t, res.GetId())
	require.EqualValues(t, size, res.GetSize())

	saveImagePath := fmt.Sprintf("../images/%s%s", res.GetId(), imageType)
	require.FileExists(t, saveImagePath)
	require.NoError(t, os.Remove(saveImagePath))
}

func TestClientRateLaptop(t *testing.T) {
	laptopStore := service.NewInMemoryLaptopStore()
	ratingStore := service.NewInMemoryRatingStore()

	laptop := sample.NewLaptop()
	err := laptopStore.Save(laptop)
	require.NoError(t, err)

	serverAddr := startTestLaptopServer(t, laptopStore, nil, ratingStore)
	conn := newClientConnection(t, serverAddr)
	defer conn.Close()
	laptopClient := protoc.NewLaptopServiceClient(conn)

	stream, err := laptopClient.RateLaptop(t.Context())
	require.NoError(t, err)

	scores := []float64{8, 7.5, 10}
	averages := []float64{8, 7.75, 8.5}
	n := len(scores)

	for i := range n {
		req := &protoc.RateLaptopRequest{
			LaptopId: laptop.GetId(),
			Score:    scores[i],
		}

		err := stream.Send(req)
		require.NoError(t, err)
	}

	err = stream.CloseSend()
	require.NoError(t, err)

	for i := 0; ; i++ {
		res, err := stream.Recv()
		if err == io.EOF {
			require.Equal(t, n, i) // reverse the loop when all responses are received
			return
		}

		require.NoError(t, err)

		require.Equal(t, laptop.GetId(), res.GetLaptopId())
		require.Equal(t, uint32(i+1), res.GetRatedCount())
		require.Equal(t, averages[i], res.GetAverageScore())
	}
}

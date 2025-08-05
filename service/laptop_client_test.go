package service_test

import (
	"io"
	"net"
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
	serverAddr := startTestLaptopServer(t, laptopStore, nil)

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

	serverAddr := startTestLaptopServer(t, store, nil)
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

func startTestLaptopServer(t *testing.T, laptopStore service.LaptopStore, imgStore service.ImageStore) string {
	t.Helper()
	laptopServer := service.NewLaptopServer(laptopStore, imgStore)

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

package service_test

import (
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

	laptopServer, serverAddr := startTestLaptopServer(t)

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

	laptopFound, err := laptopServer.Store.Find(expectedID)
	require.NoError(t, err)
	require.NotNil(t, laptopFound)
	require.Equal(t, expectedID, laptopFound.GetId())

	requireSameLaptop(t, laptop, laptopFound)
}

func startTestLaptopServer(t *testing.T) (*service.LaptopServer, string) {
	t.Helper()
	laptopServer := service.NewLaptopServer(service.NewInMemoryLaptopStore())

	grpcServer := grpc.NewServer()
	protoc.RegisterLaptopServiceServer(grpcServer, laptopServer)

	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	go grpcServer.Serve(listener)

	return laptopServer, listener.Addr().String()
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

package service_test

import (
	"context"
	"testing"

	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"github.com/go-http-server/grpc/service"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerCreateLaptop(t *testing.T) {
	t.Parallel()

	laptopNoID := sample.NewLaptop()
	laptopNoID.Id = "" // Ensure no ID is set

	laptopInvalidUUID := sample.NewLaptop()
	laptopInvalidUUID.Id = "invalid-uuid" // Set an invalid UUID

	laptopDuplicateUUID := sample.NewLaptop()
	storeDuplicateLaptop := service.NewInMemoryLaptopStore()
	err := storeDuplicateLaptop.Save(laptopDuplicateUUID)
	require.NoError(t, err)

	testCases := []struct {
		name   string
		laptop *protoc.Laptop
		store  service.LaptopStore
		code   codes.Code
	}{
		{
			name:   "Valid with uuid",
			laptop: sample.NewLaptop(),
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.OK,
		},
		{
			name:   "Valid without uuid",
			laptop: laptopNoID,
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.OK,
		},
		{
			name:   "invalid uuid",
			laptop: laptopInvalidUUID,
			store:  service.NewInMemoryLaptopStore(),
			code:   codes.InvalidArgument,
		},
		{
			name:   "failure save laptop duplicate id",
			laptop: laptopDuplicateUUID,
			store:  storeDuplicateLaptop,
			code:   codes.AlreadyExists,
		},
	}

	for _, currCase := range testCases {
		t.Run(currCase.name, func(t *testing.T) {
			t.Parallel()

			server := service.NewLaptopServer(currCase.store, nil, nil)
			req := &protoc.CreateLaptopRequest{
				Laptop: currCase.laptop,
			}

			res, err := server.CreateLaptop(context.Background(), req)

			if currCase.code == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.NotEmpty(t, res.GetId())
				if len(currCase.laptop.Id) > 0 {
					require.Equal(t, currCase.laptop.Id, res.GetId())
				}
			} else {
				require.Error(t, err)
				require.Nil(t, res)
				st, ok := status.FromError(err)
				require.True(t, ok)
				require.Equal(t, currCase.code, st.Code())
			}
		})
	}
}

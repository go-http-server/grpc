package serializer_test

import (
	"testing"

	"github.com/go-http-server/grpc/protoc"
	"github.com/go-http-server/grpc/sample"
	"github.com/go-http-server/grpc/serializer"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestFileSerializer(t *testing.T) {
	binaryFilePath := "./../protobuf_transfer/laptop.bin"
	laptop1 := sample.NewLaptop()

	err := serializer.WriteProtobufToBinaryFile(laptop1, binaryFilePath)
	require.NoError(t, err)

	var laptop2 protoc.Laptop
	err = serializer.ReadProtobufFromBinaryFile(&laptop2, binaryFilePath)
	require.NoError(t, err)
	require.Equal(t, laptop1.Id, laptop2.GetId())
	require.True(t, proto.Equal(laptop1, &laptop2))

	err = serializer.ReadProtobufFromBinaryFile(&laptop2, "not_exists")
	require.Error(t, err)

	jsonFilePath := "./../protobuf_transfer/laptop.json"
	err = serializer.WriteProtobufToJSONFile(laptop1, jsonFilePath)
	require.NoError(t, err)
}

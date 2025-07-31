package serializer_test

import (
	"testing"

	"github.com/go-http-server/grpc/sample"
	"github.com/go-http-server/grpc/serializer"
	"github.com/stretchr/testify/require"
)

func TestFileSerializer(t *testing.T) {
	binaryFilePath := "./../binary/laptop.bin"
	laptop1 := sample.NewLaptop()

	err := serializer.WriteProtobufToBinaryFile(laptop1, "../not_contains/trash_test.bin")
	require.Error(t, err)
	err = serializer.WriteProtobufToBinaryFile(laptop1, binaryFilePath)
	require.NoError(t, err)
}

// Package serializer write protobuf messages to compare transfer binary and JSON
package serializer

import (
	"os"

	"google.golang.org/protobuf/proto"
)

func WriteProtobufToBinaryFile(message proto.Message, filename string) error {
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

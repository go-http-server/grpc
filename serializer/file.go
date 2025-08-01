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

func WriteProtobufToJSONFile(message proto.Message, filename string) error {
	data, err := ProtobufToJSON(message)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}

func ReadProtobufFromBinaryFile(message proto.Message, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	return proto.Unmarshal(data, message)
}

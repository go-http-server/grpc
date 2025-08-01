package serializer

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func ProtobufToJSON(message proto.Message) ([]byte, error) {
	marshaler := protojson.MarshalOptions{
		UseEnumNumbers:    false,
		EmitDefaultValues: true,
		Indent:            "  ",
		UseProtoNames:     true,
	}

	return marshaler.Marshal(message)
}

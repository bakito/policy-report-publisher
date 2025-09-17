package ingest

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

// jsonCodec is a simple JSON codec to use with gRPC without requiring .proto generated code.
// The client must also use the same codec name ("json").
type jsonCodec struct{}

func (jsonCodec) Name() string { return "json" }

func (jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// RegisterJSONCodec registers the json codec globally.
// Call once during init of the gRPC server and any gRPC clients using this codec.
func RegisterJSONCodec() {
	encoding.RegisterCodec(jsonCodec{})
}

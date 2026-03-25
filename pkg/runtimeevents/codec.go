package runtimeevents

import (
	"fmt"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const SchemaVersionV1 = 1

func Normalize(event *runtimev1.RuntimeEventV1) (*runtimev1.RuntimeEventV1, error) {
	if event == nil {
		return nil, fmt.Errorf("runtime event is nil")
	}
	if event.SchemaVersion == 0 {
		event.SchemaVersion = SchemaVersionV1
	}
	return event, nil
}

func MarshalBinary(event *runtimev1.RuntimeEventV1) ([]byte, error) {
	event, err := Normalize(event)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(event)
}

func UnmarshalBinary(data []byte) (*runtimev1.RuntimeEventV1, error) {
	event := &runtimev1.RuntimeEventV1{}
	if err := proto.Unmarshal(data, event); err != nil {
		return nil, err
	}
	return Normalize(event)
}

func MarshalJSON(event *runtimev1.RuntimeEventV1) ([]byte, error) {
	event, err := Normalize(event)
	if err != nil {
		return nil, err
	}
	return protojson.MarshalOptions{
		UseProtoNames:   false,
		EmitUnpopulated: false,
	}.Marshal(event)
}

func UnmarshalJSON(data []byte) (*runtimev1.RuntimeEventV1, error) {
	event := &runtimev1.RuntimeEventV1{}
	if err := (protojson.UnmarshalOptions{
		DiscardUnknown: false,
	}).Unmarshal(data, event); err != nil {
		return nil, err
	}
	return Normalize(event)
}

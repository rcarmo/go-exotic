package protocol

import (
	"reflect"
	"testing"

	"github.com/rcarmo/go-exotic/internal/exotic"
)

func TestShardExecutionHTTPBridgeRoundTrip(t *testing.T) {
	req := ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 1}, Position: 2, HiddenSize: 2, Activation: []float32{1.5, -2}}
	wire, err := NewShardExecutionHTTPBridgeRequest(req)
	if err != nil {
		t.Fatalf("request wire: %v", err)
	}
	gotReq, err := wire.ShardExecutionRequest()
	if err != nil {
		t.Fatalf("request decode: %v", err)
	}
	if !reflect.DeepEqual(gotReq, req) {
		t.Fatalf("request roundtrip=%+v want %+v", gotReq, req)
	}
	resp := ShardExecutionResponse{SessionID: "s", RequestID: "r", PeerID: "p", Position: 2, HiddenSize: 2, Activation: []float32{3, 4}}
	respWire, err := NewShardExecutionHTTPBridgeResponse(resp)
	if err != nil {
		t.Fatalf("response wire: %v", err)
	}
	gotResp, err := respWire.ShardExecutionResponse()
	if err != nil {
		t.Fatalf("response decode: %v", err)
	}
	if !reflect.DeepEqual(gotResp, resp) {
		t.Fatalf("response roundtrip=%+v want %+v", gotResp, resp)
	}
}

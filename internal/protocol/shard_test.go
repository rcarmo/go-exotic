package protocol

import (
	"encoding/json"
	"testing"

	"github.com/rcarmo/go-exotic/internal/exotic"
)

func TestShardExecutionRequestValidationAndJSON(t *testing.T) {
	req := ShardExecutionRequest{
		SessionID:  "s",
		RequestID:  "r",
		ModelID:    "m",
		Shard:      exotic.Shard{DeviceID: "peer", StartLayer: 0, EndLayer: 1},
		Position:   3,
		HiddenSize: 2,
		Activation: []float32{1, 2},
	}
	if err := req.Validate(4); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ShardExecutionRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := decoded.Validate(4); err != nil {
		t.Fatalf("decoded Validate: %v", err)
	}
}

func TestShardExecutionRequestRejectsMalformedInputs(t *testing.T) {
	base := ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "peer", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 2, Activation: []float32{1, 2}}
	cases := []ShardExecutionRequest{
		{},
		func() ShardExecutionRequest { v := base; v.SessionID = ""; return v }(),
		func() ShardExecutionRequest { v := base; v.RequestID = ""; return v }(),
		func() ShardExecutionRequest { v := base; v.ModelID = ""; return v }(),
		func() ShardExecutionRequest { v := base; v.Shard.DeviceID = ""; return v }(),
		func() ShardExecutionRequest { v := base; v.Position = -1; return v }(),
		func() ShardExecutionRequest { v := base; v.HiddenSize = 0; return v }(),
		func() ShardExecutionRequest { v := base; v.Activation = []float32{1}; return v }(),
	}
	for i, tc := range cases {
		if err := tc.Validate(1); err == nil {
			t.Fatalf("case %d accepted malformed request: %+v", i, tc)
		}
	}
}

func TestShardExecutionResponseValidation(t *testing.T) {
	resp := ShardExecutionResponse{SessionID: "s", RequestID: "r", PeerID: "p", Position: 1, HiddenSize: 2, Activation: []float32{3, 4}}
	if err := resp.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	bad := []ShardExecutionResponse{
		{},
		{SessionID: "s", RequestID: "r", PeerID: "", Position: 0, HiddenSize: 1, Activation: []float32{1}},
		{SessionID: "s", RequestID: "r", PeerID: "p", Position: -1, HiddenSize: 1, Activation: []float32{1}},
		{SessionID: "s", RequestID: "r", PeerID: "p", Position: 0, HiddenSize: 2, Activation: []float32{1}},
	}
	for i, tc := range bad {
		if err := tc.Validate(); err == nil {
			t.Fatalf("case %d accepted malformed response: %+v", i, tc)
		}
	}
}

package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
	pherencemodel "github.com/rcarmo/go-pherence/model"
)

func TestPherenceLayerExecutorValidation(t *testing.T) {
	if _, err := NewPherenceLayerExecutor(nil); err == nil {
		t.Fatal("accepted nil model")
	}
	if _, err := NewPherenceLayerExecutor(&pherencemodel.LlamaModel{}); err == nil {
		t.Fatal("accepted invalid model dims")
	}
	model := &pherencemodel.LlamaModel{Config: pherencemodel.LlamaConfig{HiddenSize: 2, NumLayers: 0}}
	exec, err := NewPherenceLayerExecutor(model)
	if err != nil {
		t.Fatalf("NewPherenceLayerExecutor: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = exec.ExecuteLayerRange(ctx, protocol.ShardExecutionRequest{}, nil, nil, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
	req := protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 1, Activation: []float32{1}}
	if _, err := exec.ExecuteLayerRange(context.Background(), req, nil, nil, 1); err == nil {
		t.Fatal("accepted hidden-size mismatch")
	}
	req.HiddenSize = 2
	req.Activation = []float32{1, 2}
	if _, err := exec.ExecuteLayerRange(context.Background(), req, nil, nil, 1); err == nil {
		t.Fatal("accepted bad KV cache shape")
	}
}

func TestPherenceLayerExecutorZeroLayerRange(t *testing.T) {
	model := &pherencemodel.LlamaModel{Config: pherencemodel.LlamaConfig{HiddenSize: 2, NumLayers: 0}}
	exec, err := NewPherenceLayerExecutor(model)
	if err != nil {
		t.Fatalf("NewPherenceLayerExecutor: %v", err)
	}
	// There are no valid layer ranges for a zero-layer model; request validation should reject it.
	req := protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 2, Activation: []float32{1, 2}}
	if _, err := exec.ExecuteLayerRange(context.Background(), req, nil, nil, 0); err == nil {
		t.Fatal("accepted zero total layers")
	}
}

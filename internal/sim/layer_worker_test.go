package sim

import (
	"context"
	"errors"
	"testing"

	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

type fakeLayerRangeExecutor struct {
	wantStart int
	wantEnd   int
}

func (f fakeLayerRangeExecutor) ExecuteLayerRange(ctx context.Context, req protocol.ShardExecutionRequest, kvCacheK, kvCacheV [][]float32, totalLayers int) (protocol.ShardExecutionResponse, error) {
	if err := ctx.Err(); err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	if req.Shard.StartLayer != f.wantStart || req.Shard.EndLayer != f.wantEnd {
		return protocol.ShardExecutionResponse{}, errors.New("unexpected shard range")
	}
	if err := req.Validate(totalLayers); err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	out := append([]float32(nil), req.Activation...)
	for i := range out {
		out[i] += float32(req.Shard.EndLayer - req.Shard.StartLayer + 1)
	}
	return protocol.ShardExecutionResponse{SessionID: req.SessionID, RequestID: req.RequestID, PeerID: req.Shard.DeviceID, Position: req.Position, HiddenSize: req.HiddenSize, Activation: out}, nil
}

func TestLayerExecutorWorkerSingleLayerValidation(t *testing.T) {
	worker := &LayerExecutorWorker{Executor: fakeLayerRangeExecutor{wantStart: 0, wantEnd: 0}, TotalLayers: 2}
	resp, err := worker.ExecuteShard(context.Background(), protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 2, Activation: []float32{1, 2}})
	if err != nil {
		t.Fatalf("ExecuteShard: %v", err)
	}
	if resp.Activation[0] != 2 || resp.Activation[1] != 3 {
		t.Fatalf("activation=%v want [2 3]", resp.Activation)
	}
}

func TestLayerExecutorWorkerMultiLayerRangeValidation(t *testing.T) {
	worker := &LayerExecutorWorker{Executor: fakeLayerRangeExecutor{wantStart: 1, wantEnd: 3}, TotalLayers: 4}
	resp, err := worker.ExecuteShard(context.Background(), protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "p", StartLayer: 1, EndLayer: 3}, Position: 0, HiddenSize: 1, Activation: []float32{5}})
	if err != nil {
		t.Fatalf("ExecuteShard: %v", err)
	}
	if resp.Activation[0] != 8 {
		t.Fatalf("activation=%v want [8]", resp.Activation)
	}
}

func TestLayerExecutorWorkerRejectsMalformedInputs(t *testing.T) {
	if _, err := (*LayerExecutorWorker)(nil).ExecuteShard(context.Background(), protocol.ShardExecutionRequest{}); err == nil {
		t.Fatal("accepted nil worker")
	}
	worker := &LayerExecutorWorker{TotalLayers: 1}
	if _, err := worker.ExecuteShard(context.Background(), protocol.ShardExecutionRequest{}); err == nil {
		t.Fatal("accepted nil executor")
	}
	worker.Executor = fakeLayerRangeExecutor{wantStart: 0, wantEnd: 0}
	bad := protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 2, Activation: []float32{1}}
	if _, err := worker.ExecuteShard(context.Background(), bad); err == nil {
		t.Fatal("accepted malformed activation shape")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bad.Activation = []float32{1, 2}
	if _, err := worker.ExecuteShard(ctx, bad); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
}

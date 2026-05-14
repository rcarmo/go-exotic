package sim

import (
	"context"
	"errors"
	"testing"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
)

func TestSimulatorExecute(t *testing.T) {
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	peerB, _ := cluster.LocalPeer("b", "http://127.0.0.1:2", 1)
	routes := []router.Route{{Peer: peerA, Shard: exotic.Shard{DeviceID: "a", StartLayer: 0, EndLayer: 0}}, {Peer: peerB, Shard: exotic.Shard{DeviceID: "b", StartLayer: 1, EndLayer: 1}}}
	worker := WorkerFunc(func(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		out := append([]float32(nil), req.Activation...)
		for i := range out {
			out[i] += float32(req.Shard.StartLayer + 1)
		}
		return protocol.ShardExecutionResponse{SessionID: req.SessionID, RequestID: req.RequestID, PeerID: req.Shard.DeviceID, Position: req.Position, HiddenSize: req.HiddenSize, Activation: out}, nil
	})
	s, err := NewSimulator(PeerWorkers([]cluster.Peer{peerA, peerB}, worker))
	if err != nil {
		t.Fatalf("NewSimulator: %v", err)
	}
	resp, err := s.Execute(context.Background(), routes, protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Position: 0, HiddenSize: 2, Activation: []float32{1, 2}}, 2)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Activation) != 2 || resp.Activation[0] != 4 || resp.Activation[1] != 5 {
		t.Fatalf("activation=%v want [4 5]", resp.Activation)
	}
}

func TestSimulatorRejectsResponsePeerMismatch(t *testing.T) {
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	s, err := NewSimulator(map[string]Worker{"a": WorkerFunc(func(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		return protocol.ShardExecutionResponse{SessionID: req.SessionID, RequestID: req.RequestID, PeerID: "other", Position: req.Position, HiddenSize: req.HiddenSize, Activation: append([]float32(nil), req.Activation...)}, nil
	})})
	if err != nil {
		t.Fatalf("NewSimulator: %v", err)
	}
	_, err = s.Execute(context.Background(), []router.Route{{Peer: peerA, Shard: exotic.Shard{DeviceID: "a", StartLayer: 0, EndLayer: 0}}}, protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Position: 0, HiddenSize: 1, Activation: []float32{1}}, 1)
	if err == nil {
		t.Fatal("accepted response from wrong peer")
	}
}

func TestSimulatorRejectsMalformedInputs(t *testing.T) {
	if _, err := NewSimulator(nil); err == nil {
		t.Fatal("accepted no workers")
	}
	if _, err := NewSimulator(map[string]Worker{"": WorkerFunc(nil)}); err == nil {
		t.Fatal("accepted empty worker id")
	}
	if _, err := NewSimulator(map[string]Worker{"a": nil}); err == nil {
		t.Fatal("accepted nil worker")
	}
	if _, err := NewSimulator(map[string]Worker{"a": WorkerFunc(nil)}); err == nil {
		t.Fatal("accepted nil WorkerFunc")
	}
	var nilLayerWorker *LayerExecutorWorker
	if _, err := NewSimulator(map[string]Worker{"a": nilLayerWorker}); err == nil {
		t.Fatal("accepted nil pointer worker")
	}
	if _, err := (*Simulator)(nil).Execute(context.Background(), nil, protocol.ShardExecutionRequest{}, 1); err == nil {
		t.Fatal("nil simulator accepted")
	}
	s, _ := NewSimulator(map[string]Worker{"a": WorkerFunc(func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		return protocol.ShardExecutionResponse{}, errors.New("boom")
	})})
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	if _, err := s.Execute(context.Background(), []router.Route{{Peer: peerA, Shard: exotic.Shard{DeviceID: "a", StartLayer: 0, EndLayer: 0}}}, protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Position: 0, HiddenSize: 1, Activation: []float32{1}}, 1); err == nil {
		t.Fatal("worker error ignored")
	}
}

func TestSimulatorPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s, _ := NewSimulator(map[string]Worker{"a": WorkerFunc(func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		return protocol.ShardExecutionResponse{}, nil
	})})
	if _, err := s.Execute(ctx, nil, protocol.ShardExecutionRequest{}, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
}

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

func TestGenerateTokensThroughNShards(t *testing.T) {
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
	tokens, err := s.GenerateTokens(context.Background(), routes, protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Position: 0, HiddenSize: 1, Activation: []float32{0}}, 2, 3, func(act []float32) (int, error) {
		return int(act[0]), nil
	})
	if err != nil {
		t.Fatalf("GenerateTokens: %v", err)
	}
	want := []int{3, 6, 9}
	for i := range want {
		if tokens[i] != want[i] {
			t.Fatalf("tokens=%v want %v", tokens, want)
		}
	}
}

func TestGenerateTokensRejectsMalformedInputs(t *testing.T) {
	s, _ := NewSimulator(map[string]Worker{"a": WorkerFunc(func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		return protocol.ShardExecutionResponse{}, nil
	})})
	base := protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Position: 0, HiddenSize: 1, Activation: []float32{1}}
	if _, err := (*Simulator)(nil).GenerateTokens(context.Background(), nil, base, 1, 1, func([]float32) (int, error) { return 0, nil }); err == nil {
		t.Fatal("accepted nil simulator")
	}
	if _, err := s.GenerateTokens(context.Background(), nil, base, 1, -1, func([]float32) (int, error) { return 0, nil }); err == nil {
		t.Fatal("accepted negative maxTokens")
	}
	maxInt := int(^uint(0) >> 1)
	badPos := base
	badPos.Position = maxInt
	if _, err := s.GenerateTokens(context.Background(), nil, badPos, 1, 2, func([]float32) (int, error) { return 0, nil }); err == nil {
		t.Fatal("accepted overflowing generation positions")
	}
	if _, err := s.GenerateTokens(context.Background(), nil, base, 1, 1, nil); err == nil {
		t.Fatal("accepted nil projector")
	}
	bad := base
	bad.Activation = nil
	if _, err := s.GenerateTokens(context.Background(), nil, bad, 1, 1, func([]float32) (int, error) { return 0, nil }); err == nil {
		t.Fatal("accepted bad activation shape")
	}
}

func TestGenerateTokensPropagatesCancellationAndProjectorErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s, _ := NewSimulator(map[string]Worker{"a": WorkerFunc(func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		return protocol.ShardExecutionResponse{}, nil
	})})
	base := protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Position: 0, HiddenSize: 1, Activation: []float32{1}}
	if _, err := s.GenerateTokens(ctx, nil, base, 1, 1, func([]float32) (int, error) { return 0, nil }); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
	peer, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	route := []router.Route{{Peer: peer, Shard: exotic.Shard{DeviceID: "a", StartLayer: 0, EndLayer: 0}}}
	worker := WorkerFunc(func(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		return protocol.ShardExecutionResponse{SessionID: req.SessionID, RequestID: req.RequestID, PeerID: "a", Position: req.Position, HiddenSize: req.HiddenSize, Activation: req.Activation}, nil
	})
	s, _ = NewSimulator(map[string]Worker{"a": worker})
	boom := errors.New("boom")
	if _, err := s.GenerateTokens(context.Background(), route, base, 1, 1, func([]float32) (int, error) { return 0, boom }); !errors.Is(err, boom) {
		t.Fatalf("err=%v want boom", err)
	}
}

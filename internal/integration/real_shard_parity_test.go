package integration

import (
	"context"
	"math"
	"testing"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
	exoticruntime "github.com/rcarmo/go-exotic/internal/runtime"
	"github.com/rcarmo/go-exotic/internal/sim"
	pherencemodel "github.com/rcarmo/go-pherence/model"
)

func TestRealShardOneTokenOutputMatchesSequentialForwardLayer(t *testing.T) {
	modelPath := smolLM2FixturePath(t)
	m, err := pherencemodel.LoadLlama(modelPath)
	if err != nil {
		t.Fatalf("LoadLlama: %v", err)
	}
	if m.Config.NumLayers < 2 {
		t.Skipf("need at least two layers, got %d", m.Config.NumLayers)
	}
	exec, err := exoticruntime.NewPherenceLayerExecutor(m)
	if err != nil {
		t.Fatalf("NewPherenceLayerExecutor: %v", err)
	}
	hidden := make([]float32, m.Config.HiddenSize)
	if err := m.ScaledTokenEmbeddingInto(hidden, 0); err != nil {
		t.Fatalf("ScaledTokenEmbeddingInto: %v", err)
	}
	want := append([]float32(nil), hidden...)
	kvK := make([][]float32, m.Config.NumLayers)
	kvV := make([][]float32, m.Config.NumLayers)
	for layer := 0; layer < m.Config.NumLayers; layer++ {
		want = m.ForwardLayer(want, layer, 0, 0, kvK, kvV)
		if want == nil {
			t.Fatalf("direct ForwardLayer(%d) returned nil", layer)
		}
	}

	mid := m.Config.NumLayers / 2
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	peerB, _ := cluster.LocalPeer("b", "http://127.0.0.1:2", 1)
	routes := []router.Route{
		{Peer: peerA, Shard: exotic.Shard{DeviceID: "a", StartLayer: 0, EndLayer: mid - 1}},
		{Peer: peerB, Shard: exotic.Shard{DeviceID: "b", StartLayer: mid, EndLayer: m.Config.NumLayers - 1}},
	}
	sharedK := make([][]float32, m.Config.NumLayers)
	sharedV := make([][]float32, m.Config.NumLayers)
	workers := map[string]sim.Worker{
		"a": &sim.LayerExecutorWorker{Executor: exec, KVCacheK: sharedK, KVCacheV: sharedV, TotalLayers: m.Config.NumLayers},
		"b": &sim.LayerExecutorWorker{Executor: exec, KVCacheK: sharedK, KVCacheV: sharedV, TotalLayers: m.Config.NumLayers},
	}
	s, err := sim.NewSimulator(workers)
	if err != nil {
		t.Fatalf("NewSimulator: %v", err)
	}
	resp, err := s.Execute(context.Background(), routes, protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "smollm2", Position: 0, HiddenSize: m.Config.HiddenSize, Activation: hidden}, m.Config.NumLayers)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Activation) != len(want) {
		t.Fatalf("activation len=%d want %d", len(resp.Activation), len(want))
	}
	for i := range want {
		if math.Abs(float64(resp.Activation[i]-want[i])) > 1e-6 {
			t.Fatalf("activation[%d]=%g want %g", i, resp.Activation[i], want[i])
		}
	}
	_, _, wantToken, err := m.FinishCPUDecodeStep(want)
	if err != nil {
		t.Fatalf("direct FinishCPUDecodeStep: %v", err)
	}
	_, _, gotToken, err := m.FinishCPUDecodeStep(resp.Activation)
	if err != nil {
		t.Fatalf("sharded FinishCPUDecodeStep: %v", err)
	}
	if gotToken != wantToken {
		t.Fatalf("token=%d want %d", gotToken, wantToken)
	}
}

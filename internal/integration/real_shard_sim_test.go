package integration

import (
	"context"
	"testing"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
	exoticruntime "github.com/rcarmo/go-exotic/internal/runtime"
	"github.com/rcarmo/go-exotic/internal/sim"
	pherencemodel "github.com/rcarmo/go-pherence/model"
	"github.com/rcarmo/go-pherence/tensor"
)

func TestSingleHostRealLayerExecutorMultiShardRun(t *testing.T) {
	m := syntheticTwoLayerModel()
	exec, err := exoticruntime.NewPherenceLayerExecutor(m)
	if err != nil {
		t.Fatalf("NewPherenceLayerExecutor: %v", err)
	}
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	peerB, _ := cluster.LocalPeer("b", "http://127.0.0.1:2", 1)
	routes := []router.Route{
		{Peer: peerA, Shard: exotic.Shard{DeviceID: "a", StartLayer: 0, EndLayer: 0}},
		{Peer: peerB, Shard: exotic.Shard{DeviceID: "b", StartLayer: 1, EndLayer: 1}},
	}
	workers := map[string]sim.Worker{
		"a": &sim.LayerExecutorWorker{Executor: exec, KVCacheK: make([][]float32, m.Config.NumLayers), KVCacheV: make([][]float32, m.Config.NumLayers), TotalLayers: m.Config.NumLayers},
		"b": &sim.LayerExecutorWorker{Executor: exec, KVCacheK: make([][]float32, m.Config.NumLayers), KVCacheV: make([][]float32, m.Config.NumLayers), TotalLayers: m.Config.NumLayers},
	}
	s, err := sim.NewSimulator(workers)
	if err != nil {
		t.Fatalf("NewSimulator: %v", err)
	}
	resp, err := s.Execute(context.Background(), routes, protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "synthetic", Position: 0, HiddenSize: 2, Activation: []float32{1, 0}}, m.Config.NumLayers)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(resp.Activation) != 2 {
		t.Fatalf("activation len=%d want 2", len(resp.Activation))
	}
}

func syntheticTwoLayerModel() *pherencemodel.LlamaModel {
	identity := []float32{1, 0, 0, 1}
	layers := make([]pherencemodel.LlamaLayer, 2)
	for i := range layers {
		layers[i] = pherencemodel.LlamaLayer{
			InputNorm: tensor.Ones([]int{2}),
			PostNorm:  tensor.Ones([]int{2}),
			HasKV:     true,
			QW:        tensor.FromFloat32(append([]float32(nil), identity...), []int{2, 2}),
			KW:        tensor.FromFloat32(append([]float32(nil), identity...), []int{2, 2}),
			VW:        tensor.FromFloat32(append([]float32(nil), identity...), []int{2, 2}),
			OW:        tensor.FromFloat32(append([]float32(nil), identity...), []int{2, 2}),
			GateW:     tensor.FromFloat32(append([]float32(nil), identity...), []int{2, 2}),
			UpW:       tensor.FromFloat32(append([]float32(nil), identity...), []int{2, 2}),
			DownW:     tensor.FromFloat32(append([]float32(nil), identity...), []int{2, 2}),
		}
	}
	return &pherencemodel.LlamaModel{Config: pherencemodel.LlamaConfig{VocabSize: 2, HiddenSize: 2, NumLayers: 2, NumHeads: 1, NumKVHeads: 1, HeadDim: 2, Intermediate: 2, RMSNormEps: 1e-6}, Layers: layers}
}

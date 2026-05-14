package runtime

import (
	"context"
	"fmt"

	"github.com/rcarmo/go-exotic/internal/protocol"
	pherencemodel "github.com/rcarmo/go-pherence/model"
)

// PherenceLayerExecutor executes layer ranges in-process with go-pherence's
// CPU ForwardLayer hook. It is intentionally local-only and does not perform
// final norm/LM-head work.
type PherenceLayerExecutor struct {
	model *pherencemodel.LlamaModel
}

func NewPherenceLayerExecutor(model *pherencemodel.LlamaModel) (*PherenceLayerExecutor, error) {
	if model == nil {
		return nil, fmt.Errorf("nil model")
	}
	cfg := model.Config
	if cfg.HiddenSize <= 0 || cfg.NumLayers < 0 || len(model.Layers) < cfg.NumLayers {
		return nil, fmt.Errorf("invalid model dims hidden=%d layers=%d/%d", cfg.HiddenSize, cfg.NumLayers, len(model.Layers))
	}
	return &PherenceLayerExecutor{model: model}, nil
}

func (e *PherenceLayerExecutor) ExecuteLayerRange(ctx context.Context, req protocol.ShardExecutionRequest, kvCacheK, kvCacheV [][]float32, totalLayers int) (protocol.ShardExecutionResponse, error) {
	if e == nil || e.model == nil {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("nil layer executor")
	}
	if err := ctx.Err(); err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	if err := req.Validate(totalLayers); err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	if req.HiddenSize != e.model.Config.HiddenSize {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("request hidden size=%d, model hidden size=%d", req.HiddenSize, e.model.Config.HiddenSize)
	}
	if len(kvCacheK) != e.model.Config.NumLayers || len(kvCacheV) != e.model.Config.NumLayers {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("KV cache layers K/V=%d/%d, want %d", len(kvCacheK), len(kvCacheV), e.model.Config.NumLayers)
	}
	hidden := append([]float32(nil), req.Activation...)
	for layer := req.Shard.StartLayer; layer <= req.Shard.EndLayer; layer++ {
		if err := ctx.Err(); err != nil {
			return protocol.ShardExecutionResponse{}, err
		}
		hidden = e.model.ForwardLayer(hidden, layer, req.Position, req.Position, kvCacheK, kvCacheV)
		if hidden == nil {
			return protocol.ShardExecutionResponse{}, fmt.Errorf("forward layer %d failed", layer)
		}
	}
	return protocol.ShardExecutionResponse{SessionID: req.SessionID, RequestID: req.RequestID, PeerID: req.Shard.DeviceID, Position: req.Position, HiddenSize: req.HiddenSize, Activation: append([]float32(nil), hidden...)}, nil
}

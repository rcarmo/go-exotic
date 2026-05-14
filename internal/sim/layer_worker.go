package sim

import (
	"context"
	"fmt"

	"github.com/rcarmo/go-exotic/internal/protocol"
	exoticruntime "github.com/rcarmo/go-exotic/internal/runtime"
)

// LayerExecutorWorker adapts a runtime.LayerRangeExecutor to the simulator's
// Worker interface. It keeps KV caches local to the worker, which matches the
// current single-host simulation boundary.
type LayerExecutorWorker struct {
	Executor    exoticruntime.LayerRangeExecutor
	KVCacheK    [][]float32
	KVCacheV    [][]float32
	TotalLayers int
}

func (w *LayerExecutorWorker) ExecuteShard(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
	if w == nil {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("nil layer executor worker")
	}
	if w.Executor == nil {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("nil layer executor")
	}
	return w.Executor.ExecuteLayerRange(ctx, req, w.KVCacheK, w.KVCacheV, w.TotalLayers)
}

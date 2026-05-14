package runtime

import (
	"context"

	"github.com/rcarmo/go-exotic/internal/protocol"
)

// Metadata is the backend-neutral model information needed by placement.
type Metadata struct {
	ModelPath  string
	Layers     int
	HiddenSize int
	VocabSize  int
	ModelType  string
}

// Adapter is the minimal local runtime contract. The first implementation wraps
// go-pherence full-model loading/generation before shard execution exists.
type Adapter interface {
	Metadata(ctx context.Context, modelPath string) (Metadata, error)
	Tokenize(ctx context.Context, modelPath, text string) ([]int, error)
	Generate(ctx context.Context, modelPath, prompt string, maxTokens int) ([]int, error)
}

// LayerShardExecutor is intentionally not implemented yet. It reserves the
// shape of the future distributed runtime boundary without enabling it.
type LayerShardExecutor interface {
	ExecuteLayerShard(ctx context.Context, request protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error)
}

package runtime

import "context"

// Metadata is the backend-neutral model information needed by placement.
type Metadata struct {
	ModelPath  string
	Layers     int
	HiddenSize int
	VocabSize  int
}

// Adapter is the minimal local runtime contract. The first implementation will
// wrap go-pherence full-model loading/generation before shard execution exists.
type Adapter interface {
	Metadata(ctx context.Context, modelPath string) (Metadata, error)
	Generate(ctx context.Context, modelPath, prompt string, maxTokens int) ([]int, error)
}

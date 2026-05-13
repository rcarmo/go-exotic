package runtime

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rcarmo/go-pherence/loader/tokenizer"
	pherencemodel "github.com/rcarmo/go-pherence/model"
)

// PherenceAdapter is the local-only runtime adapter backed by go-pherence.
type PherenceAdapter struct{}

func NewPherenceAdapter() PherenceAdapter { return PherenceAdapter{} }

func (PherenceAdapter) Metadata(ctx context.Context, modelPath string) (Metadata, error) {
	if err := ctx.Err(); err != nil {
		return Metadata{}, err
	}
	m, err := loadLocalModel(modelPath)
	if err != nil {
		return Metadata{}, err
	}
	cfg := m.Config
	return Metadata{ModelPath: modelPath, Layers: cfg.NumLayers, HiddenSize: cfg.HiddenSize, VocabSize: cfg.VocabSize, ModelType: cfg.ModelType}, nil
}

func (PherenceAdapter) Tokenize(ctx context.Context, modelPath, text string) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tok, err := loadTokenizer(modelPath)
	if err != nil {
		return nil, err
	}
	return tok.Encode(text), nil
}

func (PherenceAdapter) Generate(ctx context.Context, modelPath, prompt string, maxTokens int) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if maxTokens < 0 {
		return nil, fmt.Errorf("maxTokens %d out of range", maxTokens)
	}
	m, err := loadLocalModel(modelPath)
	if err != nil {
		return nil, err
	}
	tok, err := loadTokenizer(modelPath)
	if err != nil {
		return nil, err
	}
	tokens := tok.Encode(prompt)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return m.Generate(tokens, maxTokens), nil
}

func loadLocalModel(modelPath string) (*pherencemodel.LlamaModel, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("empty model path")
	}
	return pherencemodel.LoadLlama(modelPath)
}

func loadTokenizer(modelPath string) (*tokenizer.Tokenizer, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("empty model path")
	}
	return tokenizer.Load(filepath.Join(modelPath, "tokenizer.json"))
}

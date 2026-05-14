package runtime

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strings"

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
	modelPath, err := cleanModelPath(modelPath)
	if err != nil {
		return Metadata{}, err
	}
	m, err := loadLocalModel(modelPath)
	if err != nil {
		return Metadata{}, err
	}
	cfg := m.Config
	meta := Metadata{ModelPath: modelPath, Layers: cfg.NumLayers, HiddenSize: cfg.HiddenSize, VocabSize: cfg.VocabSize, ModelType: cfg.ModelType}
	if err := validateMetadata(meta); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

func (PherenceAdapter) Tokenize(ctx context.Context, modelPath, text string) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	modelPath, err := cleanModelPath(modelPath)
	if err != nil {
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
	modelPath, err := cleanModelPath(modelPath)
	if err != nil {
		return nil, err
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

func LoadLocalModel(modelPath string) (*pherencemodel.LlamaModel, error) {
	return loadLocalModel(modelPath)
}

func loadLocalModel(modelPath string) (*pherencemodel.LlamaModel, error) {
	modelPath, err := cleanModelPath(modelPath)
	if err != nil {
		return nil, err
	}
	return pherencemodel.LoadLlama(modelPath)
}

func loadTokenizer(modelPath string) (*tokenizer.Tokenizer, error) {
	modelPath, err := cleanModelPath(modelPath)
	if err != nil {
		return nil, err
	}
	return tokenizer.Load(filepath.Join(modelPath, "tokenizer.json"))
}

func cleanModelPath(modelPath string) (string, error) {
	modelPath = strings.TrimSpace(modelPath)
	if modelPath == "" {
		return "", fmt.Errorf("empty model path")
	}
	return modelPath, nil
}

func validateMetadata(meta Metadata) error {
	if strings.TrimSpace(meta.ModelPath) == "" {
		return fmt.Errorf("empty metadata model path")
	}
	if meta.Layers < 0 || meta.HiddenSize <= 0 || meta.VocabSize <= 0 {
		return fmt.Errorf("invalid metadata dims layers=%d hidden=%d vocab=%d", meta.Layers, meta.HiddenSize, meta.VocabSize)
	}
	if meta.Layers > math.MaxInt/2 {
		return fmt.Errorf("metadata layers %d out of range", meta.Layers)
	}
	return nil
}

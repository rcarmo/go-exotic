package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rcarmo/go-exotic/internal/runtime"
	"github.com/rcarmo/go-pherence/loader/tokenizer"
	pherencemodel "github.com/rcarmo/go-pherence/model"
)

func TestLocalRuntimeAdapterMatchesDirectGoPherenceGeneration(t *testing.T) {
	modelPath := filepath.Clean("../../../go-pherence/models/smollm2-135m")
	if _, err := os.Stat(filepath.Join(modelPath, "config.json")); err != nil {
		t.Skipf("SmolLM2 fixture unavailable: %v", err)
	}
	ctx := context.Background()
	adapter := runtime.NewPherenceAdapter()
	got, err := adapter.Generate(ctx, modelPath, "Hello", 1)
	if err != nil {
		t.Fatalf("adapter Generate: %v", err)
	}
	m, err := pherencemodel.LoadLlama(modelPath)
	if err != nil {
		t.Fatalf("LoadLlama: %v", err)
	}
	tok, err := tokenizer.Load(filepath.Join(modelPath, "tokenizer.json"))
	if err != nil {
		t.Fatalf("tokenizer Load: %v", err)
	}
	want := m.Generate(tok.Encode("Hello"), 1)
	if len(got) != len(want) {
		t.Fatalf("len got=%d want=%d got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d got=%d want=%d got=%v want=%v", i, got[i], want[i], got, want)
		}
	}
}

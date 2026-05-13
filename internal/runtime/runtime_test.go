package runtime

import (
	"context"
	"testing"
)

func TestPherenceAdapterValidation(t *testing.T) {
	adapter := NewPherenceAdapter()
	if _, err := adapter.Metadata(context.Background(), ""); err == nil {
		t.Fatal("Metadata accepted empty model path")
	}
	if _, err := adapter.Generate(context.Background(), "", "hi", 1); err == nil {
		t.Fatal("Generate accepted empty model path")
	}
	if _, err := adapter.Generate(context.Background(), "missing", "hi", -1); err == nil {
		t.Fatal("Generate accepted negative maxTokens")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := adapter.Metadata(ctx, "missing"); err == nil {
		t.Fatal("Metadata ignored cancelled context")
	}
	if _, err := adapter.Tokenize(ctx, "missing", "hi"); err == nil {
		t.Fatal("Tokenize ignored cancelled context")
	}
	if _, err := adapter.Generate(ctx, "missing", "hi", 1); err == nil {
		t.Fatal("Generate ignored cancelled context")
	}
}

func TestRuntimeValidationHelpers(t *testing.T) {
	if _, err := cleanModelPath(" \t\n"); err == nil {
		t.Fatal("cleanModelPath accepted whitespace-only path")
	}
	if got, err := cleanModelPath(" ./model "); err != nil || got != "./model" {
		t.Fatalf("cleanModelPath=%q,%v want ./model,nil", got, err)
	}
	if err := validateMetadata(Metadata{ModelPath: "m", Layers: 1, HiddenSize: 2, VocabSize: 3}); err != nil {
		t.Fatalf("validateMetadata: %v", err)
	}
	bad := []Metadata{
		{},
		{ModelPath: "m", Layers: -1, HiddenSize: 2, VocabSize: 3},
		{ModelPath: "m", Layers: 1, HiddenSize: 0, VocabSize: 3},
		{ModelPath: "m", Layers: 1, HiddenSize: 2, VocabSize: 0},
	}
	for i, meta := range bad {
		if err := validateMetadata(meta); err == nil {
			t.Fatalf("case %d accepted bad metadata: %+v", i, meta)
		}
	}
}

func TestPherenceAdapterImplementsAdapter(t *testing.T) {
	var _ Adapter = NewPherenceAdapter()
}

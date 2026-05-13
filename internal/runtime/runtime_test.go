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

func TestPherenceAdapterImplementsAdapter(t *testing.T) {
	var _ Adapter = NewPherenceAdapter()
}

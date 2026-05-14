package router

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rcarmo/go-exotic/internal/cluster"
)

func TestPreviewFromRegistry(t *testing.T) {
	r := cluster.NewRegistry()
	a, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	b, _ := cluster.LocalPeer("b", "http://127.0.0.1:2", 3)
	if err := r.Upsert(a); err != nil {
		t.Fatal(err)
	}
	if err := r.Upsert(b); err != nil {
		t.Fatal(err)
	}
	preview, err := PreviewFromRegistry(context.Background(), r, "model", 4)
	if err != nil {
		t.Fatalf("PreviewFromRegistry: %v", err)
	}
	if preview.ModelID != "model" || preview.Layers != 4 || len(preview.Routes) != 2 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if preview.Routes[0].Shard.StartLayer != 0 || preview.Routes[1].Shard.EndLayer != 3 {
		t.Fatalf("unexpected shard coverage: %+v", preview.Routes)
	}
}

func TestPreviewFromRegistryAfterStaleEviction(t *testing.T) {
	r := cluster.NewRegistry()
	old, _ := cluster.LocalPeer("old", "http://127.0.0.1:1", 1)
	if err := r.Upsert(old); err != nil {
		t.Fatal(err)
	}
	evicted := r.EvictStale(time.Nanosecond)
	if len(evicted) == 0 {
		t.Fatal("expected stale eviction")
	}
	if _, err := PreviewFromRegistry(context.Background(), r, "model", 1); err == nil {
		t.Fatal("accepted empty registry after eviction")
	}
}

func TestPreviewFromRegistryRejectsMalformedInputs(t *testing.T) {
	if _, err := PreviewFromRegistry(context.Background(), nil, "model", 1); err == nil {
		t.Fatal("accepted nil registry")
	}
	r := cluster.NewRegistry()
	peer, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	if err := r.Upsert(peer); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewFromRegistry(context.Background(), r, "model", 0); err == nil {
		t.Fatal("accepted bad layer count")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := PreviewFromRegistry(ctx, r, "model", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
}

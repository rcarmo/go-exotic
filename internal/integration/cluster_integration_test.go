package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/placement"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
	"github.com/rcarmo/go-exotic/internal/server"
)

func TestInProcessMultiNodePlacementAndRouting(t *testing.T) {
	peerA, err := cluster.LocalPeer("a", "http://127.0.0.1:1", 3)
	if err != nil {
		t.Fatalf("LocalPeer a: %v", err)
	}
	peerB, err := cluster.LocalPeer("b", "http://127.0.0.1:2", 1)
	if err != nil {
		t.Fatalf("LocalPeer b: %v", err)
	}
	capA, err := peerA.Capability()
	if err != nil {
		t.Fatalf("cap a: %v", err)
	}
	capB, err := peerB.Capability()
	if err != nil {
		t.Fatalf("cap b: %v", err)
	}
	ts := httptest.NewServer(server.New([]protocol.Capability{capA, capB}).Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/placement/preview?layers=8&model=demo")
	if err != nil {
		t.Fatalf("GET placement: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	var preview protocol.PlacementPreview
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		t.Fatalf("decode placement: %v", err)
	}
	if preview.Layers != 8 || len(preview.Shards) != 2 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	if err := placement.ValidatePlan(preview.Shards, preview.Layers); err != nil {
		t.Fatalf("invalid preview plan: %v", err)
	}

	routes, err := router.BuildRoutes(context.Background(), []cluster.Peer{peerA, peerB}, preview.Shards, preview.Layers)
	if err != nil {
		t.Fatalf("BuildRoutes: %v", err)
	}
	if len(routes) != len(preview.Shards) {
		t.Fatalf("routes=%d shards=%d", len(routes), len(preview.Shards))
	}
	for i, route := range routes {
		if route.Peer.ID != preview.Shards[i].DeviceID {
			t.Fatalf("route %d peer=%s shard device=%s", i, route.Peer.ID, preview.Shards[i].DeviceID)
		}
	}
}

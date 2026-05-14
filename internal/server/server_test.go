package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

func TestServerServesWebUI(t *testing.T) {
	s := New(nil, WithWebDir("../../web"))
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK || !bytes.Contains(rr.Body.Bytes(), []byte("go-exotic planner")) {
		t.Fatalf("web status=%d body=%s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/static/app.js", nil))
	if rr.Code != http.StatusOK || rr.Body.Len() == 0 {
		t.Fatalf("static status=%d len=%d", rr.Code, rr.Body.Len())
	}
}

func TestServerHealthCapabilitiesAndPlacement(t *testing.T) {
	s := New([]protocol.Capability{{PeerID: "local", Device: exotic.Device{ID: "local", MemoryGB: 1, Backend: "go-pherence"}}})
	h := s.Handler()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", rr.Code, rr.Body.String())
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("capabilities status=%d body=%s", rr.Code, rr.Body.String())
	}
	var caps protocol.CapabilitiesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &caps); err != nil || len(caps.Capabilities) != 1 {
		t.Fatalf("capabilities decode=%v caps=%+v", err, caps)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/placement/preview?layers=4&model=test", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("placement status=%d body=%s", rr.Code, rr.Body.String())
	}
	var preview protocol.PlacementPreview
	if err := json.Unmarshal(rr.Body.Bytes(), &preview); err != nil {
		t.Fatalf("preview decode: %v", err)
	}
	if preview.Layers != 4 || len(preview.Shards) != 1 || preview.Shards[0].EndLayer != 3 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
}

func TestServerCopiesCapabilityMetadata(t *testing.T) {
	caps := []protocol.Capability{{PeerID: "local", Device: exotic.Device{ID: "local", MemoryGB: 1}, Metadata: map[string]string{"mode": "local"}}}
	s := New(caps)
	caps[0].Metadata["mode"] = "mutated"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/capabilities", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.CapabilitiesResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Capabilities[0].Metadata["mode"] != "local" {
		t.Fatalf("server metadata aliases input: %+v", got.Capabilities[0].Metadata)
	}
}

func TestServerRoutePreview(t *testing.T) {
	s := New([]protocol.Capability{
		{PeerID: "a", Device: exotic.Device{ID: "a", MemoryGB: 1}, Metadata: map[string]string{"address": "http://127.0.0.1:1", "transport": "http"}},
		{PeerID: "b", Device: exotic.Device{ID: "b", MemoryGB: 1}, Metadata: map[string]string{"address": "http://127.0.0.1:2", "transport": "http"}},
	})
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/routes/preview?layers=4&model=test", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.RoutePreview
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ModelID != "test" || got.Layers != 4 || len(got.Routes) != 2 || got.Routes[0].PeerID != "a" || got.Routes[1].PeerID != "b" {
		t.Fatalf("unexpected route preview: %+v", got)
	}
}

func TestServerRoutePreviewFromRegistry(t *testing.T) {
	registry := cluster.NewRegistry()
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	peerB, _ := cluster.LocalPeer("b", "http://127.0.0.1:2", 1)
	if err := registry.Upsert(peerB); err != nil {
		t.Fatal(err)
	}
	if err := registry.Upsert(peerA); err != nil {
		t.Fatal(err)
	}
	s := New(nil, WithRegistry(registry))
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/routes/preview?layers=4&model=registry", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.RoutePreview
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ModelID != "registry" || len(got.Routes) != 2 || got.Routes[0].PeerID != "a" || got.Routes[1].PeerID != "b" {
		t.Fatalf("unexpected registry route preview: %+v", got)
	}
}

func TestServerRoutePreviewFromRegistryRejectsEmptyRegistry(t *testing.T) {
	s := New([]protocol.Capability{{PeerID: "fallback", Device: exotic.Device{ID: "fallback", MemoryGB: 1}, Metadata: map[string]string{"address": "http://127.0.0.1:9", "transport": "http"}}}, WithRegistry(cluster.NewRegistry()))
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/routes/preview?layers=1", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
}

func TestServerRejectsMalformedRoutePreview(t *testing.T) {
	s := New([]protocol.Capability{{PeerID: "local", Device: exotic.Device{ID: "local", MemoryGB: 1}}})
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/routes/preview?layers=bad", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/routes/preview?layers=1", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 for missing capability address", rr.Code)
	}
}

func TestServerRejectsMalformedPlacementRequest(t *testing.T) {
	s := New([]protocol.Capability{{PeerID: "local", Device: exotic.Device{ID: "local", MemoryGB: 1}}})
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/placement/preview?layers=bad", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
}

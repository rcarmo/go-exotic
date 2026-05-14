package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("index cache-control=%q", got)
	}
	rr = httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/static/app.js", nil))
	if rr.Code != http.StatusOK || rr.Body.Len() == 0 {
		t.Fatalf("static status=%d len=%d", rr.Code, rr.Body.Len())
	}
	if got := rr.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("static cache-control=%q", got)
	}
}

func TestServerStaticServesConcreteFilesOnly(t *testing.T) {
	webDir := t.TempDir()
	staticDir := filepath.Join(webDir, "static")
	if err := os.Mkdir(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(staticDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(webDir, "secret.txt"), filepath.Join(staticDir, "secret-link.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	s := New(nil, WithWebDir(webDir))
	for _, path := range []string{"/static/", "/static", "/static/nested", "/static/nested/", "/static/%2e%2e/index.html", "/static/secret-link.txt"} {
		rr := httptest.NewRecorder()
		s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d body=%s", path, rr.Code, rr.Body.String())
		}
	}
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/static/app.js", nil))
	if rr.Code != http.StatusOK || rr.Body.String() != "console.log('ok')" {
		t.Fatalf("asset status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestServerMethodNotAllowedIncludesAllowHeader(t *testing.T) {
	s := New(nil, WithWebDir("../../web"))
	cases := []struct {
		path  string
		allow string
	}{
		{path: "/", allow: "GET, HEAD"},
		{path: "/static/app.js", allow: "GET, HEAD"},
		{path: "/ui/status", allow: "GET"},
		{path: "/models/local", allow: "GET"},
		{path: "/models/helpers", allow: "GET"},
		{path: "/shards/execute", allow: "POST"},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPut, tc.path, nil))
		if rr.Code != http.StatusMethodNotAllowed || rr.Header().Get("Allow") != tc.allow {
			t.Fatalf("%s status=%d allow=%q", tc.path, rr.Code, rr.Header().Get("Allow"))
		}
	}
}

func TestServerLocalModels(t *testing.T) {
	root := t.TempDir()
	complete := filepath.Join(root, "z-complete")
	if err := os.Mkdir(complete, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"config.json", "tokenizer.json", "model.safetensors"} {
		if err := os.WriteFile(filepath.Join(complete, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	incomplete := filepath.Join(root, "a-incomplete")
	if err := os.Mkdir(incomplete, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(incomplete, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/models/local?root="+root, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.LocalModelsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Root != root || got.Limit != maxLocalModelInventory || got.Truncated || len(got.Models) != 2 || got.Models[0].ID != "a-incomplete" || got.Models[1].ID != "z-complete" || got.Models[0].Complete || !got.Models[1].Complete {
		t.Fatalf("unexpected local models: %+v", got)
	}
}

func TestServerModelFileStatusIgnoresNonRegularMatches(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "config.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "weights.safetensors"), 0o755); err != nil {
		t.Fatal(err)
	}
	statuses := modelFileStatuses(root, []string{"config.json", "*.safetensors"})
	if len(statuses) != 2 || statuses[0].Present || statuses[1].Present {
		t.Fatalf("non-regular matches should not count as model files: %+v", statuses)
	}
}

func TestServerModelFileStatusEscapesModelPathGlobChars(t *testing.T) {
	root := filepath.Join(t.TempDir(), "model[1]")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "model.safetensors"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	statuses := modelFileStatuses(root, []string{"*.safetensors"})
	if len(statuses) != 1 || !statuses[0].Present || statuses[0].Truncated || statuses[0].Limit != 0 || len(statuses[0].Matches) != 1 {
		t.Fatalf("unexpected file status for glob-like path: %+v", statuses)
	}
}

func TestServerModelFileStatusTruncatesMatches(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < maxModelFileStatusMatches+1; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("weights-%03d.safetensors", i)), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	statuses := modelFileStatuses(root, []string{"*.safetensors"})
	if len(statuses) != 1 || !statuses[0].Present || !statuses[0].Truncated || statuses[0].Limit != maxModelFileStatusMatches || len(statuses[0].Matches) != maxModelFileStatusMatches {
		t.Fatalf("unexpected file status: %+v", statuses)
	}
	if statuses[0].Matches[len(statuses[0].Matches)-1] != fmt.Sprintf("weights-%03d.safetensors", maxModelFileStatusMatches-1) {
		t.Fatalf("matches were not sorted before truncation: %+v", statuses[0].Matches)
	}
}

func TestServerLocalModelsRejectsBadLimit(t *testing.T) {
	root := t.TempDir()
	s := New(nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/models/local?root="+root+"&limit=bad", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400", rr.Code)
	}
}

func TestServerLocalModelsHonorsLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("model-%03d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	s := New(nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/models/local?root="+root+"&limit=2", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.LocalModelsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Truncated || got.Limit != 2 || len(got.Models) != 2 {
		t.Fatalf("unexpected limited response: %+v", got)
	}
}

func TestServerLocalModelsTruncatesLargeInventory(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "zzz-over-limit"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxLocalModelInventory; i++ {
		if err := os.Mkdir(filepath.Join(root, fmt.Sprintf("model-%03d", i)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	s := New(nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/models/local?root="+root, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.LocalModelsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.Truncated || got.Limit != maxLocalModelInventory || len(got.Models) != maxLocalModelInventory {
		t.Fatalf("unexpected truncation response: %+v", got)
	}
	if got.Models[len(got.Models)-1].ID != fmt.Sprintf("model-%03d", maxLocalModelInventory-1) {
		t.Fatalf("inventory was not sorted before truncation: last=%s", got.Models[len(got.Models)-1].ID)
	}
}

func TestServerModelHelpersQuotesCommands(t *testing.T) {
	s := New(nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/models/helpers?model=demo%20model&path=/tmp/model%20with%20space", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.ModelHelperResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	commands := make([]string, 0, len(got.Commands))
	for _, command := range got.Commands {
		if command.Label == "" || command.Command == "" {
			t.Fatalf("unlabeled command: %+v", command)
		}
		commands = append(commands, command.Command)
	}
	joined := strings.Join(commands, "\n")
	if !strings.Contains(joined, "'/tmp/model with space'") || !strings.Contains(joined, "'demo model'") {
		t.Fatalf("commands are not shell-quoted: %v", got.Commands)
	}
}

func TestServerModelHelpers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model.safetensors"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/models/helpers?model=demo&path="+dir, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.ModelHelperResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Status != "manual" || got.ModelPath != dir || len(got.Presets) == 0 || len(got.RequiredFiles) != 3 || len(got.Commands) == 0 || !bytes.Contains(rr.Body.Bytes(), []byte(dir)) {
		t.Fatalf("unexpected model helpers: %+v", got)
	}
	if len(got.Files) != 3 || !got.Files[0].Present || got.Files[1].Present || !got.Files[2].Present {
		t.Fatalf("unexpected model helpers: %+v", got)
	}
}

func TestServerUIStatusIgnoresSymlinkedBundle(t *testing.T) {
	webDir := t.TempDir()
	staticDir := filepath.Join(webDir, "static")
	if err := os.Mkdir(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "target.js"), []byte("console.log('target')"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(webDir, "target.js"), filepath.Join(staticDir, "app.js")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	s := New(nil, WithWebDir(webDir))
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ui/status", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.UIStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.WebBundle != "" {
		t.Fatalf("symlinked bundle should not be hashed: %+v", got)
	}
}

func TestServerUIStatus(t *testing.T) {
	s := New(nil, WithWebDir("../../web"))
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/ui/status", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.UIStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "go-exotic" || got.APIVersion == "" || !got.WebUI || got.StartedAt == "" || got.UptimeSeconds < 0 || got.WebBundle == "" || len(got.Endpoints) == 0 || got.Boundary == "" {
		t.Fatalf("unexpected ui status: %+v", got)
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

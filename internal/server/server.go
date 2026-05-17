package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
)

const (
	maxLocalModelInventory    = 200
	maxModelFileStatusMatches = 20
)

type ShardWorker interface {
	ExecuteShard(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error)
}

type Server struct {
	capabilities []protocol.Capability
	registry     *cluster.Registry
	shardWorker  ShardWorker
	totalLayers  int
	webDir       string
	startedAt    time.Time
}

type Option func(*Server)

func WithWebDir(dir string) Option {
	return func(s *Server) {
		s.webDir = dir
	}
}

func WithRegistry(registry *cluster.Registry) Option {
	return func(s *Server) {
		s.registry = registry
	}
}

func WithShardExecution(worker ShardWorker, totalLayers int) Option {
	return func(s *Server) {
		s.shardWorker = worker
		s.totalLayers = totalLayers
	}
}

func New(capabilities []protocol.Capability, opts ...Option) *Server {
	caps := make([]protocol.Capability, 0, len(capabilities))
	for _, cap := range capabilities {
		cap.Metadata = cloneMetadata(cap.Metadata)
		caps = append(caps, cap)
	}
	s := &Server{capabilities: caps, startedAt: time.Now().UTC()}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	return s
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWeb)
	mux.HandleFunc("/static", s.handleStatic)
	mux.HandleFunc("/static/", s.handleStatic)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/ui/status", s.handleUIStatus)
	mux.HandleFunc("/capabilities", s.handleCapabilities)
	mux.HandleFunc("/models/local", s.handleLocalModels)
	mux.HandleFunc("/models/helpers", s.handleModelHelpers)
	mux.HandleFunc("/placement/preview", s.handlePlacementPreview)
	mux.HandleFunc("/routes/preview", s.handleRoutesPreview)
	mux.HandleFunc("/shards/execute", s.handleShardExecute)
	return mux
}

func (s *Server) handleWeb(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeMethodNotAllowed(w, http.MethodGet, http.MethodHead)
		return
	}
	file, ok := s.webIndexPath()
	if !ok {
		http.NotFound(w, r)
		return
	}
	setWebCacheHeaders(w)
	http.ServeFile(w, r, file)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeMethodNotAllowed(w, http.MethodGet, http.MethodHead)
		return
	}
	asset := strings.TrimPrefix(r.URL.Path, "/static/")
	file, ok := s.staticAssetPath(asset)
	if !ok {
		http.NotFound(w, r)
		return
	}
	setWebCacheHeaders(w)
	http.ServeFile(w, r, file)
}

func setWebCacheHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-cache")
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, protocol.HealthResponse{Status: "ok"})
}

func (s *Server) handleUIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	boundary := "shard execution disabled"
	if s.shardWorker != nil && s.totalLayers > 0 {
		boundary = "local shard execution explicitly wired"
	}
	now := time.Now().UTC()
	writeJSON(w, http.StatusOK, protocol.UIStatusResponse{
		Name:          "go-exotic",
		APIVersion:    "v1alpha",
		WebUI:         true,
		StartedAt:     s.startedAt.Format(time.RFC3339),
		UptimeSeconds: int64(now.Sub(s.startedAt).Seconds()),
		WebBundle:     s.webBundleID(),
		Endpoints: []string{
			"GET /health",
			"GET /capabilities",
			"GET /models/local",
			"GET /models/helpers",
			"GET /placement/preview",
			"GET /routes/preview",
			"POST /shards/execute (gated)",
		},
		Boundary: boundary,
	})
}

func (s *Server) webIndexPath() (string, bool) {
	dir := s.webDir
	if dir == "" {
		dir = "web"
	}
	return regularFilePath(filepath.Join(dir, "index.html"))
}

func (s *Server) webBundleID() string {
	file, ok := s.staticAssetPath("app.js")
	if !ok {
		return ""
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}

func (s *Server) staticAssetPath(asset string) (string, bool) {
	if asset == "" || strings.HasSuffix(asset, "/") || path.Clean("/"+asset) != "/"+asset {
		return "", false
	}
	dir := s.webDir
	if dir == "" {
		dir = "web"
	}
	return regularFilePath(filepath.Join(dir, "static", filepath.FromSlash(asset)))
}

func regularFilePath(file string) (string, bool) {
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", false
	}
	return file, true
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, protocol.CapabilitiesResponse{Capabilities: s.capabilitiesCopy()})
}

func (s *Server) handleLocalModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	root := r.URL.Query().Get("root")
	if root == "" {
		root = "../go-pherence/models"
	}
	limit := maxLocalModelInventory
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		if parsed < limit {
			limit = parsed
		}
	}
	required := []string{"config.json", "tokenizer.json", "*.safetensors"}
	entries, err := os.ReadDir(root)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	dirs := make([]os.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry)
		}
	}
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].Name() < dirs[j].Name() })
	truncated := len(dirs) > limit
	if truncated {
		dirs = dirs[:limit]
	}
	models := make([]protocol.LocalModel, 0, len(dirs))
	for _, entry := range dirs {
		path := filepath.Join(root, entry.Name())
		files := modelFileStatuses(path, required)
		complete := true
		for _, file := range files {
			complete = complete && file.Present
		}
		models = append(models, protocol.LocalModel{ID: entry.Name(), Path: path, Files: files, Complete: complete})
	}
	writeJSON(w, http.StatusOK, protocol.LocalModelsResponse{Root: root, Models: models, Truncated: truncated, Limit: limit})
}

func (s *Server) handleModelHelpers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	modelID := r.URL.Query().Get("model")
	if modelID == "" {
		modelID = "MODEL"
	}
	modelPath := r.URL.Query().Get("path")
	if modelPath == "" {
		modelPath = "../go-pherence/models/" + modelID
	}
	required := []string{"config.json", "tokenizer.json", "*.safetensors"}
	quotedPath := shellQuote(modelPath)
	quotedModel := shellQuote(modelID)
	writeJSON(w, http.StatusOK, protocol.ModelHelperResponse{
		Status:        "manual",
		ModelPath:     modelPath,
		Presets:       modelPresets(),
		RequiredFiles: required,
		Files:         modelFileStatuses(modelPath, required),
		Commands: []protocol.ModelCommand{
			{Label: "Create fixture directory", Command: "mkdir -p " + quotedPath},
			{Label: "List required files", Command: "find " + quotedPath + " -maxdepth 1 \\( -name 'config.json' -o -name 'tokenizer.json' -o -name '*.safetensors' \\) -print"},
			{Label: "Local generation smoke", Command: "go run ./cmd/go-exotic run -model " + quotedPath + " -prompt \"Hello\" -tokens 1"},
			{Label: "Planning preview", Command: "go run ./cmd/go-exotic routes -layers 4 -model " + quotedModel + " -json"},
			{Label: "Explicit shard-server opt-in", Command: "go run ./cmd/go-exotic serve -addr 127.0.0.1:8089 -shard-model " + quotedPath},
		},
	})
}

func (s *Server) handlePlacementPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	layers, err := positiveIntQuery(r, "layers")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	devices := make([]exotic.Device, 0, len(s.capabilities))
	for _, cap := range s.capabilities {
		devices = append(devices, cap.Device)
	}
	plan, err := placement.NewPlan(devices, layers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, protocol.PlacementPreview{ModelID: r.URL.Query().Get("model"), Layers: layers, Shards: plan.Shards})
}

func (s *Server) handleRoutesPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	layers, err := positiveIntQuery(r, "layers")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.registry != nil {
		preview, err := router.PreviewFromRegistry(r.Context(), s.registry, r.URL.Query().Get("model"), layers)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, preview)
		return
	}
	preview, err := router.PreviewFromCapabilities(r.Context(), s.capabilitiesCopy(), r.URL.Query().Get("model"), layers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) handleShardExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	if s.shardWorker == nil || s.totalLayers <= 0 {
		writeError(w, http.StatusServiceUnavailable, "shard execution disabled")
		return
	}
	defer r.Body.Close()
	var wire protocol.ShardExecutionHTTPBridgeRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&wire); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain a single JSON object")
		return
	}
	req, err := wire.ShardExecutionRequest()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if headerID := r.Header.Get("X-Go-Exotic-Request-ID"); headerID != "" && headerID != req.RequestID {
		writeError(w, http.StatusBadRequest, "request id header mismatch")
		return
	}
	if err := req.Validate(s.totalLayers); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := s.shardWorker.ExecuteShard(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := resp.Validate(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out, err := protocol.NewShardExecutionHTTPBridgeResponse(resp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func modelPresets() []protocol.ModelPreset {
	return []protocol.ModelPreset{
		{ID: "smollm2-135m", Name: "SmolLM2 135M", Path: "../go-pherence/models/smollm2-135m", Description: "Small local fixture used by go-exotic parity tests."},
		{ID: "demo", Name: "Demo placeholder", Path: "../go-pherence/models/demo", Description: "Template path for staging a local go-pherence model."},
	}
}

func modelFileStatuses(modelPath string, required []string) []protocol.ModelFileStatus {
	entries, _ := os.ReadDir(modelPath)
	out := make([]protocol.ModelFileStatus, 0, len(required))
	for _, pattern := range required {
		matches := make([]string, 0)
		for _, entry := range entries {
			matched, err := filepath.Match(pattern, entry.Name())
			if err == nil && matched && entryRegularFile(entry) {
				matches = append(matches, entry.Name())
			}
		}
		sort.Strings(matches)
		truncated := len(matches) > maxModelFileStatusMatches
		limit := 0
		if truncated {
			matches = matches[:maxModelFileStatusMatches]
			limit = maxModelFileStatusMatches
		}
		out = append(out, protocol.ModelFileStatus{Pattern: pattern, Present: len(matches) > 0, Matches: matches, Truncated: truncated, Limit: limit})
	}
	return out
}

func entryRegularFile(entry os.DirEntry) bool {
	if entry.Type()&os.ModeType != 0 {
		return false
	}
	info, err := entry.Info()
	return err == nil && info.Mode().IsRegular()
}

func positiveIntQuery(r *http.Request, name string) (int, error) {
	value := r.URL.Query().Get(name)
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return parsed, nil
}

func (s *Server) capabilitiesCopy() []protocol.Capability {
	caps := make([]protocol.Capability, 0, len(s.capabilities))
	for _, cap := range s.capabilities {
		cap.Metadata = cloneMetadata(cap.Metadata)
		caps = append(caps, cap)
	}
	return caps
}

func cloneMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Response has already started; nothing useful to do for this skeleton.
		return
	}
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf("%s", msg)})
}

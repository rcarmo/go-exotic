package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
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
	s := &Server{capabilities: caps}
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
	mux.HandleFunc("/static/", s.handleStatic)
	mux.HandleFunc("/health", s.handleHealth)
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	dir := s.webDir
	if dir == "" {
		dir = "web"
	}
	http.ServeFile(w, r, filepath.Join(dir, "index.html"))
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	dir := s.webDir
	if dir == "" {
		dir = "web"
	}
	fs := http.FileServer(http.FS(os.DirFS(filepath.Join(dir, "static"))))
	http.StripPrefix("/static/", fs).ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, protocol.HealthResponse{Status: "ok"})
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, protocol.CapabilitiesResponse{Capabilities: s.capabilitiesCopy()})
}

func (s *Server) handleLocalModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	root := r.URL.Query().Get("root")
	if root == "" {
		root = "../go-pherence/models"
	}
	required := []string{"config.json", "tokenizer.json", "*.safetensors"}
	entries, err := os.ReadDir(root)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	models := make([]protocol.LocalModel, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(root, entry.Name())
		files := modelFileStatuses(path, required)
		complete := true
		for _, file := range files {
			complete = complete && file.Present
		}
		models = append(models, protocol.LocalModel{ID: entry.Name(), Path: path, Files: files, Complete: complete})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })
	writeJSON(w, http.StatusOK, protocol.LocalModelsResponse{Root: root, Models: models})
}

func (s *Server) handleModelHelpers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
	writeJSON(w, http.StatusOK, protocol.ModelHelperResponse{
		Status:        "manual",
		ModelPath:     modelPath,
		Presets:       modelPresets(),
		RequiredFiles: required,
		Files:         modelFileStatuses(modelPath, required),
		Commands: []string{
			"mkdir -p " + modelPath,
			"find " + modelPath + " -maxdepth 1 \\( -name 'config.json' -o -name 'tokenizer.json' -o -name '*.safetensors' \\) -print",
			"go run ./cmd/go-exotic run -model " + modelPath + " -prompt \"Hello\" -tokens 1",
			"go run ./cmd/go-exotic routes -layers 4 -model " + modelID + " -json",
			"go run ./cmd/go-exotic serve -addr 127.0.0.1:8089 -shard-model " + modelPath,
		},
	})
}

func (s *Server) handlePlacementPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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

func modelPresets() []protocol.ModelPreset {
	return []protocol.ModelPreset{
		{ID: "smollm2-135m", Name: "SmolLM2 135M", Path: "../go-pherence/models/smollm2-135m", Description: "Small local fixture used by go-exotic parity tests."},
		{ID: "demo", Name: "Demo placeholder", Path: "../go-pherence/models/demo", Description: "Template path for staging a local go-pherence model."},
	}
}

func modelFileStatuses(modelPath string, required []string) []protocol.ModelFileStatus {
	out := make([]protocol.ModelFileStatus, 0, len(required))
	for _, pattern := range required {
		matches, _ := filepath.Glob(filepath.Join(modelPath, pattern))
		for i := range matches {
			matches[i] = filepath.Base(matches[i])
		}
		sort.Strings(matches)
		out = append(out, protocol.ModelFileStatus{Pattern: pattern, Present: len(matches) > 0, Matches: matches})
	}
	return out
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

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": fmt.Sprintf("%s", msg)})
}

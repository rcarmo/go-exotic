package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

type ShardWorker interface {
	ExecuteShard(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error)
}

type Server struct {
	capabilities []protocol.Capability
	shardWorker  ShardWorker
	totalLayers  int
}

type Option func(*Server)

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
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/capabilities", s.handleCapabilities)
	mux.HandleFunc("/placement/preview", s.handlePlacementPreview)
	mux.HandleFunc("/shards/execute", s.handleShardExecute)
	return mux
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

func (s *Server) handlePlacementPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	layersText := r.URL.Query().Get("layers")
	layers, err := strconv.Atoi(layersText)
	if err != nil || layers <= 0 {
		writeError(w, http.StatusBadRequest, "layers must be a positive integer")
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
	req, err := wire.ShardExecutionRequest()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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

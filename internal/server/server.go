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
	"github.com/rcarmo/go-exotic/internal/router"
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
	mux.HandleFunc("/routes/preview", s.handleRoutesPreview)
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
	devices := make([]exotic.Device, 0, len(s.capabilities))
	for _, cap := range s.capabilities {
		devices = append(devices, cap.Device)
	}
	plan, err := placement.NewPlan(devices, layers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	routes, err := router.BuildRoutesFromCapabilities(r.Context(), s.capabilitiesCopy(), plan.Shards, layers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entries := make([]protocol.RouteEntry, 0, len(routes))
	for _, route := range routes {
		entries = append(entries, protocol.RouteEntry{PeerID: route.Peer.ID, Address: route.Peer.Address, Transport: route.Peer.Transport, Shard: route.Shard})
	}
	writeJSON(w, http.StatusOK, protocol.RoutePreview{ModelID: r.URL.Query().Get("model"), Layers: layers, Routes: entries})
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

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

type Server struct {
	capabilities []protocol.Capability
}

func New(capabilities []protocol.Capability) *Server {
	caps := append([]protocol.Capability(nil), capabilities...)
	return &Server{capabilities: caps}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/capabilities", s.handleCapabilities)
	mux.HandleFunc("/placement/preview", s.handlePlacementPreview)
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
	writeJSON(w, http.StatusOK, protocol.CapabilitiesResponse{Capabilities: append([]protocol.Capability(nil), s.capabilities...)})
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

package protocol

import "github.com/rcarmo/go-exotic/internal/exotic"

// Capability is advertised by peers during membership exchange.
type Capability struct {
	PeerID   string            `json:"peer_id"`
	Device   exotic.Device     `json:"device"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// PlacementPreview is the wire-facing representation of a shard plan.
type PlacementPreview struct {
	ModelID string         `json:"model_id"`
	Layers  int            `json:"layers"`
	Shards  []exotic.Shard `json:"shards"`
}

// RoutePreview is the wire-facing representation of planned shard routes.
type RoutePreview struct {
	ModelID string       `json:"model_id"`
	Layers  int          `json:"layers"`
	Routes  []RouteEntry `json:"routes"`
}

type RouteEntry struct {
	PeerID    string       `json:"peer_id"`
	Address   string       `json:"address"`
	Transport string       `json:"transport"`
	Shard     exotic.Shard `json:"shard"`
}

type ModelPreset struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Description string `json:"description"`
}

type ModelFileStatus struct {
	Pattern string   `json:"pattern"`
	Present bool     `json:"present"`
	Matches []string `json:"matches,omitempty"`
}

type LocalModel struct {
	ID       string            `json:"id"`
	Path     string            `json:"path"`
	Files    []ModelFileStatus `json:"files"`
	Complete bool              `json:"complete"`
}

type LocalModelsResponse struct {
	Root   string       `json:"root"`
	Models []LocalModel `json:"models"`
}

type ModelHelperResponse struct {
	Status        string            `json:"status"`
	ModelPath     string            `json:"model_path"`
	Presets       []ModelPreset     `json:"presets"`
	RequiredFiles []string          `json:"required_files"`
	Files         []ModelFileStatus `json:"files"`
	Commands      []string          `json:"commands"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type UIStatusResponse struct {
	Name       string   `json:"name"`
	APIVersion string   `json:"api_version"`
	WebUI      bool     `json:"web_ui"`
	Endpoints  []string `json:"endpoints"`
	Boundary   string   `json:"boundary"`
}

type CapabilitiesResponse struct {
	Capabilities []Capability `json:"capabilities"`
}

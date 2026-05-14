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

type ModelHelperResponse struct {
	Status        string   `json:"status"`
	RequiredFiles []string `json:"required_files"`
	Commands      []string `json:"commands"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type CapabilitiesResponse struct {
	Capabilities []Capability `json:"capabilities"`
}

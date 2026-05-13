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

type HealthResponse struct {
	Status string `json:"status"`
}

type CapabilitiesResponse struct {
	Capabilities []Capability `json:"capabilities"`
}

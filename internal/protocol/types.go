package protocol

import "github.com/rcarmo/go-exotic/internal/exotic"

// Capability is advertised by peers during membership exchange.
type Capability struct {
	PeerID   string
	Device   exotic.Device
	Metadata map[string]string
}

// PlacementPreview is the wire-facing representation of a shard plan.
type PlacementPreview struct {
	ModelID string
	Layers  int
	Shards  []exotic.Shard
}

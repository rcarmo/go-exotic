package router

import (
	"context"
	"fmt"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

// PreviewFromRegistry builds a planning-only route preview from the current
// registry peer snapshot. It does not execute shards or contact peers.
func PreviewFromRegistry(ctx context.Context, registry *cluster.Registry, modelID string, totalLayers int) (protocol.RoutePreview, error) {
	if err := ctx.Err(); err != nil {
		return protocol.RoutePreview{}, err
	}
	if registry == nil {
		return protocol.RoutePreview{}, fmt.Errorf("nil registry")
	}
	peers := registry.Peers()
	if len(peers) == 0 {
		return protocol.RoutePreview{}, fmt.Errorf("no peers")
	}
	caps := make([]protocol.Capability, 0, len(peers))
	for _, peer := range peers {
		if err := ctx.Err(); err != nil {
			return protocol.RoutePreview{}, err
		}
		cap, err := peer.Capability()
		if err != nil {
			return protocol.RoutePreview{}, fmt.Errorf("peer %q capability: %w", peer.ID, err)
		}
		caps = append(caps, cap)
	}
	return PreviewFromCapabilities(ctx, caps, modelID, totalLayers)
}

package router

import (
	"context"
	"fmt"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
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
	devices := make([]exotic.Device, 0, len(peers))
	caps := make([]protocol.Capability, 0, len(peers))
	for _, peer := range peers {
		if err := ctx.Err(); err != nil {
			return protocol.RoutePreview{}, err
		}
		cap, err := peer.Capability()
		if err != nil {
			return protocol.RoutePreview{}, fmt.Errorf("peer %q capability: %w", peer.ID, err)
		}
		devices = append(devices, cap.Device)
		caps = append(caps, cap)
	}
	plan, err := placement.NewPlan(devices, totalLayers)
	if err != nil {
		return protocol.RoutePreview{}, err
	}
	routes, err := BuildRoutesFromCapabilities(ctx, caps, plan.Shards, totalLayers)
	if err != nil {
		return protocol.RoutePreview{}, err
	}
	entries := make([]protocol.RouteEntry, 0, len(routes))
	for _, route := range routes {
		entries = append(entries, protocol.RouteEntry{PeerID: route.Peer.ID, Address: route.Peer.Address, Transport: route.Peer.Transport, Shard: route.Shard})
	}
	return protocol.RoutePreview{ModelID: modelID, Layers: totalLayers, Routes: entries}, nil
}

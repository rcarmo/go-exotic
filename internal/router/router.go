package router

import (
	"context"
	"fmt"
	"sort"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

// Route maps one shard to the peer that should execute it.
type Route struct {
	Peer  cluster.Peer
	Shard exotic.Shard
}

// BuildRoutesFromCapabilities converts wire-facing capability advertisements
// into cluster peers, then validates a route plan. It is planning-only; callers
// must still explicitly choose whether to execute requests remotely.
func BuildRoutesFromCapabilities(ctx context.Context, caps []protocol.Capability, shards []exotic.Shard, totalLayers int) ([]Route, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	peers := make([]cluster.Peer, 0, len(caps))
	for _, cap := range caps {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		peer, err := peerFromCapability(cap)
		if err != nil {
			return nil, fmt.Errorf("capability %q: %w", cap.PeerID, err)
		}
		peers = append(peers, peer)
	}
	return BuildRoutes(ctx, peers, shards, totalLayers)
}

func PreviewFromRoutes(modelID string, totalLayers int, routes []Route) protocol.RoutePreview {
	entries := make([]protocol.RouteEntry, 0, len(routes))
	for _, route := range routes {
		entries = append(entries, protocol.RouteEntry{PeerID: route.Peer.ID, Address: route.Peer.Address, Transport: route.Peer.Transport, Shard: route.Shard})
	}
	return protocol.RoutePreview{ModelID: modelID, Layers: totalLayers, Routes: entries}
}

func peerFromCapability(cap protocol.Capability) (cluster.Peer, error) {
	address := cap.Metadata["address"]
	transport := cap.Metadata["transport"]
	if transport == "" {
		transport = cluster.TransportHTTP
	}
	peer := cluster.Peer{ID: cap.PeerID, Address: address, Transport: transport, Device: cap.Device}
	if peer.Device.ID == "" {
		peer.Device.ID = cap.PeerID
	}
	if err := peer.Validate(); err != nil {
		return cluster.Peer{}, err
	}
	return peer, nil
}

// BuildRoutes validates a placement plan against the current peer snapshot and
// returns deterministic shard routes. Actual remote execution is deliberately
// deferred; this is the request-routing contract and cancellation boundary.
func BuildRoutes(ctx context.Context, peers []cluster.Peer, shards []exotic.Shard, totalLayers int) ([]Route, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := placement.ValidatePlan(shards, totalLayers); err != nil {
		return nil, err
	}
	if len(peers) == 0 {
		return nil, fmt.Errorf("no peers")
	}
	peerByID := make(map[string]cluster.Peer, len(peers))
	for _, peer := range peers {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := peer.Validate(); err != nil {
			return nil, fmt.Errorf("peer %q: %w", peer.ID, err)
		}
		if _, exists := peerByID[peer.ID]; exists {
			return nil, fmt.Errorf("duplicate peer id %q", peer.ID)
		}
		peerByID[peer.ID] = peer
	}
	if len(peerByID) < len(shards) {
		return nil, fmt.Errorf("peers=%d fewer than shards=%d", len(peerByID), len(shards))
	}
	routes := make([]Route, 0, len(shards))
	for _, shard := range shards {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		peer, ok := peerByID[shard.DeviceID]
		if !ok {
			return nil, fmt.Errorf("no peer for shard device %q", shard.DeviceID)
		}
		routes = append(routes, Route{Peer: peer, Shard: shard})
	}
	sort.Slice(routes, func(i, j int) bool { return routes[i].Shard.StartLayer < routes[j].Shard.StartLayer })
	return routes, nil
}

package router

import (
	"context"
	"fmt"
	"sort"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/placement"
)

// Route maps one shard to the peer that should execute it.
type Route struct {
	Peer  cluster.Peer
	Shard exotic.Shard
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

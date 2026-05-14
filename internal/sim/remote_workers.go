package sim

import (
	"fmt"
	"net/http"
	"time"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/router"
)

// RemoteWorkersFromRoutes builds simulator workers that call each planned peer's
// HTTP shard endpoint. It is an explicit orchestration helper and is not wired
// into default generation paths.
func RemoteWorkersFromRoutes(routes []router.Route, client *http.Client, timeout time.Duration) (map[string]Worker, error) {
	if len(routes) == 0 {
		return nil, fmt.Errorf("no routes")
	}
	workers := make(map[string]Worker, len(routes))
	for _, route := range routes {
		peer := route.Peer
		if err := peer.Validate(); err != nil {
			return nil, fmt.Errorf("route peer %q: %w", peer.ID, err)
		}
		if existing, exists := workers[peer.ID]; exists {
			remote, ok := existing.(cluster.RemoteShardWorker)
			if !ok || remote.Endpoint != peer.Address {
				return nil, fmt.Errorf("duplicate peer id %q with conflicting route", peer.ID)
			}
			continue
		}
		workers[peer.ID] = cluster.RemoteShardWorker{Endpoint: peer.Address, Client: client, Timeout: timeout}
	}
	simulator, err := NewSimulator(workers)
	if err != nil {
		return nil, err
	}
	return simulator.workers, nil
}

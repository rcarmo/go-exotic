package cluster

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Registry tracks known peers and supports heartbeat refresh plus stale-peer
// eviction. It is intentionally in-memory for the first HTTP/LAN transport.
type Registry struct {
	mu    sync.RWMutex
	peers map[string]Peer
	now   func() time.Time
}

func NewRegistry() *Registry {
	return &Registry{peers: make(map[string]Peer), now: func() time.Time { return time.Now().UTC() }}
}

func newRegistryWithClock(now func() time.Time) *Registry {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Registry{peers: make(map[string]Peer), now: now}
}

func (r *Registry) Upsert(peer Peer) error {
	if r == nil {
		return fmt.Errorf("nil registry")
	}
	if err := peer.Validate(); err != nil {
		return err
	}
	peer.LastSeen = r.now()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.peers[peer.ID] = peer
	return nil
}

func (r *Registry) Heartbeat(peerID string) error {
	if r == nil {
		return fmt.Errorf("nil registry")
	}
	if peerID == "" {
		return fmt.Errorf("empty peer id")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	peer, ok := r.peers[peerID]
	if !ok {
		return fmt.Errorf("unknown peer %q", peerID)
	}
	peer.LastSeen = r.now()
	r.peers[peerID] = peer
	return nil
}

func (r *Registry) EvictStale(maxAge time.Duration) []Peer {
	if r == nil || maxAge <= 0 {
		return nil
	}
	cutoff := r.now().Add(-maxAge)
	r.mu.Lock()
	defer r.mu.Unlock()
	evicted := make([]Peer, 0)
	for id, peer := range r.peers {
		if peer.LastSeen.Before(cutoff) {
			evicted = append(evicted, peer)
			delete(r.peers, id)
		}
	}
	sortPeers(evicted)
	return evicted
}

func (r *Registry) Peers() []Peer {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	peers := make([]Peer, 0, len(r.peers))
	for _, peer := range r.peers {
		peers = append(peers, peer)
	}
	sortPeers(peers)
	return peers
}

func sortPeers(peers []Peer) {
	sort.Slice(peers, func(i, j int) bool { return peers[i].ID < peers[j].ID })
}

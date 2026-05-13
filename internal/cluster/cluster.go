package cluster

import "github.com/rcarmo/go-exotic/internal/exotic"

// Peer is the cluster-facing view of a node that can host model shards.
type Peer struct {
	ID      string
	Address string
	Device  exotic.Device
}

package cluster

import (
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

// Transport names the first network surface. We intentionally start with HTTP
// over LAN before adding exo-style discovery transports.
const TransportHTTP = "http"

// Peer is the cluster-facing view of a node that can host model shards.
type Peer struct {
	ID        string
	Address   string
	Transport string
	Device    exotic.Device
	LastSeen  time.Time
}

// Capability returns the protocol DTO advertised by this peer.
func (p Peer) Capability() (protocol.Capability, error) {
	if err := p.Validate(); err != nil {
		return protocol.Capability{}, err
	}
	return protocol.Capability{
		PeerID: p.ID,
		Device: p.Device,
		Metadata: map[string]string{
			"address":   p.Address,
			"transport": p.Transport,
		},
	}, nil
}

func (p Peer) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("empty peer id")
	}
	if strings.TrimSpace(p.Address) == "" {
		return fmt.Errorf("empty peer address")
	}
	if p.Transport == "" {
		return fmt.Errorf("empty peer transport")
	}
	if p.Transport != TransportHTTP {
		return fmt.Errorf("unsupported peer transport %q", p.Transport)
	}
	u, err := url.Parse(p.Address)
	if err != nil || u.Scheme == "" || u.Host == "" || u.Scheme != "http" {
		return fmt.Errorf("invalid peer address %q", p.Address)
	}
	if p.Device.ID == "" {
		return fmt.Errorf("peer device id is empty")
	}
	if p.Device.MemoryGB <= 0 || math.IsNaN(p.Device.MemoryGB) || math.IsInf(p.Device.MemoryGB, 0) {
		return fmt.Errorf("peer device memory %.2f out of range", p.Device.MemoryGB)
	}
	return nil
}

func LocalPeer(id, address string, memoryGB float64) (Peer, error) {
	p := Peer{
		ID:        strings.TrimSpace(id),
		Address:   strings.TrimRight(strings.TrimSpace(address), "/"),
		Transport: TransportHTTP,
		Device:    exotic.Device{ID: strings.TrimSpace(id), MemoryGB: memoryGB, Backend: "go-pherence"},
		LastSeen:  time.Now().UTC(),
	}
	if err := p.Validate(); err != nil {
		return Peer{}, err
	}
	return p, nil
}

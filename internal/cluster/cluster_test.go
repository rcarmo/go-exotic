package cluster

import (
	"testing"

	"github.com/rcarmo/go-exotic/internal/exotic"
)

func TestLocalPeerCapability(t *testing.T) {
	p, err := LocalPeer("local", "http://127.0.0.1:8089", 1)
	if err != nil {
		t.Fatalf("LocalPeer: %v", err)
	}
	cap, err := p.Capability()
	if err != nil {
		t.Fatalf("Capability: %v", err)
	}
	if cap.PeerID != "local" || cap.Device.Backend != "go-pherence" || cap.Metadata["transport"] != TransportHTTP {
		t.Fatalf("unexpected capability: %+v", cap)
	}
}

func TestPeerValidationRejectsMalformedInputs(t *testing.T) {
	valid := Peer{ID: "p", Address: "http://127.0.0.1:1", Transport: TransportHTTP, Device: localDevice("p")}
	cases := []Peer{
		{},
		{ID: "p", Address: "", Transport: TransportHTTP, Device: localDevice("p")},
		{ID: "p", Address: "127.0.0.1:1", Transport: TransportHTTP, Device: localDevice("p")},
		{ID: "p", Address: "http://127.0.0.1:1", Transport: "udp", Device: localDevice("p")},
		{ID: "p", Address: "http://127.0.0.1:1", Transport: TransportHTTP},
		{ID: "p", Address: "http://127.0.0.1:1", Transport: TransportHTTP, Device: localDevice("p", 0)},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid peer rejected: %v", err)
	}
	for i, tc := range cases {
		if err := tc.Validate(); err == nil {
			t.Fatalf("case %d accepted malformed peer: %+v", i, tc)
		}
	}
}

func localDevice(id string, mem ...float64) exotic.Device {
	memory := 1.0
	if len(mem) > 0 {
		memory = mem[0]
	}
	return exotic.Device{ID: id, MemoryGB: memory, Backend: "go-pherence"}
}

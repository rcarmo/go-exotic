package router

import (
	"context"
	"errors"
	"testing"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

func TestBuildRoutes(t *testing.T) {
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	peerB, _ := cluster.LocalPeer("b", "http://127.0.0.1:2", 1)
	shards := []exotic.Shard{{DeviceID: "a", StartLayer: 0, EndLayer: 1}, {DeviceID: "b", StartLayer: 2, EndLayer: 3}}
	routes, err := BuildRoutes(context.Background(), []cluster.Peer{peerB, peerA}, shards, 4)
	if err != nil {
		t.Fatalf("BuildRoutes: %v", err)
	}
	if len(routes) != 2 || routes[0].Peer.ID != "a" || routes[1].Peer.ID != "b" {
		t.Fatalf("unexpected routes: %+v", routes)
	}
}

func TestPreviewFromRoutes(t *testing.T) {
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	preview := PreviewFromRoutes("m", 2, []Route{{Peer: peerA, Shard: exotic.Shard{DeviceID: "a", StartLayer: 0, EndLayer: 1}}})
	if preview.ModelID != "m" || preview.Layers != 2 || len(preview.Routes) != 1 || preview.Routes[0].Address != "http://127.0.0.1:1" {
		t.Fatalf("unexpected preview: %+v", preview)
	}
}

func TestBuildRoutesFromCapabilities(t *testing.T) {
	peerA, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	peerB, _ := cluster.LocalPeer("b", "http://127.0.0.1:2", 2)
	capA, _ := peerA.Capability()
	capB, _ := peerB.Capability()
	routes, err := BuildRoutesFromCapabilities(context.Background(), []protocol.Capability{capB, capA}, []exotic.Shard{{DeviceID: "a", StartLayer: 0, EndLayer: 0}, {DeviceID: "b", StartLayer: 1, EndLayer: 2}}, 3)
	if err != nil {
		t.Fatalf("BuildRoutesFromCapabilities: %v", err)
	}
	if len(routes) != 2 || routes[0].Peer.Address != "http://127.0.0.1:1" || routes[1].Peer.Address != "http://127.0.0.1:2" {
		t.Fatalf("unexpected routes: %+v", routes)
	}
}

func TestBuildRoutesFromCapabilitiesRejectsMalformedInputs(t *testing.T) {
	peer, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	capA, _ := peer.Capability()
	duplicate := capA
	badAddress := capA
	badAddress.Metadata = map[string]string{"address": "not-a-url", "transport": cluster.TransportHTTP}
	badTransport := capA
	badTransport.PeerID = "b"
	badTransport.Device.ID = "b"
	badTransport.Metadata = map[string]string{"address": "http://127.0.0.1:2", "transport": "udp"}
	missing := capA
	missing.PeerID = "missing"
	missing.Device.ID = "missing"
	missing.Metadata = map[string]string{"address": "http://127.0.0.1:3"}
	cases := []struct {
		name string
		caps []protocol.Capability
	}{
		{"duplicate capability", []protocol.Capability{capA, duplicate}},
		{"bad address", []protocol.Capability{badAddress}},
		{"bad transport", []protocol.Capability{badTransport}},
		{"missing shard peer", []protocol.Capability{missing}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BuildRoutesFromCapabilities(context.Background(), tc.caps, []exotic.Shard{{DeviceID: "a", StartLayer: 0, EndLayer: 0}}, 1); err == nil {
				t.Fatal("accepted malformed capabilities")
			}
		})
	}
}

func TestBuildRoutesRejectsMalformedInputs(t *testing.T) {
	peer, _ := cluster.LocalPeer("a", "http://127.0.0.1:1", 1)
	validShard := []exotic.Shard{{DeviceID: "a", StartLayer: 0, EndLayer: 0}}
	cases := []struct {
		name   string
		peers  []cluster.Peer
		shards []exotic.Shard
	}{
		{"bad plan", []cluster.Peer{peer}, []exotic.Shard{{DeviceID: "a", StartLayer: 1, EndLayer: 1}}},
		{"bad peer", []cluster.Peer{{ID: "a"}}, validShard},
		{"duplicate peer", []cluster.Peer{peer, peer}, validShard},
		{"no peers", nil, validShard},
		{"fewer peers than shards", []cluster.Peer{peer}, []exotic.Shard{{DeviceID: "a", StartLayer: 0, EndLayer: 0}, {DeviceID: "b", StartLayer: 1, EndLayer: 1}}},
		{"missing peer", []cluster.Peer{peer}, []exotic.Shard{{DeviceID: "b", StartLayer: 0, EndLayer: 0}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BuildRoutes(context.Background(), tc.peers, tc.shards, 1); err == nil {
				t.Fatal("accepted malformed route inputs")
			}
		})
	}
}

func TestBuildRoutesPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := BuildRoutes(ctx, nil, []exotic.Shard{{DeviceID: "a", StartLayer: 0, EndLayer: 0}}, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
}

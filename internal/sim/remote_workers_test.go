package sim

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
	"github.com/rcarmo/go-exotic/internal/server"
)

func TestRemoteWorkersFromRoutesExecuteThroughSimulator(t *testing.T) {
	worker := WorkerFunc(func(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		out := append([]float32(nil), req.Activation...)
		out[0]++
		return protocol.ShardExecutionResponse{SessionID: req.SessionID, RequestID: req.RequestID, PeerID: req.Shard.DeviceID, Position: req.Position, HiddenSize: req.HiddenSize, Activation: out}, nil
	})
	ts := httptest.NewServer(server.New(nil, server.WithShardExecution(worker, 1)).Handler())
	defer ts.Close()
	peer, err := cluster.LocalPeer("p", ts.URL, 1)
	if err != nil {
		t.Fatalf("LocalPeer: %v", err)
	}
	routes := []router.Route{{Peer: peer, Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}}}
	workers, err := RemoteWorkersFromRoutes(routes, ts.Client(), time.Second)
	if err != nil {
		t.Fatalf("RemoteWorkersFromRoutes: %v", err)
	}
	s, err := NewSimulator(workers)
	if err != nil {
		t.Fatalf("NewSimulator: %v", err)
	}
	resp, err := s.Execute(context.Background(), routes, protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Position: 0, HiddenSize: 1, Activation: []float32{4}}, 1)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.Activation[0] != 5 {
		t.Fatalf("activation=%v want [5]", resp.Activation)
	}
}

func TestRemoteWorkersFromRoutesRejectsMalformedInputs(t *testing.T) {
	if _, err := RemoteWorkersFromRoutes(nil, nil, 0); err == nil {
		t.Fatal("accepted empty routes")
	}
	valid, _ := cluster.LocalPeer("p", "http://127.0.0.1:1", 1)
	badAddr := valid
	badAddr.Address = ""
	if _, err := RemoteWorkersFromRoutes([]router.Route{{Peer: badAddr, Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}}}, nil, 0); err == nil {
		t.Fatal("accepted missing address")
	}
	badTransport := valid
	badTransport.Transport = "udp"
	if _, err := RemoteWorkersFromRoutes([]router.Route{{Peer: badTransport, Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}}}, nil, 0); err == nil {
		t.Fatal("accepted unsupported transport")
	}
	other := valid
	other.Address = "http://127.0.0.1:2"
	if _, err := RemoteWorkersFromRoutes([]router.Route{{Peer: valid}, {Peer: other}}, nil, 0); err == nil {
		t.Fatal("accepted conflicting duplicate peer")
	}
	if workers, err := RemoteWorkersFromRoutes([]router.Route{{Peer: valid}, {Peer: valid}}, nil, 0); err != nil || len(workers) != 1 {
		t.Fatalf("duplicate same peer err=%v workers=%v", err, workers)
	}
}

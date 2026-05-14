package cluster_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/server"
)

func TestRemoteShardWorkerExecutesViaHTTP(t *testing.T) {
	var sawHeader string
	srv := httptest.NewServer(server.New(nil, server.WithShardExecution(serverShardFunc(func(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		out := append([]float32(nil), req.Activation...)
		out[0] += 2
		return protocol.ShardExecutionResponse{SessionID: req.SessionID, RequestID: req.RequestID, PeerID: req.Shard.DeviceID, Position: req.Position, HiddenSize: req.HiddenSize, Activation: out}, nil
	}), 1)).Handler())
	defer srv.Close()
	srv.Config.ErrorLog = nil
	worker := cluster.RemoteShardWorker{Endpoint: srv.URL, Client: srv.Client(), Timeout: time.Second}
	orig := worker.Client.Transport
	worker.Client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		sawHeader = req.Header.Get(cluster.ShardRequestIDHeader)
		return orig.RoundTrip(req)
	})
	resp, err := worker.ExecuteShard(context.Background(), protocol.ShardExecutionRequest{SessionID: "s", RequestID: "rid", ModelID: "m", Shard: exotic.Shard{DeviceID: "peer", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 1, Activation: []float32{3}})
	if err != nil {
		t.Fatalf("ExecuteShard: %v", err)
	}
	if sawHeader != "rid" {
		t.Fatalf("request id header=%q", sawHeader)
	}
	if resp.Activation[0] != 5 || resp.RequestID != "rid" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRemoteShardWorkerPropagatesCancellationAndTimeout(t *testing.T) {
	srv := httptest.NewServer(server.New(nil, server.WithShardExecution(serverShardFunc(func(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		<-ctx.Done()
		return protocol.ShardExecutionResponse{}, ctx.Err()
	}), 1)).Handler())
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	worker := cluster.RemoteShardWorker{Endpoint: srv.URL, Client: srv.Client()}
	_, err := worker.ExecuteShard(ctx, protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "peer", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 1, Activation: []float32{1}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want context.Canceled", err)
	}
	worker.Timeout = time.Nanosecond
	_, err = worker.ExecuteShard(context.Background(), protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r2", ModelID: "m", Shard: exotic.Shard{DeviceID: "peer", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 1, Activation: []float32{1}})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestShardExecuteRejectsRequestIDHeaderMismatch(t *testing.T) {
	srv := httptest.NewServer(server.New(nil, server.WithShardExecution(serverShardFunc(func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		t.Fatal("worker should not be called")
		return protocol.ShardExecutionResponse{}, nil
	}), 1)).Handler())
	defer srv.Close()
	worker := cluster.RemoteShardWorker{Endpoint: srv.URL, Client: srv.Client()}
	req := protocol.ShardExecutionRequest{SessionID: "s", RequestID: "body", ModelID: "m", Shard: exotic.Shard{DeviceID: "peer", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 1, Activation: []float32{1}}
	client := srv.Client()
	orig := client.Transport
	client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.Header.Set(cluster.ShardRequestIDHeader, "header")
		return orig.RoundTrip(r)
	})
	worker.Client = client
	_, err := worker.ExecuteShard(context.Background(), req)
	if err == nil {
		t.Fatal("accepted request id header mismatch")
	}
}

type serverShardFunc func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error)

func (f serverShardFunc) ExecuteShard(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
	return f(ctx, req)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

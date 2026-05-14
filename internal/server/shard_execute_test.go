package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rcarmo/go-exotic/internal/exotic"
	"github.com/rcarmo/go-exotic/internal/protocol"
)

type shardWorkerFunc func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error)

func (f shardWorkerFunc) ExecuteShard(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
	return f(ctx, req)
}

func TestShardExecuteDisabledByDefault(t *testing.T) {
	s := New(nil)
	req := protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 1, Activation: []float32{1}}
	wire, err := protocol.NewShardExecutionHTTPBridgeRequest(req)
	if err != nil {
		t.Fatalf("wire: %v", err)
	}
	body, _ := json.Marshal(wire)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/shards/execute", bytes.NewReader(body)))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestShardExecuteEndpoint(t *testing.T) {
	worker := shardWorkerFunc(func(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		out := append([]float32(nil), req.Activation...)
		out[0] += 1
		return protocol.ShardExecutionResponse{SessionID: req.SessionID, RequestID: req.RequestID, PeerID: req.Shard.DeviceID, Position: req.Position, HiddenSize: req.HiddenSize, Activation: out}, nil
	})
	s := New(nil, WithShardExecution(worker, 1))
	req := protocol.ShardExecutionRequest{SessionID: "s", RequestID: "r", ModelID: "m", Shard: exotic.Shard{DeviceID: "p", StartLayer: 0, EndLayer: 0}, Position: 0, HiddenSize: 1, Activation: []float32{1}}
	wire, err := protocol.NewShardExecutionHTTPBridgeRequest(req)
	if err != nil {
		t.Fatalf("wire: %v", err)
	}
	body, _ := json.Marshal(wire)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/shards/execute", bytes.NewReader(body)))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got protocol.ShardExecutionHTTPBridgeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	resp, err := got.ShardExecutionResponse()
	if err != nil {
		t.Fatalf("response: %v", err)
	}
	if resp.Activation[0] != 2 || resp.PeerID != "p" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestShardExecuteRejectsMalformedPayload(t *testing.T) {
	s := New(nil, WithShardExecution(shardWorkerFunc(func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
		t.Fatal("worker should not be called")
		return protocol.ShardExecutionResponse{}, nil
	}), 1))
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/shards/execute", bytes.NewBufferString(`{"session_id":"s","unknown":true}`)))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

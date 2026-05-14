package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rcarmo/go-exotic/internal/protocol"
)

const ShardRequestIDHeader = "X-Go-Exotic-Request-ID"

// RemoteShardWorker executes shard requests against a peer's HTTP shard endpoint.
type RemoteShardWorker struct {
	Endpoint string
	Client   *http.Client
	Timeout  time.Duration
}

func (w RemoteShardWorker) ExecuteShard(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
	endpoint := strings.TrimRight(strings.TrimSpace(w.Endpoint), "/")
	if endpoint == "" {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("empty remote shard endpoint")
	}
	client := w.Client
	if client == nil {
		client = http.DefaultClient
	}
	if w.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.Timeout)
		defer cancel()
	}
	wire, err := protocol.NewShardExecutionHTTPBridgeRequest(req)
	if err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	body, err := json.Marshal(wire)
	if err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/shards/execute", bytes.NewReader(body))
	if err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set(ShardRequestIDHeader, req.RequestID)
	resp, err := client.Do(httpReq)
	if err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errBody map[string]string
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if msg := errBody["error"]; msg != "" {
			return protocol.ShardExecutionResponse{}, fmt.Errorf("remote shard status=%d: %s", resp.StatusCode, msg)
		}
		return protocol.ShardExecutionResponse{}, fmt.Errorf("remote shard status=%d", resp.StatusCode)
	}
	var out protocol.ShardExecutionHTTPBridgeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	decoded, err := out.ShardExecutionResponse()
	if err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	if decoded.RequestID != req.RequestID {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("remote response request_id=%q, want %q", decoded.RequestID, req.RequestID)
	}
	return decoded, nil
}

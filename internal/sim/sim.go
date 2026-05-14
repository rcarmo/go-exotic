package sim

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/rcarmo/go-exotic/internal/cluster"
	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
)

// Worker executes one shard request in-process. It is a simulation boundary,
// not a network transport.
type Worker interface {
	ExecuteShard(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error)
}

// WorkerFunc adapts a function to Worker.
type WorkerFunc func(context.Context, protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error)

func (f WorkerFunc) ExecuteShard(ctx context.Context, req protocol.ShardExecutionRequest) (protocol.ShardExecutionResponse, error) {
	return f(ctx, req)
}

// Simulator executes routed shard requests sequentially in one process. This is
// the first distributed-inference harness before remote transport exists.
type Simulator struct {
	workers map[string]Worker
}

func NewSimulator(workers map[string]Worker) (*Simulator, error) {
	if len(workers) == 0 {
		return nil, fmt.Errorf("no workers")
	}
	copyWorkers := make(map[string]Worker, len(workers))
	for id, worker := range workers {
		if id == "" {
			return nil, fmt.Errorf("empty worker id")
		}
		if isNilWorker(worker) {
			return nil, fmt.Errorf("nil worker %q", id)
		}
		copyWorkers[id] = worker
	}
	return &Simulator{workers: copyWorkers}, nil
}

func (s *Simulator) Execute(ctx context.Context, routes []router.Route, initial protocol.ShardExecutionRequest, totalLayers int) (protocol.ShardExecutionResponse, error) {
	if s == nil {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("nil simulator")
	}
	if err := ctx.Err(); err != nil {
		return protocol.ShardExecutionResponse{}, err
	}
	if len(routes) == 0 {
		return protocol.ShardExecutionResponse{}, fmt.Errorf("no routes")
	}
	ordered := append([]router.Route(nil), routes...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Shard.StartLayer < ordered[j].Shard.StartLayer })
	req := initial
	var resp protocol.ShardExecutionResponse
	for i, route := range ordered {
		if err := ctx.Err(); err != nil {
			return protocol.ShardExecutionResponse{}, err
		}
		worker, ok := s.workers[route.Peer.ID]
		if !ok {
			return protocol.ShardExecutionResponse{}, fmt.Errorf("no worker for peer %q", route.Peer.ID)
		}
		req.Shard = route.Shard
		if err := req.Validate(totalLayers); err != nil {
			return protocol.ShardExecutionResponse{}, fmt.Errorf("route %d request: %w", i, err)
		}
		var err error
		resp, err = worker.ExecuteShard(ctx, req)
		if err != nil {
			return protocol.ShardExecutionResponse{}, fmt.Errorf("route %d peer %s: %w", i, route.Peer.ID, err)
		}
		if err := resp.Validate(); err != nil {
			return protocol.ShardExecutionResponse{}, fmt.Errorf("route %d response: %w", i, err)
		}
		if resp.PeerID != route.Peer.ID {
			return protocol.ShardExecutionResponse{}, fmt.Errorf("route %d response peer=%q, want %q", i, resp.PeerID, route.Peer.ID)
		}
		if resp.SessionID != req.SessionID || resp.RequestID != req.RequestID || resp.Position != req.Position || resp.HiddenSize != req.HiddenSize {
			return protocol.ShardExecutionResponse{}, fmt.Errorf("route %d response metadata mismatch", i)
		}
		req.Activation = append([]float32(nil), resp.Activation...)
	}
	return resp, nil
}

func isNilWorker(worker Worker) bool {
	if worker == nil {
		return true
	}
	v := reflect.ValueOf(worker)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func PeerWorkers(peers []cluster.Peer, worker Worker) map[string]Worker {
	workers := make(map[string]Worker, len(peers))
	for _, peer := range peers {
		workers[peer.ID] = worker
	}
	return workers
}

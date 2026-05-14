package sim

import (
	"context"
	"fmt"

	"github.com/rcarmo/go-exotic/internal/protocol"
	"github.com/rcarmo/go-exotic/internal/router"
)

// TokenProjector maps the final activation after all shard routes into one
// token. It is synthetic for now; real logits/LM-head work remains in the
// go-pherence runtime boundary.
type TokenProjector func([]float32) (int, error)

// GenerateTokens runs a synthetic decode loop through routed shards. It is the
// first end-to-end N-shard token-generation harness and intentionally does not
// claim numerical parity with go-pherence yet.
func (s *Simulator) GenerateTokens(ctx context.Context, routes []router.Route, base protocol.ShardExecutionRequest, totalLayers, maxTokens int, project TokenProjector) ([]int, error) {
	if s == nil {
		return nil, fmt.Errorf("nil simulator")
	}
	if maxTokens < 0 {
		return nil, fmt.Errorf("maxTokens %d out of range", maxTokens)
	}
	maxInt := int(^uint(0) >> 1)
	if maxTokens > 0 && base.Position > maxInt-(maxTokens-1) {
		return nil, fmt.Errorf("generation positions overflow: start=%d count=%d", base.Position, maxTokens)
	}
	if project == nil {
		return nil, fmt.Errorf("nil token projector")
	}
	if len(base.Activation) != base.HiddenSize || base.HiddenSize <= 0 {
		return nil, fmt.Errorf("invalid base activation shape hidden=%d len=%d", base.HiddenSize, len(base.Activation))
	}
	out := make([]int, 0, maxTokens)
	activation := append([]float32(nil), base.Activation...)
	for step := 0; step < maxTokens; step++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		req := base
		req.Position = base.Position + step
		req.Activation = append([]float32(nil), activation...)
		resp, err := s.Execute(ctx, routes, req, totalLayers)
		if err != nil {
			return nil, err
		}
		tok, err := project(resp.Activation)
		if err != nil {
			return nil, err
		}
		out = append(out, tok)
		activation = append([]float32(nil), resp.Activation...)
	}
	return out, nil
}

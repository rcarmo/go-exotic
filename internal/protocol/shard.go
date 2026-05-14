package protocol

import (
	"fmt"
	"strings"

	"github.com/rcarmo/go-exotic/internal/exotic"
)

// ShardExecutionRequest is the wire contract for executing one contiguous layer
// range on a peer. Activation is a row-major float32 vector owned by the caller;
// the receiver must return a new activation and must not mutate input bytes.
type ShardExecutionRequest struct {
	SessionID  string       `json:"session_id"`
	RequestID  string       `json:"request_id"`
	ModelID    string       `json:"model_id"`
	Shard      exotic.Shard `json:"shard"`
	Position   int          `json:"position"`
	HiddenSize int          `json:"hidden_size"`
	Activation []float32    `json:"activation"`
}

// ShardExecutionResponse carries the transformed activation for the next shard.
type ShardExecutionResponse struct {
	SessionID  string    `json:"session_id"`
	RequestID  string    `json:"request_id"`
	PeerID     string    `json:"peer_id"`
	Position   int       `json:"position"`
	HiddenSize int       `json:"hidden_size"`
	Activation []float32 `json:"activation"`
}

func (r ShardExecutionRequest) Validate(totalLayers int) error {
	if strings.TrimSpace(r.SessionID) == "" {
		return fmt.Errorf("empty session id")
	}
	if strings.TrimSpace(r.RequestID) == "" {
		return fmt.Errorf("empty request id")
	}
	if strings.TrimSpace(r.ModelID) == "" {
		return fmt.Errorf("empty model id")
	}
	if err := r.Shard.Validate(totalLayers); err != nil {
		return err
	}
	if r.Position < 0 {
		return fmt.Errorf("position %d out of range", r.Position)
	}
	if r.HiddenSize <= 0 {
		return fmt.Errorf("hidden size %d out of range", r.HiddenSize)
	}
	if len(r.Activation) != r.HiddenSize {
		return fmt.Errorf("activation len=%d, want hidden size=%d", len(r.Activation), r.HiddenSize)
	}
	return nil
}

func (r ShardExecutionResponse) Validate() error {
	if strings.TrimSpace(r.SessionID) == "" {
		return fmt.Errorf("empty session id")
	}
	if strings.TrimSpace(r.RequestID) == "" {
		return fmt.Errorf("empty request id")
	}
	if strings.TrimSpace(r.PeerID) == "" {
		return fmt.Errorf("empty peer id")
	}
	if r.Position < 0 {
		return fmt.Errorf("position %d out of range", r.Position)
	}
	if r.HiddenSize <= 0 {
		return fmt.Errorf("hidden size %d out of range", r.HiddenSize)
	}
	if len(r.Activation) != r.HiddenSize {
		return fmt.Errorf("activation len=%d, want hidden size=%d", len(r.Activation), r.HiddenSize)
	}
	return nil
}

package protocol

import "github.com/rcarmo/go-exotic/internal/exotic"

// ShardExecutionHTTPBridgeRequest is the HTTP transport form of a shard
// execution request. It carries activations as an explicit flat f32 payload
// instead of JSON float arrays.
type ShardExecutionHTTPBridgeRequest struct {
	SessionID  string            `json:"session_id"`
	RequestID  string            `json:"request_id"`
	ModelID    string            `json:"model_id"`
	Shard      exotic.Shard      `json:"shard"`
	Position   int               `json:"position"`
	HiddenSize int               `json:"hidden_size"`
	Activation ActivationPayload `json:"activation"`
}

// ShardExecutionHTTPBridgeResponse is the HTTP transport form of a shard
// execution response.
type ShardExecutionHTTPBridgeResponse struct {
	SessionID  string            `json:"session_id"`
	RequestID  string            `json:"request_id"`
	PeerID     string            `json:"peer_id"`
	Position   int               `json:"position"`
	HiddenSize int               `json:"hidden_size"`
	Activation ActivationPayload `json:"activation"`
}

func NewShardExecutionHTTPBridgeRequest(req ShardExecutionRequest) (ShardExecutionHTTPBridgeRequest, error) {
	payload, err := NewActivationPayload(req.Activation)
	if err != nil {
		return ShardExecutionHTTPBridgeRequest{}, err
	}
	return ShardExecutionHTTPBridgeRequest{SessionID: req.SessionID, RequestID: req.RequestID, ModelID: req.ModelID, Shard: req.Shard, Position: req.Position, HiddenSize: req.HiddenSize, Activation: payload}, nil
}

func (r ShardExecutionHTTPBridgeRequest) ShardExecutionRequest() (ShardExecutionRequest, error) {
	activation, err := r.Activation.Float32()
	if err != nil {
		return ShardExecutionRequest{}, err
	}
	return ShardExecutionRequest{SessionID: r.SessionID, RequestID: r.RequestID, ModelID: r.ModelID, Shard: r.Shard, Position: r.Position, HiddenSize: r.HiddenSize, Activation: activation}, nil
}

func NewShardExecutionHTTPBridgeResponse(resp ShardExecutionResponse) (ShardExecutionHTTPBridgeResponse, error) {
	payload, err := NewActivationPayload(resp.Activation)
	if err != nil {
		return ShardExecutionHTTPBridgeResponse{}, err
	}
	return ShardExecutionHTTPBridgeResponse{SessionID: resp.SessionID, RequestID: resp.RequestID, PeerID: resp.PeerID, Position: resp.Position, HiddenSize: resp.HiddenSize, Activation: payload}, nil
}

func (r ShardExecutionHTTPBridgeResponse) ShardExecutionResponse() (ShardExecutionResponse, error) {
	activation, err := r.Activation.Float32()
	if err != nil {
		return ShardExecutionResponse{}, err
	}
	return ShardExecutionResponse{SessionID: r.SessionID, RequestID: r.RequestID, PeerID: r.PeerID, Position: r.Position, HiddenSize: r.HiddenSize, Activation: activation}, nil
}

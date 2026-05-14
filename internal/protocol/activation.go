package protocol

import (
	"encoding/binary"
	"fmt"
	"math"
)

const ActivationEncodingF32LE = "f32le"

// ActivationPayload is the transport-friendly representation of one activation
// vector. It is intentionally flat for the first shard protocol; tensor shape
// metadata can be expanded later when batched/paged execution is introduced.
type ActivationPayload struct {
	Encoding   string `json:"encoding"`
	HiddenSize int    `json:"hidden_size"`
	Data       []byte `json:"data"`
}

func NewActivationPayload(values []float32) (ActivationPayload, error) {
	if len(values) == 0 {
		return ActivationPayload{}, fmt.Errorf("empty activation")
	}
	dataLen, ok := checkedMulInt(len(values), 4)
	if !ok {
		return ActivationPayload{}, fmt.Errorf("activation byte length overflow")
	}
	data := make([]byte, dataLen)
	for i, v := range values {
		binary.LittleEndian.PutUint32(data[i*4:(i+1)*4], math.Float32bits(v))
	}
	return ActivationPayload{Encoding: ActivationEncodingF32LE, HiddenSize: len(values), Data: data}, nil
}

func (p ActivationPayload) Float32() ([]float32, error) {
	if p.Encoding != ActivationEncodingF32LE {
		return nil, fmt.Errorf("unsupported activation encoding %q", p.Encoding)
	}
	if p.HiddenSize <= 0 {
		return nil, fmt.Errorf("hidden size %d out of range", p.HiddenSize)
	}
	want, ok := checkedMulInt(p.HiddenSize, 4)
	if !ok {
		return nil, fmt.Errorf("activation byte length overflow")
	}
	if len(p.Data) != want {
		return nil, fmt.Errorf("activation bytes=%d, want %d", len(p.Data), want)
	}
	out := make([]float32, p.HiddenSize)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(p.Data[i*4 : (i+1)*4]))
	}
	return out, nil
}

func checkedMulInt(a, b int) (int, bool) {
	if a < 0 || b < 0 {
		return 0, false
	}
	if a == 0 || b == 0 {
		return 0, true
	}
	maxInt := int(^uint(0) >> 1)
	if a > maxInt/b {
		return 0, false
	}
	return a * b, true
}

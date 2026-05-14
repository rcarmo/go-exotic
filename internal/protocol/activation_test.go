package protocol

import (
	"encoding/json"
	"testing"
)

func TestActivationPayloadRoundTrip(t *testing.T) {
	payload, err := NewActivationPayload([]float32{1.25, -2.5, 3})
	if err != nil {
		t.Fatalf("NewActivationPayload: %v", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded ActivationPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got, err := decoded.Float32()
	if err != nil {
		t.Fatalf("Float32: %v", err)
	}
	want := []float32{1.25, -2.5, 3}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%f want %f", i, got[i], want[i])
		}
	}
}

func TestActivationPayloadRejectsMalformedInputs(t *testing.T) {
	if _, err := NewActivationPayload(nil); err == nil {
		t.Fatal("accepted empty activation")
	}
	bad := []ActivationPayload{
		{},
		{Encoding: "f16", HiddenSize: 1, Data: []byte{0, 0}},
		{Encoding: ActivationEncodingF32LE, HiddenSize: 0, Data: nil},
		{Encoding: ActivationEncodingF32LE, HiddenSize: 2, Data: []byte{0, 0, 0, 0}},
	}
	for i, tc := range bad {
		if _, err := tc.Float32(); err == nil {
			t.Fatalf("case %d accepted malformed payload: %+v", i, tc)
		}
	}
}

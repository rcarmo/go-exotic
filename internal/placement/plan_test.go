package placement

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/rcarmo/go-exotic/internal/exotic"
)

func TestPlanLayerShards(t *testing.T) {
	got, err := PlanLayerShards([]exotic.Device{{ID: "b", MemoryGB: 1}, {ID: "a", MemoryGB: 3}}, 8)
	if err != nil {
		t.Fatalf("PlanLayerShards: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d want 2", len(got))
	}
	if got[0].DeviceID != "a" || got[0].StartLayer != 0 || got[0].EndLayer < got[0].StartLayer {
		t.Fatalf("unexpected first shard: %+v", got[0])
	}
	if got[1].DeviceID != "b" || got[1].StartLayer != got[0].EndLayer+1 || got[1].EndLayer != 7 {
		t.Fatalf("unexpected second shard: %+v", got[1])
	}
}

func TestPlanLayerShardsMemoryWeightedEdgeCases(t *testing.T) {
	cases := []struct {
		name    string
		devices []exotic.Device
		layers  int
	}{
		{"one layer per tiny device", []exotic.Device{{ID: "a", MemoryGB: 0.001}, {ID: "b", MemoryGB: 999}}, 2},
		{"many devices exact coverage", []exotic.Device{{ID: "c", MemoryGB: 1}, {ID: "a", MemoryGB: 1}, {ID: "b", MemoryGB: 1}}, 3},
		{"large skew keeps tail contiguous", []exotic.Device{{ID: "a", MemoryGB: 1000}, {ID: "b", MemoryGB: 1}, {ID: "c", MemoryGB: 1}}, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := PlanLayerShards(tc.devices, tc.layers)
			if err != nil {
				t.Fatalf("PlanLayerShards: %v", err)
			}
			if err := ValidatePlan(got, tc.layers); err != nil {
				t.Fatalf("ValidatePlan: %v", err)
			}
			if len(got) != len(tc.devices) {
				t.Fatalf("shards=%d devices=%d", len(got), len(tc.devices))
			}
		})
	}
}

func TestPlanLayerShardsRejectsMalformedInputs(t *testing.T) {
	cases := []struct {
		name    string
		devices []exotic.Device
		layers  int
	}{
		{"no devices", nil, 1},
		{"invalid layers", []exotic.Device{{ID: "x", MemoryGB: 1}}, 0},
		{"too many devices", []exotic.Device{{ID: "a", MemoryGB: 1}, {ID: "b", MemoryGB: 1}}, 1},
		{"empty id", []exotic.Device{{ID: "", MemoryGB: 1}}, 1},
		{"zero memory", []exotic.Device{{ID: "x", MemoryGB: 0}}, 1},
		{"negative memory", []exotic.Device{{ID: "x", MemoryGB: -1}}, 1},
		{"nan memory", []exotic.Device{{ID: "x", MemoryGB: math.NaN()}}, 1},
		{"inf memory", []exotic.Device{{ID: "x", MemoryGB: math.Inf(1)}}, 1},
		{"duplicate", []exotic.Device{{ID: "x", MemoryGB: 1}, {ID: "x", MemoryGB: 2}}, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := PlanLayerShards(tc.devices, tc.layers); err == nil {
				t.Fatalf("accepted malformed input")
			}
		})
	}
}

func TestPlanJSON(t *testing.T) {
	plan, err := NewPlan([]exotic.Device{{ID: "local", MemoryGB: 1}}, 2)
	if err != nil {
		t.Fatalf("NewPlan: %v", err)
	}
	data, err := plan.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if !strings.Contains(string(data), `"total_layers": 2`) || !strings.Contains(string(data), `"device_id": "local"`) {
		t.Fatalf("unexpected JSON: %s", data)
	}
	var decoded Plan
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}
	if err := ValidatePlan(decoded.Shards, decoded.TotalLayers); err != nil {
		t.Fatalf("decoded plan invalid: %v", err)
	}
}

func TestValidatePlanRejectsGapsOverlapsAndDuplicates(t *testing.T) {
	cases := [][]exotic.Shard{
		{{DeviceID: "a", StartLayer: 1, EndLayer: 1}},
		{{DeviceID: "a", StartLayer: 0, EndLayer: 0}, {DeviceID: "b", StartLayer: 2, EndLayer: 2}},
		{{DeviceID: "a", StartLayer: 0, EndLayer: 1}, {DeviceID: "b", StartLayer: 1, EndLayer: 2}},
		{{DeviceID: "a", StartLayer: 0, EndLayer: 0}, {DeviceID: "a", StartLayer: 1, EndLayer: 1}},
	}
	for i, tc := range cases {
		if err := ValidatePlan(tc, 3); err == nil {
			t.Fatalf("case %d accepted invalid plan: %+v", i, tc)
		}
	}
}

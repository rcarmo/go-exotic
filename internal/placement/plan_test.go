package placement

import (
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

func TestPlanLayerShardsRejectsMalformedInputs(t *testing.T) {
	if _, err := PlanLayerShards(nil, 1); err == nil {
		t.Fatal("accepted no devices")
	}
	if _, err := PlanLayerShards([]exotic.Device{{ID: "x", MemoryGB: 1}}, 0); err == nil {
		t.Fatal("accepted invalid layer count")
	}
	if _, err := PlanLayerShards([]exotic.Device{{ID: "", MemoryGB: 1}}, 1); err == nil {
		t.Fatal("accepted empty device id")
	}
	if _, err := PlanLayerShards([]exotic.Device{{ID: "x", MemoryGB: 0}}, 1); err == nil {
		t.Fatal("accepted zero memory")
	}
	if _, err := PlanLayerShards([]exotic.Device{{ID: "x", MemoryGB: 1}, {ID: "x", MemoryGB: 2}}, 2); err == nil {
		t.Fatal("accepted duplicate device id")
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

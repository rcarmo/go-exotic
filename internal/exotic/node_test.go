package exotic

import "testing"

func TestPlanLayerShards(t *testing.T) {
	got, err := PlanLayerShards([]Device{{ID: "b", MemoryGB: 1}, {ID: "a", MemoryGB: 3}}, 8)
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
	for _, shard := range got {
		if err := shard.Validate(8); err != nil {
			t.Fatalf("invalid shard %+v: %v", shard, err)
		}
	}
}

func TestPlanLayerShardsRejectsMalformedInputs(t *testing.T) {
	if _, err := PlanLayerShards(nil, 1); err == nil {
		t.Fatal("accepted no devices")
	}
	if _, err := PlanLayerShards([]Device{{ID: "x", MemoryGB: 1}}, 0); err == nil {
		t.Fatal("accepted invalid layer count")
	}
	if _, err := PlanLayerShards([]Device{{ID: "", MemoryGB: 1}}, 1); err == nil {
		t.Fatal("accepted empty device id")
	}
	if _, err := PlanLayerShards([]Device{{ID: "x", MemoryGB: 0}}, 1); err == nil {
		t.Fatal("accepted zero memory")
	}
}

package exotic

import "testing"

func TestShardValidate(t *testing.T) {
	if err := (Shard{DeviceID: "local", StartLayer: 0, EndLayer: 3}).Validate(4); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	cases := []Shard{
		{},
		{DeviceID: "x", StartLayer: -1, EndLayer: 0},
		{DeviceID: "x", StartLayer: 2, EndLayer: 1},
		{DeviceID: "x", StartLayer: 0, EndLayer: 4},
	}
	for i, tc := range cases {
		if err := tc.Validate(4); err == nil {
			t.Fatalf("case %d accepted invalid shard: %+v", i, tc)
		}
	}
}

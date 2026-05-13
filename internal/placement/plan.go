package placement

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/rcarmo/go-exotic/internal/exotic"
)

// Plan is a serializable placement preview.
type Plan struct {
	TotalLayers int            `json:"total_layers"`
	Shards      []exotic.Shard `json:"shards"`
}

// PlanLayerShards is a small deterministic baseline placement policy inspired
// by exo's master placement code. It assigns contiguous layer ranges weighted
// by advertised memory. Later implementations will replace this with live peer
// discovery and go-pherence backend measurements.
func PlanLayerShards(devices []exotic.Device, totalLayers int) ([]exotic.Shard, error) {
	if totalLayers <= 0 {
		return nil, fmt.Errorf("invalid total layers %d", totalLayers)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices")
	}
	if len(devices) > totalLayers {
		return nil, fmt.Errorf("devices=%d exceeds total layers=%d", len(devices), totalLayers)
	}
	devs := append([]exotic.Device(nil), devices...)
	sort.Slice(devs, func(i, j int) bool { return devs[i].ID < devs[j].ID })
	totalWeight := 0.0
	seen := make(map[string]struct{}, len(devs))
	for _, d := range devs {
		if d.ID == "" {
			return nil, fmt.Errorf("device with empty id")
		}
		if _, ok := seen[d.ID]; ok {
			return nil, fmt.Errorf("duplicate device id %q", d.ID)
		}
		seen[d.ID] = struct{}{}
		if d.MemoryGB <= 0 || math.IsNaN(d.MemoryGB) || math.IsInf(d.MemoryGB, 0) {
			return nil, fmt.Errorf("device %s has invalid memory %.2f", d.ID, d.MemoryGB)
		}
		totalWeight += d.MemoryGB
		if math.IsInf(totalWeight, 0) {
			return nil, fmt.Errorf("device memory weights overflow")
		}
	}
	if totalWeight <= 0 || math.IsNaN(totalWeight) || math.IsInf(totalWeight, 0) {
		return nil, fmt.Errorf("invalid total device memory weight %.2f", totalWeight)
	}
	start := 0
	shards := make([]exotic.Shard, 0, len(devs))
	for i, d := range devs {
		remainingLayers := totalLayers - start
		remainingDevices := len(devs) - i
		count := remainingLayers
		if remainingDevices > 1 {
			share := float64(totalLayers) * d.MemoryGB / totalWeight
			if share > float64(math.MaxInt) {
				return nil, fmt.Errorf("device %s layer share overflows int", d.ID)
			}
			count = int(share)
			if count < 1 {
				count = 1
			}
			maxForThis := remainingLayers - (remainingDevices - 1)
			if count > maxForThis {
				count = maxForThis
			}
		}
		end := start + count - 1
		shards = append(shards, exotic.Shard{DeviceID: d.ID, StartLayer: start, EndLayer: end})
		start = end + 1
	}
	if err := ValidatePlan(shards, totalLayers); err != nil {
		return nil, err
	}
	return shards, nil
}

// NewPlan builds a validated serializable placement plan.
func NewPlan(devices []exotic.Device, totalLayers int) (Plan, error) {
	shards, err := PlanLayerShards(devices, totalLayers)
	if err != nil {
		return Plan{}, err
	}
	return Plan{TotalLayers: totalLayers, Shards: shards}, nil
}

// JSON returns a stable indented JSON representation of the placement plan.
func (p Plan) JSON() ([]byte, error) {
	if err := ValidatePlan(p.Shards, p.TotalLayers); err != nil {
		return nil, err
	}
	return json.MarshalIndent(p, "", "  ")
}

// ValidatePlan checks that shards exactly cover [0,totalLayers) without gaps or overlaps.
func ValidatePlan(shards []exotic.Shard, totalLayers int) error {
	if totalLayers <= 0 {
		return fmt.Errorf("invalid total layers %d", totalLayers)
	}
	if len(shards) == 0 {
		return fmt.Errorf("no shards")
	}
	wantStart := 0
	seen := make(map[string]struct{}, len(shards))
	for i, shard := range shards {
		if err := shard.Validate(totalLayers); err != nil {
			return fmt.Errorf("shard %d: %w", i, err)
		}
		if _, ok := seen[shard.DeviceID]; ok {
			return fmt.Errorf("duplicate shard device id %q", shard.DeviceID)
		}
		seen[shard.DeviceID] = struct{}{}
		if shard.StartLayer != wantStart {
			return fmt.Errorf("shard %d starts at %d, want %d", i, shard.StartLayer, wantStart)
		}
		wantStart = shard.EndLayer + 1
	}
	if wantStart != totalLayers {
		return fmt.Errorf("plan covers %d layers, want %d", wantStart, totalLayers)
	}
	return nil
}

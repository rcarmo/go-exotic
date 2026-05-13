package exotic

import (
	"fmt"
	"sort"
)

// Device describes a local or remote worker's inference capacity.
type Device struct {
	ID       string
	MemoryGB float64
	Backend  string
}

// Shard identifies the inclusive layer range assigned to a worker.
type Shard struct {
	DeviceID   string
	StartLayer int
	EndLayer   int
}

// Validate checks the shard range before runtime execution.
func (s Shard) Validate(totalLayers int) error {
	if s.DeviceID == "" {
		return fmt.Errorf("empty device id")
	}
	if totalLayers <= 0 {
		return fmt.Errorf("invalid total layers %d", totalLayers)
	}
	if s.StartLayer < 0 || s.EndLayer < s.StartLayer || s.EndLayer >= totalLayers {
		return fmt.Errorf("invalid shard range [%d,%d] for %d layers", s.StartLayer, s.EndLayer, totalLayers)
	}
	return nil
}

// PlanLayerShards is a small deterministic baseline placement policy inspired
// by exo's master placement code. It assigns contiguous layer ranges weighted
// by advertised memory. Later implementations will replace this with live peer
// discovery and go-pherence backend measurements.
func PlanLayerShards(devices []Device, totalLayers int) ([]Shard, error) {
	if totalLayers <= 0 {
		return nil, fmt.Errorf("invalid total layers %d", totalLayers)
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no devices")
	}
	devs := append([]Device(nil), devices...)
	sort.Slice(devs, func(i, j int) bool { return devs[i].ID < devs[j].ID })
	totalWeight := 0.0
	for _, d := range devs {
		if d.ID == "" {
			return nil, fmt.Errorf("device with empty id")
		}
		if d.MemoryGB <= 0 {
			return nil, fmt.Errorf("device %s has invalid memory %.2f", d.ID, d.MemoryGB)
		}
		totalWeight += d.MemoryGB
	}
	start := 0
	shards := make([]Shard, 0, len(devs))
	for i, d := range devs {
		remainingLayers := totalLayers - start
		remainingDevices := len(devs) - i
		count := remainingLayers
		if remainingDevices > 1 {
			count = int(float64(totalLayers) * d.MemoryGB / totalWeight)
			if count < 1 {
				count = 1
			}
			maxForThis := remainingLayers - (remainingDevices - 1)
			if count > maxForThis {
				count = maxForThis
			}
		}
		end := start + count - 1
		shards = append(shards, Shard{DeviceID: d.ID, StartLayer: start, EndLayer: end})
		start = end + 1
	}
	return shards, nil
}

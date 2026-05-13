package exotic

import "fmt"

// Device describes a local or remote worker's inference capacity.
type Device struct {
	ID       string  `json:"id"`
	MemoryGB float64 `json:"memory_gb"`
	Backend  string  `json:"backend,omitempty"`
}

// Shard identifies the inclusive layer range assigned to a worker.
type Shard struct {
	DeviceID   string `json:"device_id"`
	StartLayer int    `json:"start_layer"`
	EndLayer   int    `json:"end_layer"`
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

package exotic

import "fmt"

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

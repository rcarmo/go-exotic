package integration

import (
	"os"
	"path/filepath"
	"testing"
)

func smolLM2FixturePath(t *testing.T) string {
	t.Helper()
	candidates := []string{}
	if env := os.Getenv("GO_EXOTIC_SMOLLM2_MODEL"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates, filepath.Clean("../../../go-pherence/models/smollm2-135m"))
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "config.json")); err == nil {
			return candidate
		}
	}
	t.Skipf("SmolLM2 fixture unavailable; set GO_EXOTIC_SMOLLM2_MODEL or place fixture at %s", candidates[len(candidates)-1])
	return ""
}

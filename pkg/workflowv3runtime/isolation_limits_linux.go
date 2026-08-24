//go:build linux

package workflowv3runtime

import (
	"fmt"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"golang.org/x/sys/unix"
)

func applyIsolationLimits(policy workflowv3.IsolationPolicy) error {
	if policy.MaxOutputBytes <= 0 {
		return fmt.Errorf("max output bytes must be positive")
	}
	limits := []struct {
		resource int
		value    uint64
		name     string
	}{
		{unix.RLIMIT_NOFILE, 64, "file-descriptor"},
		{unix.RLIMIT_FSIZE, uint64(policy.MaxOutputBytes), "file-size"},
	}
	for _, limit := range limits {
		if err := unix.Setrlimit(limit.resource, &unix.Rlimit{Cur: limit.value, Max: limit.value}); err != nil {
			return fmt.Errorf("set %s limit: %w", limit.name, err)
		}
	}
	return nil
}

package scheduler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConfigValidateRejectsZeroValues(t *testing.T) {
	cfg := Config{}

	err := cfg.Validate()
	require.Error(t, err)
}

func TestConfigValidateAcceptsPositiveValues(t *testing.T) {
	cfg := Config{
		MaxWorkers:           4,
		PollInterval:         1 * time.Second,
		DefaultLeaseDuration: 30 * time.Second,
	}

	require.NoError(t, cfg.Validate())
}

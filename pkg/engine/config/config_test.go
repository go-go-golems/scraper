package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidateAcceptsReasonableDefaults(t *testing.T) {
	cfg := Config{
		Paths: Paths{
			EngineDB: "state/engine.db",
			SitesDir: "state/sites",
		},
		Worker: Worker{
			WorkerID:             "worker-1",
			MaxConcurrentOps:     4,
			PollInterval:         1 * time.Second,
			DefaultLeaseDuration: 30 * time.Second,
		},
		HTTP: HTTP{
			UserAgent: "scraper/dev",
			Timeout:   15 * time.Second,
		},
	}

	require.NoError(t, cfg.Validate())
}

package config

import (
	"fmt"
	"time"
)

type Paths struct {
	EngineDB string
	SitesDir string
}

type Worker struct {
	WorkerID             string
	MaxConcurrentOps     int
	PollInterval         time.Duration
	DefaultLeaseDuration time.Duration
}

type HTTP struct {
	UserAgent string
	Timeout   time.Duration
}

type Config struct {
	Paths  Paths
	Worker Worker
	HTTP   HTTP
}

func (c Config) Validate() error {
	if c.Paths.EngineDB == "" {
		return fmt.Errorf("engine db path is required")
	}
	if c.Paths.SitesDir == "" {
		return fmt.Errorf("sites dir is required")
	}
	if c.Worker.WorkerID == "" {
		return fmt.Errorf("worker id is required")
	}
	if c.Worker.MaxConcurrentOps <= 0 {
		return fmt.Errorf("max concurrent ops must be > 0")
	}
	if c.Worker.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be > 0")
	}
	if c.Worker.DefaultLeaseDuration <= 0 {
		return fmt.Errorf("default lease duration must be > 0")
	}
	if c.HTTP.Timeout <= 0 {
		return fmt.Errorf("http timeout must be > 0")
	}

	return nil
}

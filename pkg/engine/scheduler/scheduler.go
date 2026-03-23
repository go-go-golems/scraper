package scheduler

import (
	"fmt"
	"time"
)

type Config struct {
	MaxWorkers           int
	PollInterval         time.Duration
	DefaultLeaseDuration time.Duration
}

func (c Config) Validate() error {
	if c.MaxWorkers <= 0 {
		return fmt.Errorf("max workers must be > 0")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll interval must be > 0")
	}
	if c.DefaultLeaseDuration <= 0 {
		return fmt.Errorf("default lease duration must be > 0")
	}

	return nil
}

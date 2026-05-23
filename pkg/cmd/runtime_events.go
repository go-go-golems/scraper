package cmd

import (
	"fmt"

	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	"github.com/spf13/cobra"
)

type runtimeEventOptions struct {
	backend            string
	redisAddress       string
	redisUsername      string
	redisPassword      string
	redisDB            int
	redisConsumer      string
	redisConsumerGroup string
	redisStreamMaxLen  int64
	recentEventLimit   int
	sessionstreamDB    string
}

func addRuntimeEventFlags(cmd *cobra.Command, options *runtimeEventOptions, includeConsumerGroup bool, includeRecentLimit bool) {
	cmd.Flags().StringVar(&options.backend, "events-backend", string(runtimeevents.BackendOff), "Runtime event backend: off, gochannel, or redis")
	cmd.Flags().StringVar(&options.redisAddress, "events-redis-address", runtimeevents.DefaultRedisAddress, "Redis address for runtime event transport")
	cmd.Flags().StringVar(&options.redisUsername, "events-redis-username", "", "Redis username for runtime event transport")
	cmd.Flags().StringVar(&options.redisPassword, "events-redis-password", "", "Redis password for runtime event transport")
	cmd.Flags().IntVar(&options.redisDB, "events-redis-db", 0, "Redis logical database for runtime event transport")
	cmd.Flags().StringVar(&options.redisConsumer, "events-redis-consumer", "", "Explicit Redis consumer name for runtime event subscriptions")
	cmd.Flags().Int64Var(&options.redisStreamMaxLen, "events-redis-stream-maxlen", runtimeevents.DefaultRedisStreamMaxLen, "Approximate Redis stream retention length for runtime events")
	if includeConsumerGroup {
		cmd.Flags().StringVar(&options.redisConsumerGroup, "events-redis-consumer-group", runtimeevents.DefaultRedisConsumerGroup, "Redis consumer group used by the API runtime event subscriber")
	}
	if includeRecentLimit {
		cmd.Flags().IntVar(&options.recentEventLimit, "events-recent-limit", runtimeevents.DefaultRecentEventLimit, "Recent runtime events kept per sessionstream snapshot")
		cmd.Flags().StringVar(&options.sessionstreamDB, "events-sessionstream-db", "state/runtime-events-sessionstream.db", "SQLite database path for runtime event sessionstream hydration snapshots")
	}
}

func (o *runtimeEventOptions) publisherConfig() (runtimeevents.Config, error) {
	return o.config(false)
}

func (o *runtimeEventOptions) pubSubConfig() (runtimeevents.Config, error) {
	return o.config(true)
}

func (o *runtimeEventOptions) config(includeConsumerGroup bool) (runtimeevents.Config, error) {
	cfg := runtimeevents.Config{
		Backend:           runtimeevents.Backend(o.backend),
		RedisAddress:      o.redisAddress,
		RedisUsername:     o.redisUsername,
		RedisPassword:     o.redisPassword,
		RedisDB:           o.redisDB,
		RedisConsumer:     o.redisConsumer,
		RedisStreamMaxLen: o.redisStreamMaxLen,
	}
	if includeConsumerGroup {
		cfg.RedisConsumerGroup = o.redisConsumerGroup
	}
	if err := cfg.Validate(); err != nil {
		return runtimeevents.Config{}, fmt.Errorf("invalid runtime event config: %w", err)
	}
	return cfg.Normalize(), nil
}

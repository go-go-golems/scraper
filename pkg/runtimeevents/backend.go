package runtimeevents

import (
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/redis/go-redis/v9"
)

type Backend string

const (
	BackendOff       Backend = "off"
	BackendGoChannel Backend = "gochannel"
	BackendRedis     Backend = "redis"
)

const (
	DefaultRedisAddress       = "127.0.0.1:6379"
	DefaultRedisConsumerGroup = "scraper-api"
	DefaultRecentEventLimit   = 256
	DefaultRedisStreamMaxLen  = 10000
)

type Config struct {
	Backend            Backend
	Topic              string
	GoChannel          *gochannel.GoChannel
	RedisAddress       string
	RedisUsername      string
	RedisPassword      string
	RedisDB            int
	RedisConsumer      string
	RedisConsumerGroup string
	RedisStreamMaxLen  int64
}

type Resources struct {
	Topic      string
	Publisher  message.Publisher
	Subscriber message.Subscriber
	closers    []func() error
}

func (c Config) Normalize() Config {
	if c.Backend == "" {
		c.Backend = BackendOff
	}
	if c.Topic == "" {
		c.Topic = TopicRuntimeEventsV1
	}
	if c.RedisAddress == "" {
		c.RedisAddress = DefaultRedisAddress
	}
	if c.RedisConsumerGroup == "" {
		c.RedisConsumerGroup = DefaultRedisConsumerGroup
	}
	if c.RedisStreamMaxLen <= 0 {
		c.RedisStreamMaxLen = DefaultRedisStreamMaxLen
	}
	return c
}

func (c Config) Validate() error {
	c = c.Normalize()
	switch c.Backend {
	case BackendOff, BackendGoChannel:
		return nil
	case BackendRedis:
		if c.RedisAddress == "" {
			return errors.New("runtime events redis address is required")
		}
		return nil
	default:
		return fmt.Errorf("unsupported runtime events backend %q", c.Backend)
	}
}

func OpenPublisher(cfg Config) (*Resources, error) {
	return open(cfg, false)
}

func OpenPublisherSubscriber(cfg Config) (*Resources, error) {
	return open(cfg, true)
}

func open(cfg Config, withSubscriber bool) (*Resources, error) {
	cfg = cfg.Normalize()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	switch cfg.Backend {
	case BackendOff:
		return &Resources{Topic: cfg.Topic, Publisher: nopPublisher{}}, nil
	case BackendGoChannel:
		ch := cfg.GoChannel
		if ch == nil {
			ch = NewGoChannelPubSub()
		}
		res := &Resources{
			Topic:     cfg.Topic,
			Publisher: ch,
			closers:   []func() error{ch.Close},
		}
		if withSubscriber {
			res.Subscriber = ch
		}
		return res, nil
	case BackendRedis:
		return openRedis(cfg, withSubscriber)
	default:
		return nil, fmt.Errorf("unsupported runtime events backend %q", cfg.Backend)
	}
}

func openRedis(cfg Config, withSubscriber bool) (*Resources, error) {
	logger := watermill.NopLogger{}
	publisherClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Username: cfg.RedisUsername,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client:        publisherClient,
		DefaultMaxlen: cfg.RedisStreamMaxLen,
	}, logger)
	if err != nil {
		_ = publisherClient.Close()
		return nil, fmt.Errorf("open redis runtime-event publisher: %w", err)
	}

	res := &Resources{
		Topic:     cfg.Topic,
		Publisher: publisher,
		closers:   []func() error{publisher.Close},
	}
	if !withSubscriber {
		return res, nil
	}

	subscriberClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Username: cfg.RedisUsername,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        subscriberClient,
		Consumer:      cfg.RedisConsumer,
		ConsumerGroup: cfg.RedisConsumerGroup,
		OldestId:      "$",
	}, logger)
	if err != nil {
		_ = publisher.Close()
		_ = subscriberClient.Close()
		return nil, fmt.Errorf("open redis runtime-event subscriber: %w", err)
	}

	res.Subscriber = subscriber
	res.closers = append(res.closers, subscriber.Close)
	return res, nil
}

func (r *Resources) EventPublisher() *Publisher {
	if r == nil || r.Publisher == nil {
		return nil
	}
	return NewPublisher(r.Publisher, r.Topic)
}

func (r *Resources) EventSubscriber() *Subscriber {
	if r == nil || r.Subscriber == nil {
		return nil
	}
	return NewSubscriber(r.Subscriber, r.Topic)
}

func (r *Resources) Close() error {
	if r == nil {
		return nil
	}

	var firstErr error
	for i := len(r.closers) - 1; i >= 0; i-- {
		if err := r.closers[i](); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	r.closers = nil
	return firstErr
}

type nopPublisher struct{}

func (nopPublisher) Publish(topic string, messages ...*message.Message) error {
	return nil
}

func (nopPublisher) Close() error {
	return nil
}

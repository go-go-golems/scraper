package model

import (
	"testing"
	"time"
)

func TestQueuePolicyNormalizeDefaults(t *testing.T) {
	policy := (QueuePolicy{}).Normalize()
	if policy.MaxInFlight != 1 {
		t.Fatalf("expected default MaxInFlight=1, got %d", policy.MaxInFlight)
	}
	if policy.RateLimit != nil {
		t.Fatalf("expected nil rate limit by default")
	}
}

func TestQueuePolicyNormalizeTokenBucketDefaults(t *testing.T) {
	policy := QueuePolicy{
		RateLimit: &RateLimitPolicy{
			RatePerSecond: 2,
			Burst:         3,
		},
	}.Normalize()

	if policy.MaxInFlight != 1 {
		t.Fatalf("expected normalized MaxInFlight=1, got %d", policy.MaxInFlight)
	}
	if policy.RateLimit == nil {
		t.Fatalf("expected rate limit to be retained")
	}
	if policy.RateLimit.Kind != RateLimitKindTokenBucket {
		t.Fatalf("expected default rate limit kind %q, got %q", RateLimitKindTokenBucket, policy.RateLimit.Kind)
	}
}

func TestQueuePolicyNormalizeDisablesInvalidLimiter(t *testing.T) {
	policy := QueuePolicy{
		MaxInFlight: 2,
		RateLimit: &RateLimitPolicy{
			Kind:          RateLimitKindTokenBucket,
			RatePerSecond: 0,
			Burst:         2,
		},
	}.Normalize()

	if policy.MaxInFlight != 2 {
		t.Fatalf("expected MaxInFlight to be preserved, got %d", policy.MaxInFlight)
	}
	if policy.RateLimit != nil {
		t.Fatalf("expected invalid limiter to normalize to nil")
	}
}

func TestOpSpecQueueWaitDurationUsesUpdatedAtForReadyOps(t *testing.T) {
	op := OpSpec{
		CreatedAt: time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 7, 12, 0, 10, 0, time.UTC),
	}

	wait := op.QueueWaitDuration(time.Date(2026, 4, 7, 12, 0, 25, 0, time.UTC))
	if wait != 15*time.Second {
		t.Fatalf("expected queue wait of 15s, got %s", wait)
	}
}

func TestOpSpecQueueWaitDurationUsesNextReadyAtForRetries(t *testing.T) {
	nextReadyAt := time.Date(2026, 4, 7, 12, 1, 0, 0, time.UTC)
	op := OpSpec{
		CreatedAt:   time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 7, 12, 0, 30, 0, time.UTC),
		NextReadyAt: &nextReadyAt,
	}

	wait := op.QueueWaitDuration(time.Date(2026, 4, 7, 12, 1, 45, 0, time.UTC))
	if wait != 45*time.Second {
		t.Fatalf("expected queue wait of 45s, got %s", wait)
	}
}

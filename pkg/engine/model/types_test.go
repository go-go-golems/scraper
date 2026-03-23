package model

import "testing"

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

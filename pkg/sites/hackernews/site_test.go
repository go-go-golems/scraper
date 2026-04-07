package hackernews

import (
	"testing"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/stretchr/testify/require"
)

func TestDefinitionSetsHTTPQueueRateLimit(t *testing.T) {
	def := Definition()

	policy, ok := def.QueuePolicies[model.QueueKey("site:hackernews:http")]
	require.True(t, ok, "expected hackernews http queue policy to be registered")

	policy = policy.Normalize()
	require.Equal(t, 1, policy.MaxInFlight)
	require.NotNil(t, policy.RateLimit)
	require.Equal(t, model.RateLimitKindTokenBucket, policy.RateLimit.Kind)
	require.Equal(t, 1.0, policy.RateLimit.RatePerSecond)
	require.Equal(t, 1, policy.RateLimit.Burst)
}

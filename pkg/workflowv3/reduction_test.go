package workflowv3

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReductionPartitionCanonicalRoundTripAndIdentity(t *testing.T) {
	partition, err := NewReductionPartition(
		"merge", "sha256:"+strings.Repeat("b", 64), "count/v1", 0, 0, 2,
		[]ManifestItem{
			{Key: "a", Value: manifestRef("count/v1", "a", "cas://a")},
			{Key: "b", Value: manifestRef("count/v1", "c", "cas://b")},
		},
	)
	require.NoError(t, err)
	body, err := EncodeReductionPartition(partition, 2)
	require.NoError(t, err)
	decoded, err := DecodeReductionPartition(body, 2)
	require.NoError(t, err)
	require.Equal(t, partition, decoded)
	key, err := ReductionPartitionNodeKey(partition, 2)
	require.NoError(t, err)
	require.Equal(t, NodeKey("reduce:a804905bf62ddc0981f852c994a80601a0a852674b47d48b0125dc471c262863"), key)
}

func TestReductionPartitionRejectsUnboundedOrNonCanonicalMembers(t *testing.T) {
	source := "sha256:" + strings.Repeat("b", 64)
	_, err := NewReductionPartition("merge", source, "count/v1", 0, 0, 1, []ManifestItem{
		{Key: "a", Value: manifestRef("count/v1", "a", "cas://a")},
		{Key: "b", Value: manifestRef("count/v1", "b", "cas://b")},
	})
	require.ErrorContains(t, err, "exceeds fan-in")
	_, err = NewReductionPartition("merge", source, "count/v1", 0, 0, 2, []ManifestItem{
		{Key: "b", Value: manifestRef("count/v1", "a", "cas://b")},
		{Key: "a", Value: manifestRef("count/v1", "b", "cas://a")},
	})
	require.ErrorContains(t, err, "strictly increasing")
	_, err = NewReductionPartition("merge", source, "count/v1", 0, 0, 2, []ManifestItem{
		{Key: "a", Value: manifestRef("wrong/v1", "a", "cas://a")},
	})
	require.ErrorContains(t, err, "does not match")
}

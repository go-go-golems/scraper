package workflowv3observations

import (
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func TestDynamicItemsAreNotCountedAsRetriesAndLeaseLossIsVisible(t *testing.T) {
	base := time.Unix(1_800_100_000, 0).UTC()
	plan := workflowv3.WorkflowPlan{Schema: workflowv3.PlanSchema, Name: "map", Maps: []workflowv3.PlanMap{{Key: "items"}}}
	plan.Digest, _ = workflowv3.Digest(plan)
	source := SourceSnapshot{
		Run: RunSource{RunID: "dynamic", Status: "succeeded", PlanDigest: plan.Digest, Plan: plan, CreatedAt: base, TerminalAt: base.Add(time.Second)},
		Nodes: []NodeSource{
			{NodeKey: "map:one", Origin: "map-item", RetryBackoffMillis: 10},
			{NodeKey: "map:two", Origin: "map-item", RetryBackoffMillis: 10},
			{NodeKey: "reduce:0:0", Origin: "reduction-partition"},
		},
		Attempts: []AttemptSource{
			{NodeKey: "map:one", Number: 1, Status: "lease_lost", StartedAt: base.Add(10 * time.Millisecond), FinishedAt: base.Add(20 * time.Millisecond), Failure: &FailureSource{Class: "execution", Code: "LEASE_EXPIRED", Retryable: true}},
			{NodeKey: "map:one", Number: 2, Status: "succeeded", StartedAt: base.Add(40 * time.Millisecond), FinishedAt: base.Add(50 * time.Millisecond)},
			{NodeKey: "map:two", Number: 1, Status: "succeeded", StartedAt: base.Add(10 * time.Millisecond), FinishedAt: base.Add(30 * time.Millisecond)},
			{NodeKey: "reduce:0:0", Number: 1, Status: "succeeded", StartedAt: base.Add(60 * time.Millisecond), FinishedAt: base.Add(70 * time.Millisecond)},
		},
	}
	observations, err := ProjectSnapshot(source, DefaultProjectOptions())
	require.NoError(t, err)
	require.Equal(t, int64(1), integerValue(t, metricByName(t, observations, "workflow.retries")))
	require.Equal(t, int64(1), integerValue(t, metricByName(t, observations, "workflow.lease_losses")))
	require.Equal(t, CountCoverage{Observed: 1, Total: 4}, observations.Coverage.QueueWaits)
	require.Equal(t, CountCoverage{Observed: 0, Total: 3}, observations.Coverage.CriticalPath)
}

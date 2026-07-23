package workflowv3observations

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/stretchr/testify/require"
)

func observationFixture() SourceSnapshot {
	at := func(millis int64) time.Time { return time.Unix(1_800_000_000, millis*int64(time.Millisecond)).UTC() }
	plan := workflowv3.WorkflowPlan{Schema: workflowv3.PlanSchema, Name: "neutral", Nodes: []workflowv3.PlanNode{{Key: "a", Retry: workflowv3.RetryPolicy{BackoffMillis: 100}}, {Key: "b", DependsOn: []workflowv3.NodeKey{"a"}}}}
	plan.Digest, _ = workflowv3.Digest(plan)
	failed := &FailureSource{Class: "transport", Code: "FIXTURE_RETRY", Retryable: true}
	return SourceSnapshot{
		Run: RunSource{RunID: "run-1", Status: "succeeded", PlanDigest: plan.Digest, Plan: plan, CreatedAt: at(0), TerminalAt: at(10_000), EventSequence: 42},
		Nodes: []NodeSource{
			{NodeKey: "b", Origin: "static", Dependencies: []workflowv3.NodeKey{"a"}},
			{NodeKey: "a", Origin: "static", RetryBackoffMillis: 100},
			{NodeKey: "map:item-1", Origin: "map-item"},
		},
		Attempts: []AttemptSource{
			{NodeKey: "b", Number: 1, Status: "succeeded", ResourceClass: "cpu.default", RegistryGeneration: "g1", StartedAt: at(2_500), FinishedAt: at(4_000)},
			{NodeKey: "a", Number: 2, Status: "succeeded", ResourceClass: "cpu.default", RegistryGeneration: "g1", StartedAt: at(1_200), FinishedAt: at(2_000)},
			{NodeKey: "map:item-1", Number: 1, Status: "succeeded", ResourceClass: "cpu.default", RegistryGeneration: "g1", StartedAt: at(200), FinishedAt: at(1_000)},
			{NodeKey: "a", Number: 1, Status: "failed", ResourceClass: "cpu.default", RegistryGeneration: "g1", StartedAt: at(100), FinishedAt: at(1_000), Failure: failed},
		},
		Operations: []workflowv3.ExternalOperation{
			{OperationID: "op-2", RunID: "run-1", NodeKey: "a", Attempt: 2, Ordinal: 1, AdmittedAt: at(1_200), Completion: &workflowv3.ExternalOperationCompletion{ProviderStartedAt: at(1_300), ElapsedMicros: 700_000, Outcome: workflowv3.ExternalOperationOutcomeSucceeded, AccountingMode: workflowv3.ExternalOperationAccountingActual, CompletedAt: at(2_100)}},
			{OperationID: "op-1", RunID: "run-1", NodeKey: "a", Attempt: 1, Ordinal: 1, AdmittedAt: at(100), Completion: &workflowv3.ExternalOperationCompletion{ProviderStartedAt: at(200), ElapsedMicros: 21_472_000, Outcome: workflowv3.ExternalOperationOutcomeFailed, Failure: &workflowv3.ExternalOperationFailure{Class: "transport", Code: "FIXTURE_RETRY"}, AccountingMode: workflowv3.ExternalOperationAccountingConservative, CompletedAt: at(1_000)}},
			{OperationID: "op-3", RunID: "run-1", NodeKey: "b", Attempt: 1, Ordinal: 1, AdmittedAt: at(3_000)},
		},
		Artifacts: []ArtifactSource{{Name: "result", Schema: "fixture/v1", Digest: "sha256:" + strings.Repeat("b", 64), MediaType: "application/json", SizeBytes: 17}},
	}
}

func metricByName(t *testing.T, observations ObservationSet, name string) Metric {
	t.Helper()
	for _, metric := range observations.Metrics {
		if metric.Name == name {
			return metric
		}
	}
	t.Fatalf("metric %s not found", name)
	return Metric{}
}

func integerValue(t *testing.T, metric Metric) int64 {
	t.Helper()
	var value int64
	require.NoError(t, json.Unmarshal(metric.Value, &value))
	return value
}

func ratioValue(t *testing.T, metric Metric) Ratio {
	t.Helper()
	var value Ratio
	require.NoError(t, json.Unmarshal(metric.Value, &value))
	return value
}

func TestProjectRetryInclusiveDeterministicAndPrivacySafe(t *testing.T) {
	source := observationFixture()
	first, err := ProjectSnapshot(source, DefaultProjectOptions())
	require.NoError(t, err)
	require.Equal(t, "sha256:5ab0ca105f9161765c51b63f3f7abf94d7f2e8cf9999e08d2091b33faea5b0d2", first.SourceDigest)
	require.Equal(t, "sha256:00dc6910f089c85d022145398659694abad054096b532e998450d8c6e0c83cc9", first.Digest)
	require.Equal(t, int64(4), integerValue(t, metricByName(t, first, "workflow.job_attempts")))
	require.Equal(t, int64(1), integerValue(t, metricByName(t, first, "workflow.retries")))
	require.Equal(t, int64(1), integerValue(t, metricByName(t, first, "workflow.failed_job_attempts")))
	require.Equal(t, int64(3), integerValue(t, metricByName(t, first, "workflow.external_operations.admitted")))
	require.Equal(t, int64(1), integerValue(t, metricByName(t, first, "workflow.external_operations.failed")))
	require.Equal(t, int64(22_172_000), integerValue(t, metricByName(t, first, "workflow.external_operations.elapsed_sum")))
	require.Equal(t, int64(21_472_000), integerValue(t, metricByName(t, first, "workflow.external_operations.elapsed_union")))
	require.Equal(t, int64(700_000), integerValue(t, metricByName(t, first, "workflow.queue_wait")))
	require.Equal(t, Ratio{Numerator: 2, Denominator: 3}, ratioValue(t, metricByName(t, first, "workflow.external_operations.completion_coverage")))
	require.Equal(t, CountCoverage{Observed: 3, Total: 4}, first.Coverage.QueueWaits)
	require.Equal(t, CountCoverage{Observed: 2, Total: 3}, first.Coverage.CriticalPath)
	var critical struct {
		Entries []criticalPathEntry `json:"entries"`
	}
	for _, trace := range first.Traces {
		if trace.Kind == "workflow.critical_path" {
			require.NoError(t, json.Unmarshal(trace.Value, &critical))
		}
	}
	require.Len(t, critical.Entries, 2)
	require.Equal(t, workflowv3.NodeKey("b"), critical.Entries[1].NodeKey)
	require.Equal(t, int64(3_200_000), critical.Entries[1].CumulativeMicros)
	require.Equal(t, "result", first.ArtifactLineage[0].Name)
	body, err := workflowv3.CanonicalJSON(first)
	require.NoError(t, err)
	require.NotContains(t, string(body), "locator")
	decoded, err := Decode(body)
	require.NoError(t, err)
	require.Equal(t, first.Digest, decoded.Digest)

	shuffled := observationFixture()
	slices.Reverse(shuffled.Nodes)
	slices.Reverse(shuffled.Attempts)
	slices.Reverse(shuffled.Operations)
	second, err := ProjectSnapshot(shuffled, DefaultProjectOptions())
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestIntervalsUseHalfOpenUnionAndPeak(t *testing.T) {
	base := time.Unix(1_800_000_000, 0).UTC()
	intervals := []interval{{base, base.Add(10 * time.Microsecond)}, {base.Add(5 * time.Microsecond), base.Add(8 * time.Microsecond)}, {base.Add(10 * time.Microsecond), base.Add(14 * time.Microsecond)}}
	sum, union, peak := intervalMicros(intervals)
	require.Equal(t, int64(17), sum)
	require.Equal(t, int64(14), union)
	require.Equal(t, 2, peak)
	slices.Reverse(intervals)
	require.Equal(t, []any{int64(17), int64(14), 2}, func() []any { a, b, c := intervalMicros(intervals); return []any{a, b, c} }())
}

func TestProjectZeroOperationsAndTruncatedCriticalPath(t *testing.T) {
	source := observationFixture()
	source.Operations = nil
	observations, err := ProjectSnapshot(source, ProjectOptions{MaxCriticalPathEntries: 1})
	require.NoError(t, err)
	require.Equal(t, Ratio{Numerator: 0, Denominator: 0}, ratioValue(t, metricByName(t, observations, "workflow.external_operations.completion_coverage")))
	for _, trace := range observations.Traces {
		if trace.Kind == "workflow.critical_path" {
			require.True(t, trace.Truncated)
		}
	}
}

func TestObservationContractRejectsUnknownFieldsTamperingAndNonterminalSources(t *testing.T) {
	observations, err := ProjectSnapshot(observationFixture(), DefaultProjectOptions())
	require.NoError(t, err)
	body, err := workflowv3.CanonicalJSON(observations)
	require.NoError(t, err)
	body = append(body[:len(body)-1], []byte(`,"unknown":true}`)...)
	_, err = Decode(body)
	require.ErrorContains(t, err, "unknown field")
	observations.Metrics[0].Boundary = "changed"
	require.ErrorContains(t, Validate(observations), "digest mismatch")
	source := observationFixture()
	source.Run.Status = "running"
	_, err = ProjectSnapshot(source, DefaultProjectOptions())
	require.ErrorContains(t, err, "terminal run")
}

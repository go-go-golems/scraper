package workflowv3observations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const (
	boundaryRunElapsed       = "run-admission-to-terminal-record/v1"
	boundaryAttempts         = "all-durable-attempts/v1"
	boundaryOperations       = "all-admitted-external-operations/v1"
	boundaryClosedOperations = "all-admitted-closed-external-operations/v1"
	boundaryQueueWait        = "known-eligibility-to-attempt-start/v1"
	boundaryArtifacts        = "terminal-named-output-references/v1"
)

func Project(ctx context.Context, source Source, runID workflowv3.RunID, options ProjectOptions) (ObservationSet, error) {
	if source == nil {
		return ObservationSet{}, fmt.Errorf("workflow observation source is required")
	}
	snapshot, err := source.ObservationSnapshot(ctx, runID)
	if err != nil {
		return ObservationSet{}, err
	}
	return ProjectSnapshot(snapshot, options)
}

func ProjectSnapshot(source SourceSnapshot, options ProjectOptions) (ObservationSet, error) {
	if options.MaxCriticalPathEntries <= 0 || options.MaxCriticalPathEntries > 10_000 {
		return ObservationSet{}, fmt.Errorf("critical-path entry limit must be between 1 and 10000")
	}
	if err := validateSource(source); err != nil {
		return ObservationSet{}, err
	}
	source = cloneSource(source)
	canonicalizeSource(&source)
	sourceBody, err := workflowv3.CanonicalJSON(source)
	if err != nil {
		return ObservationSet{}, err
	}
	sourceHash := sha256.Sum256(sourceBody)
	sourceDigest := "sha256:" + hex.EncodeToString(sourceHash[:])
	result := ObservationSet{
		SchemaVersion: SchemaVersion, DerivationVersion: DerivationVersion,
		PrivacyClass: PrivacyClass, RunID: source.Run.RunID, RunStatus: source.Run.Status,
		PlanDigest: source.Run.PlanDigest, EventSequence: source.Run.EventSequence,
		SourceDigest: sourceDigest,
	}
	result.ArtifactLineage = artifactLineage(source.Artifacts)
	result.Metrics, result.Coverage = deriveMetrics(source)
	critical, criticalCoverage := criticalPath(source, options.MaxCriticalPathEntries)
	result.Coverage.CriticalPath = criticalCoverage
	result.Traces = []Trace{
		failureTrace(source.Attempts),
		critical,
		artifactTrace(result.ArtifactLineage),
	}
	sort.Slice(result.Traces, func(i, j int) bool { return result.Traces[i].Kind < result.Traces[j].Kind })
	withoutDigest := result
	withoutDigest.Digest = ""
	digest, err := workflowv3.Digest(withoutDigest)
	if err != nil {
		return ObservationSet{}, err
	}
	result.Digest = digest
	if err := Validate(result); err != nil {
		return ObservationSet{}, err
	}
	return result, nil
}

func validateSource(source SourceSnapshot) error {
	if source.Run.RunID == "" || source.Run.PlanDigest == "" || source.Run.Plan.Digest != source.Run.PlanDigest {
		return fmt.Errorf("observation source run identity is invalid")
	}
	plan := source.Run.Plan
	plan.Digest = ""
	planDigest, err := workflowv3.Digest(plan)
	if err != nil || planDigest != source.Run.PlanDigest {
		return fmt.Errorf("observation source plan digest is invalid")
	}
	if source.Run.Status != "succeeded" && source.Run.Status != "failed" && source.Run.Status != "canceled" {
		return fmt.Errorf("workflow observations require a terminal run")
	}
	if source.Run.CreatedAt.IsZero() || source.Run.TerminalAt.Before(source.Run.CreatedAt) || source.Run.CreatedAt.Location() != time.UTC || source.Run.TerminalAt.Location() != time.UTC {
		return fmt.Errorf("observation source run timing is invalid")
	}
	if len(source.Nodes) > 100_000 || len(source.Attempts) > 100_000 || len(source.Operations) > 100_000 || len(source.Artifacts) > 10_000 {
		return fmt.Errorf("observation source exceeds bounded record limits")
	}
	seenNodes := map[workflowv3.NodeKey]bool{}
	staticNodes := map[workflowv3.NodeKey]bool{}
	for _, node := range source.Nodes {
		if node.NodeKey == "" || seenNodes[node.NodeKey] {
			return fmt.Errorf("observation source node identity is invalid")
		}
		seenNodes[node.NodeKey] = true
		switch node.Origin {
		case "static":
			staticNodes[node.NodeKey] = true
		case "map-item", "reduction-partition":
		default:
			return fmt.Errorf("observation source node origin is invalid")
		}
	}
	if len(staticNodes) != len(source.Run.Plan.Nodes) {
		return fmt.Errorf("observation source static nodes do not match plan")
	}
	nodeSources := map[workflowv3.NodeKey]NodeSource{}
	for _, node := range source.Nodes {
		nodeSources[node.NodeKey] = node
		for _, dependency := range node.Dependencies {
			if !seenNodes[dependency] {
				return fmt.Errorf("observation source node dependency is invalid")
			}
		}
	}
	for _, node := range source.Run.Plan.Nodes {
		current, ok := nodeSources[node.Key]
		if !ok || current.Origin != "static" || current.RetryBackoffMillis != node.Retry.BackoffMillis || !nodeKeysEqual(current.Dependencies, node.DependsOn) {
			return fmt.Errorf("observation source static nodes do not match plan")
		}
	}
	seenAttempts := map[string]bool{}
	for _, attempt := range source.Attempts {
		key := fmt.Sprintf("%s\x00%d", attempt.NodeKey, attempt.Number)
		if !seenNodes[attempt.NodeKey] || attempt.Number < 1 || attempt.StartedAt.IsZero() || attempt.StartedAt.Location() != time.UTC || seenAttempts[key] {
			return fmt.Errorf("observation source attempt identity or timing is invalid")
		}
		seenAttempts[key] = true
		switch attempt.Status {
		case "succeeded", "failed", "lease_lost", "canceled":
		default:
			return fmt.Errorf("observation source attempt status is invalid")
		}
		if attempt.FinishedAt.IsZero() || attempt.FinishedAt.Location() != time.UTC || attempt.FinishedAt.Before(attempt.StartedAt) {
			return fmt.Errorf("terminal observation source contains incomplete attempt")
		}
	}
	seenOperations := map[string]bool{}
	for _, operation := range source.Operations {
		attemptKey := fmt.Sprintf("%s\x00%d", operation.NodeKey, operation.Attempt)
		if operation.RunID != source.Run.RunID || operation.OperationID == "" || seenOperations[operation.OperationID] || !seenAttempts[attemptKey] || operation.AdmittedAt.IsZero() || operation.AdmittedAt.Location() != time.UTC {
			return fmt.Errorf("observation source operation identity is invalid")
		}
		seenOperations[operation.OperationID] = true
		if operation.Completion != nil && operation.Completion.ElapsedMicros < 0 {
			return fmt.Errorf("observation source operation timing is invalid")
		}
		if operation.Completion != nil && (operation.Completion.ProviderStartedAt.IsZero() || operation.Completion.CompletedAt.IsZero() || operation.Completion.ProviderStartedAt.Location() != time.UTC || operation.Completion.CompletedAt.Location() != time.UTC) {
			return fmt.Errorf("observation source operation timing is invalid")
		}
	}
	seenArtifacts := map[string]bool{}
	for _, artifact := range source.Artifacts {
		if artifact.Name == "" || seenArtifacts[artifact.Name] || artifact.Digest == "" || artifact.Schema == "" || artifact.SizeBytes < 0 {
			return fmt.Errorf("observation source artifact identity is invalid")
		}
		seenArtifacts[artifact.Name] = true
	}
	return nil
}

func nodeKeysEqual(left, right []workflowv3.NodeKey) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy, rightCopy := append([]workflowv3.NodeKey(nil), left...), append([]workflowv3.NodeKey(nil), right...)
	sort.Slice(leftCopy, func(i, j int) bool { return leftCopy[i] < leftCopy[j] })
	sort.Slice(rightCopy, func(i, j int) bool { return rightCopy[i] < rightCopy[j] })
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}

func cloneSource(source SourceSnapshot) SourceSnapshot {
	ret := source
	ret.Nodes = append([]NodeSource(nil), source.Nodes...)
	for index := range ret.Nodes {
		ret.Nodes[index].Dependencies = append([]workflowv3.NodeKey(nil), ret.Nodes[index].Dependencies...)
	}
	ret.Attempts = append([]AttemptSource(nil), source.Attempts...)
	ret.Operations = append([]workflowv3.ExternalOperation(nil), source.Operations...)
	ret.Artifacts = append([]ArtifactSource(nil), source.Artifacts...)
	return ret
}

func canonicalizeSource(source *SourceSnapshot) {
	sort.Slice(source.Nodes, func(i, j int) bool { return source.Nodes[i].NodeKey < source.Nodes[j].NodeKey })
	for index := range source.Nodes {
		sort.Slice(source.Nodes[index].Dependencies, func(i, j int) bool { return source.Nodes[index].Dependencies[i] < source.Nodes[index].Dependencies[j] })
	}
	sort.Slice(source.Attempts, func(i, j int) bool {
		if source.Attempts[i].NodeKey == source.Attempts[j].NodeKey {
			return source.Attempts[i].Number < source.Attempts[j].Number
		}
		return source.Attempts[i].NodeKey < source.Attempts[j].NodeKey
	})
	sort.Slice(source.Operations, func(i, j int) bool { return source.Operations[i].OperationID < source.Operations[j].OperationID })
	sort.Slice(source.Artifacts, func(i, j int) bool { return source.Artifacts[i].Name < source.Artifacts[j].Name })
}

func deriveMetrics(source SourceSnapshot) ([]Metric, Coverage) {
	elapsed := source.Run.TerminalAt.Sub(source.Run.CreatedAt).Microseconds()
	attemptIntervals := make([]interval, 0, len(source.Attempts))
	failed, canceled, leaseLost := 0, 0, 0
	succeededNodes := map[workflowv3.NodeKey]bool{}
	attemptsByNode := map[workflowv3.NodeKey][]AttemptSource{}
	for _, attempt := range source.Attempts {
		attemptIntervals = append(attemptIntervals, interval{start: attempt.StartedAt, end: attempt.FinishedAt})
		attemptsByNode[attempt.NodeKey] = append(attemptsByNode[attempt.NodeKey], attempt)
		switch attempt.Status {
		case "succeeded":
			succeededNodes[attempt.NodeKey] = true
		case "failed":
			failed++
		case "canceled":
			canceled++
		case "lease_lost":
			failed++
			leaseLost++
		}
	}
	retries := 0
	for _, attempts := range attemptsByNode {
		if len(attempts) > 1 {
			retries += len(attempts) - 1
		}
	}
	_, _, attemptPeak := intervalMicros(attemptIntervals)

	outcomes := map[string]int{}
	operationIntervals := make([]interval, 0, len(source.Operations))
	accounted := 0
	for _, operation := range source.Operations {
		if operation.Completion == nil {
			continue
		}
		completion := operation.Completion
		outcomes[completion.Outcome]++
		operationIntervals = append(operationIntervals, interval{start: completion.ProviderStartedAt, end: completion.ProviderStartedAt.Add(time.Duration(completion.ElapsedMicros) * time.Microsecond)})
		if completion.AccountingMode == workflowv3.ExternalOperationAccountingActual || completion.AccountingMode == workflowv3.ExternalOperationAccountingConservative {
			accounted++
		}
	}
	opSum, opUnion, opPeak := intervalMicros(operationIntervals)
	_, operationRunUnion, _ := intervalMicros(intersectIntervals(operationIntervals, interval{start: source.Run.CreatedAt, end: source.Run.TerminalAt}))
	queueSum, queueObserved := queueWait(source, attemptsByNode)
	closedAttempts := len(source.Attempts)
	closedOperations := len(operationIntervals)
	coverage := Coverage{
		Attempts:       CountCoverage{Observed: closedAttempts, Total: len(source.Attempts)},
		QueueWaits:     CountCoverage{Observed: queueObserved, Total: len(source.Attempts)},
		Operations:     CountCoverage{Observed: closedOperations, Total: len(source.Operations)},
		Accounting:     CountCoverage{Observed: accounted, Total: len(source.Operations)},
		TerminalSource: true,
	}
	metrics := []Metric{
		integerMetric("workflow.attempt_peak_active", int64(attemptPeak), "count", boundaryAttempts),
		ratioMetric("workflow.accounting.coverage", int64(accounted), int64(len(source.Operations)), "ratio", boundaryOperations),
		integerMetric("workflow.canceled_job_attempts", int64(canceled), "count", boundaryAttempts),
		integerMetric("workflow.elapsed", elapsed, "microseconds", boundaryRunElapsed),
		ratioMetric("workflow.external_operations.completion_coverage", int64(closedOperations), int64(len(source.Operations)), "ratio", boundaryOperations),
		ratioMetric("workflow.external_operations.coverage", operationRunUnion, elapsed, "ratio", boundaryRunElapsed),
		integerMetric("workflow.external_operations.admitted", int64(len(source.Operations)), "count", boundaryOperations),
		integerMetric("workflow.external_operations.elapsed_sum", opSum, "microseconds", boundaryClosedOperations),
		integerMetric("workflow.external_operations.elapsed_union", opUnion, "microseconds", boundaryClosedOperations),
		integerMetric("workflow.external_operations.failed", int64(outcomes[workflowv3.ExternalOperationOutcomeFailed]), "count", boundaryOperations),
		integerMetric("workflow.external_operations.canceled", int64(outcomes[workflowv3.ExternalOperationOutcomeCanceled]), "count", boundaryOperations),
		integerMetric("workflow.external_operations.succeeded", int64(outcomes[workflowv3.ExternalOperationOutcomeSucceeded]), "count", boundaryOperations),
		integerMetric("workflow.external_operations.timed_out", int64(outcomes[workflowv3.ExternalOperationOutcomeTimedOut]), "count", boundaryOperations),
		integerMetric("workflow.external_operations.unknown", int64(outcomes[workflowv3.ExternalOperationOutcomeUnknown]), "count", boundaryOperations),
		integerMetric("workflow.failed_job_attempts", int64(failed), "count", boundaryAttempts),
		integerMetric("workflow.job_attempts", int64(len(source.Attempts)), "count", boundaryAttempts),
		integerMetric("workflow.lease_losses", int64(leaseLost), "count", boundaryAttempts),
		ratioMetric("workflow.node_throughput", int64(len(succeededNodes))*1_000_000, elapsed, "nodes/second", boundaryRunElapsed),
		integerMetric("workflow.operation_peak_active", int64(opPeak), "count", boundaryClosedOperations),
		integerMetric("workflow.queue_wait", queueSum, "microseconds", boundaryQueueWait),
		integerMetric("workflow.retries", int64(retries), "count", boundaryAttempts),
		stringMetric("workflow.terminal_status", source.Run.Status, "status", boundaryRunElapsed),
	}
	sort.Slice(metrics, func(i, j int) bool { return metrics[i].Name < metrics[j].Name })
	return metrics, coverage
}

func queueWait(source SourceSnapshot, attemptsByNode map[workflowv3.NodeKey][]AttemptSource) (int64, int) {
	nodes := make(map[workflowv3.NodeKey]NodeSource, len(source.Nodes))
	for _, node := range source.Nodes {
		nodes[node.NodeKey] = node
	}
	successEnd := map[workflowv3.NodeKey]time.Time{}
	for key, attempts := range attemptsByNode {
		for _, attempt := range attempts {
			if attempt.Status == "succeeded" && attempt.FinishedAt.After(successEnd[key]) {
				successEnd[key] = attempt.FinishedAt
			}
		}
	}
	var total int64
	observed := 0
	for key, attempts := range attemptsByNode {
		sort.Slice(attempts, func(i, j int) bool { return attempts[i].Number < attempts[j].Number })
		node, known := nodes[key]
		for index, attempt := range attempts {
			var eligible time.Time
			if index > 0 {
				eligible = attempts[index-1].FinishedAt.Add(time.Duration(node.RetryBackoffMillis) * time.Millisecond)
			} else if known && node.Origin == "static" && !node.HasGate && !node.HasBudget {
				eligible = source.Run.CreatedAt
				for _, dependency := range node.Dependencies {
					finished, ok := successEnd[dependency]
					if !ok {
						eligible = time.Time{}
						break
					}
					if finished.After(eligible) {
						eligible = finished
					}
				}
			}
			if eligible.IsZero() {
				continue
			}
			wait := attempt.StartedAt.Sub(eligible).Microseconds()
			if wait < 0 {
				wait = 0
			}
			total += wait
			observed++
		}
	}
	return total, observed
}

func integerMetric(name string, value int64, unit, boundary string) Metric {
	return Metric{Name: name, Scope: "workflow", ValueKind: "integer", Value: mustJSON(value), Unit: unit, Boundary: boundary, Metadata: json.RawMessage(`{}`)}
}
func ratioMetric(name string, numerator, denominator int64, unit, boundary string) Metric {
	return Metric{Name: name, Scope: "workflow", ValueKind: "ratio", Value: mustJSON(Ratio{Numerator: numerator, Denominator: denominator}), Unit: unit, Boundary: boundary, Metadata: json.RawMessage(`{}`)}
}
func stringMetric(name, value, unit, boundary string) Metric {
	return Metric{Name: name, Scope: "workflow", ValueKind: "string", Value: mustJSON(value), Unit: unit, Boundary: boundary, Metadata: json.RawMessage(`{}`)}
}

func artifactLineage(source []ArtifactSource) []ArtifactLineage {
	ret := make([]ArtifactLineage, len(source))
	for index, artifact := range source {
		ret[index] = ArtifactLineage(artifact)
	}
	return ret
}

func failureTrace(attempts []AttemptSource) Trace {
	type item struct {
		NodeKey workflowv3.NodeKey `json:"nodeKey"`
		Attempt int                `json:"attempt"`
		Status  string             `json:"status"`
		Failure *FailureSource     `json:"failure,omitempty"`
	}
	values := make([]item, 0)
	for _, attempt := range attempts {
		if attempt.Status != "succeeded" {
			values = append(values, item{NodeKey: attempt.NodeKey, Attempt: attempt.Number, Status: attempt.Status, Failure: attempt.Failure})
		}
	}
	return Trace{Kind: "workflow.failures", SchemaVersion: "scraper-workflow-failure-trace/v1", Value: mustJSON(map[string]any{"attempts": values})}
}

func artifactTrace(artifacts []ArtifactLineage) Trace {
	return Trace{Kind: "workflow.artifact_lineage", SchemaVersion: "scraper-workflow-artifact-lineage/v1", Value: mustJSON(map[string]any{"boundary": boundaryArtifacts, "outputs": artifacts})}
}

func mustJSON(value any) json.RawMessage {
	body, err := workflowv3.CanonicalJSON(value)
	if err != nil {
		panic(err)
	}
	return body
}

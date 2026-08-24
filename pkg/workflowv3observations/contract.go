package workflowv3observations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

func Decode(body []byte) (ObservationSet, error) {
	var observations ObservationSet
	if err := workflowv3.StrictDecode(body, &observations); err != nil {
		return ObservationSet{}, fmt.Errorf("decode workflow observations: %w", err)
	}
	if err := Validate(observations); err != nil {
		return ObservationSet{}, err
	}
	return observations, nil
}

type metricContract struct{ valueKind, unit, boundary string }

var canonicalMetricNames = map[string]metricContract{
	"workflow.accounting.coverage":                     {"ratio", "ratio", boundaryOperations},
	"workflow.attempt_peak_active":                     {"integer", "count", boundaryAttempts},
	"workflow.canceled_job_attempts":                   {"integer", "count", boundaryAttempts},
	"workflow.elapsed":                                 {"integer", "microseconds", boundaryRunElapsed},
	"workflow.external_operations.admitted":            {"integer", "count", boundaryOperations},
	"workflow.external_operations.canceled":            {"integer", "count", boundaryOperations},
	"workflow.external_operations.completion_coverage": {"ratio", "ratio", boundaryOperations},
	"workflow.external_operations.coverage":            {"ratio", "ratio", boundaryRunElapsed},
	"workflow.external_operations.elapsed_sum":         {"integer", "microseconds", boundaryClosedOperations},
	"workflow.external_operations.elapsed_union":       {"integer", "microseconds", boundaryClosedOperations},
	"workflow.external_operations.failed":              {"integer", "count", boundaryOperations},
	"workflow.external_operations.succeeded":           {"integer", "count", boundaryOperations},
	"workflow.external_operations.timed_out":           {"integer", "count", boundaryOperations},
	"workflow.external_operations.unknown":             {"integer", "count", boundaryOperations},
	"workflow.failed_job_attempts":                     {"integer", "count", boundaryAttempts},
	"workflow.job_attempts":                            {"integer", "count", boundaryAttempts},
	"workflow.lease_losses":                            {"integer", "count", boundaryAttempts},
	"workflow.node_throughput":                         {"ratio", "nodes/second", boundaryRunElapsed},
	"workflow.operation_peak_active":                   {"integer", "count", boundaryClosedOperations},
	"workflow.queue_wait":                              {"integer", "microseconds", boundaryQueueWait},
	"workflow.retries":                                 {"integer", "count", boundaryAttempts},
	"workflow.terminal_status":                         {"string", "status", boundaryRunElapsed},
}

var canonicalTraceKinds = map[string]string{
	"workflow.artifact_lineage": "scraper-workflow-artifact-lineage/v1",
	"workflow.critical_path":    "scraper-workflow-critical-path/v1",
	"workflow.failures":         "scraper-workflow-failure-trace/v1",
}

func Validate(observations ObservationSet) error {
	if observations.SchemaVersion != SchemaVersion || observations.DerivationVersion != DerivationVersion || observations.PrivacyClass != PrivacyClass {
		return fmt.Errorf("workflow observation contract identity is invalid")
	}
	if observations.RunID == "" || !observationDigest(observations.PlanDigest) || !observationDigest(observations.SourceDigest) || !observationDigest(observations.Digest) || observations.EventSequence < 0 {
		return fmt.Errorf("workflow observation identities are required")
	}
	if observations.RunStatus != "succeeded" && observations.RunStatus != "failed" && observations.RunStatus != "canceled" {
		return fmt.Errorf("workflow observation terminal status is invalid")
	}
	if !observations.Coverage.TerminalSource {
		return fmt.Errorf("workflow observation source is not terminal")
	}
	for _, coverage := range []CountCoverage{observations.Coverage.Attempts, observations.Coverage.QueueWaits, observations.Coverage.Operations, observations.Coverage.Accounting, observations.Coverage.CriticalPath} {
		if coverage.Observed < 0 || coverage.Total < 0 || coverage.Observed > coverage.Total {
			return fmt.Errorf("workflow observation coverage is invalid")
		}
	}
	if len(observations.Metrics) != len(canonicalMetricNames) || len(observations.Traces) != len(canonicalTraceKinds) {
		return fmt.Errorf("workflow observation canonical metric or trace set is incomplete")
	}
	previous := ""
	for _, metric := range observations.Metrics {
		contract, known := canonicalMetricNames[metric.Name]
		if metric.Name <= previous || !known || metric.Scope != "workflow" || metric.ValueKind != contract.valueKind || metric.Unit != contract.unit || metric.Boundary != contract.boundary || len(metric.Value) == 0 {
			return fmt.Errorf("workflow observation metrics must be strictly sorted and canonical")
		}
		switch metric.ValueKind {
		case "integer":
			var value int64
			if err := workflowv3.StrictDecode(metric.Value, &value); err != nil || value < 0 {
				return fmt.Errorf("workflow observation metric %q integer value is invalid", metric.Name)
			}
		case "ratio":
			var value Ratio
			if err := workflowv3.StrictDecode(metric.Value, &value); err != nil || value.Numerator < 0 || value.Denominator < 0 || (value.Denominator == 0 && value.Numerator != 0) {
				return fmt.Errorf("workflow observation metric %q ratio value is invalid", metric.Name)
			}
		case "string":
			var value string
			if err := workflowv3.StrictDecode(metric.Value, &value); err != nil || metric.Name != "workflow.terminal_status" || value != observations.RunStatus {
				return fmt.Errorf("workflow observation metric %q string value is invalid", metric.Name)
			}
		default:
			return fmt.Errorf("workflow observation metric %q value kind is invalid", metric.Name)
		}
		var metadata map[string]json.RawMessage
		if err := workflowv3.StrictDecode(metric.Metadata, &metadata); err != nil || len(metadata) != 0 {
			return fmt.Errorf("workflow observation metric %q metadata is invalid", metric.Name)
		}
		previous = metric.Name
	}
	previous = ""
	for _, trace := range observations.Traces {
		schema, known := canonicalTraceKinds[trace.Kind]
		if trace.Kind <= previous || !known || trace.SchemaVersion != schema || len(trace.Value) == 0 {
			return fmt.Errorf("workflow observation traces must be strictly sorted and canonical")
		}
		previous = trace.Kind
	}
	for index, artifact := range observations.ArtifactLineage {
		if strings.TrimSpace(artifact.Name) == "" || !observationDigest(artifact.Digest) || artifact.SizeBytes < 0 || (index > 0 && artifact.Name <= observations.ArtifactLineage[index-1].Name) {
			return fmt.Errorf("workflow observation artifact lineage is invalid")
		}
	}
	withoutDigest := observations
	withoutDigest.Digest = ""
	digest, err := workflowv3.Digest(withoutDigest)
	if err != nil {
		return err
	}
	if digest != observations.Digest {
		return fmt.Errorf("workflow observation digest mismatch")
	}
	return nil
}

func observationDigest(value string) bool {
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, current := range value[len("sha256:"):] {
		if (current < '0' || current > '9') && (current < 'a' || current > 'f') {
			return false
		}
	}
	return true
}

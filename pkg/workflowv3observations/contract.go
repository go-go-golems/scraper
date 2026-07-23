package workflowv3observations

import (
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

var canonicalMetricNames = map[string]bool{
	"workflow.accounting.coverage":                     true,
	"workflow.attempt_peak_active":                     true,
	"workflow.canceled_job_attempts":                   true,
	"workflow.elapsed":                                 true,
	"workflow.external_operations.admitted":            true,
	"workflow.external_operations.canceled":            true,
	"workflow.external_operations.completion_coverage": true,
	"workflow.external_operations.coverage":            true,
	"workflow.external_operations.elapsed_sum":         true,
	"workflow.external_operations.elapsed_union":       true,
	"workflow.external_operations.failed":              true,
	"workflow.external_operations.succeeded":           true,
	"workflow.external_operations.timed_out":           true,
	"workflow.external_operations.unknown":             true,
	"workflow.failed_job_attempts":                     true,
	"workflow.job_attempts":                            true,
	"workflow.lease_losses":                            true,
	"workflow.node_throughput":                         true,
	"workflow.operation_peak_active":                   true,
	"workflow.queue_wait":                              true,
	"workflow.retries":                                 true,
	"workflow.terminal_status":                         true,
}

var canonicalTraceKinds = map[string]bool{
	"workflow.artifact_lineage": true,
	"workflow.critical_path":    true,
	"workflow.failures":         true,
}

func Validate(observations ObservationSet) error {
	if observations.SchemaVersion != SchemaVersion || observations.DerivationVersion != DerivationVersion || observations.PrivacyClass != PrivacyClass {
		return fmt.Errorf("workflow observation contract identity is invalid")
	}
	if observations.RunID == "" || observations.PlanDigest == "" || observations.SourceDigest == "" || observations.Digest == "" {
		return fmt.Errorf("workflow observation identities are required")
	}
	if len(observations.Metrics) != len(canonicalMetricNames) || len(observations.Traces) != len(canonicalTraceKinds) {
		return fmt.Errorf("workflow observation canonical metric or trace set is incomplete")
	}
	previous := ""
	for _, metric := range observations.Metrics {
		if metric.Name <= previous || !canonicalMetricNames[metric.Name] || metric.Scope != "workflow" || metric.Boundary == "" || len(metric.Value) == 0 {
			return fmt.Errorf("workflow observation metrics must be strictly sorted and complete")
		}
		switch metric.ValueKind {
		case "integer", "ratio", "string":
		default:
			return fmt.Errorf("workflow observation metric %q value kind is invalid", metric.Name)
		}
		previous = metric.Name
	}
	previous = ""
	for _, trace := range observations.Traces {
		if trace.Kind <= previous || !canonicalTraceKinds[trace.Kind] || trace.SchemaVersion == "" || len(trace.Value) == 0 {
			return fmt.Errorf("workflow observation traces must be strictly sorted and complete")
		}
		previous = trace.Kind
	}
	for index, artifact := range observations.ArtifactLineage {
		if strings.TrimSpace(artifact.Name) == "" || artifact.Digest == "" || artifact.SizeBytes < 0 || (index > 0 && artifact.Name <= observations.ArtifactLineage[index-1].Name) {
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

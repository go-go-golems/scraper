// Package workflowv3observations derives deterministic, privacy-safe evidence
// from authoritative Workflow V3 records. It never persists aggregate state.
package workflowv3observations

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const (
	SchemaVersion     = "scraper-workflow-observations/v1"
	DerivationVersion = "workflow-observations/v1"
	PrivacyClass      = "bounded-identifiers-digests-integers"
)

type RunSource struct {
	RunID         workflowv3.RunID        `json:"runId"`
	Status        string                  `json:"status"`
	PlanDigest    string                  `json:"planDigest"`
	Plan          workflowv3.WorkflowPlan `json:"plan"`
	CreatedAt     time.Time               `json:"createdAt"`
	TerminalAt    time.Time               `json:"terminalAt"`
	EventSequence int64                   `json:"eventSequence"`
}

type AttemptSource struct {
	NodeKey            workflowv3.NodeKey `json:"nodeKey"`
	Number             int                `json:"number"`
	Status             string             `json:"status"`
	ResourceClass      string             `json:"resourceClass"`
	RegistryGeneration string             `json:"registryGeneration"`
	StartedAt          time.Time          `json:"startedAt"`
	FinishedAt         time.Time          `json:"finishedAt,omitempty"`
	Failure            *FailureSource     `json:"failure,omitempty"`
}

type FailureSource struct {
	Class     string `json:"class"`
	Code      string `json:"code"`
	Retryable bool   `json:"retryable"`
}

type NodeSource struct {
	NodeKey            workflowv3.NodeKey   `json:"nodeKey"`
	Origin             string               `json:"origin"`
	Dependencies       []workflowv3.NodeKey `json:"dependencies,omitempty"`
	RetryBackoffMillis int64                `json:"retryBackoffMillis"`
	HasGate            bool                 `json:"hasGate"`
	HasBudget          bool                 `json:"hasBudget"`
}

type ArtifactSource struct {
	Name      string `json:"name"`
	Schema    string `json:"schema"`
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	SizeBytes int64  `json:"sizeBytes"`
}

type SourceSnapshot struct {
	Run        RunSource                      `json:"run"`
	Nodes      []NodeSource                   `json:"nodes"`
	Attempts   []AttemptSource                `json:"attempts"`
	Operations []workflowv3.ExternalOperation `json:"operations"`
	Artifacts  []ArtifactSource               `json:"artifacts"`
}

type Source interface {
	ObservationSnapshot(context.Context, workflowv3.RunID) (SourceSnapshot, error)
}

type ProjectOptions struct {
	MaxCriticalPathEntries int
}

func DefaultProjectOptions() ProjectOptions { return ProjectOptions{MaxCriticalPathEntries: 1024} }

type ObservationSet struct {
	SchemaVersion     string            `json:"schemaVersion"`
	DerivationVersion string            `json:"derivationVersion"`
	PrivacyClass      string            `json:"privacyClass"`
	RunID             workflowv3.RunID  `json:"runId"`
	RunStatus         string            `json:"runStatus"`
	PlanDigest        string            `json:"planDigest"`
	EventSequence     int64             `json:"eventSequence"`
	SourceDigest      string            `json:"sourceDigest"`
	Metrics           []Metric          `json:"metrics"`
	Traces            []Trace           `json:"traces"`
	Coverage          Coverage          `json:"coverage"`
	ArtifactLineage   []ArtifactLineage `json:"artifactLineage"`
	Digest            string            `json:"digest"`
}

type Metric struct {
	Name      string          `json:"name"`
	Scope     string          `json:"scope"`
	ValueKind string          `json:"valueKind"`
	Value     json.RawMessage `json:"value"`
	Unit      string          `json:"unit"`
	Boundary  string          `json:"boundary"`
	Metadata  json.RawMessage `json:"metadata"`
}

type Trace struct {
	Kind          string          `json:"kind"`
	SchemaVersion string          `json:"schemaVersion"`
	Value         json.RawMessage `json:"value"`
	Truncated     bool            `json:"truncated"`
}

type CountCoverage struct {
	Observed int `json:"observed"`
	Total    int `json:"total"`
}

type Coverage struct {
	Attempts       CountCoverage `json:"attempts"`
	QueueWaits     CountCoverage `json:"queueWaits"`
	Operations     CountCoverage `json:"operations"`
	Accounting     CountCoverage `json:"accounting"`
	CriticalPath   CountCoverage `json:"criticalPath"`
	TerminalSource bool          `json:"terminalSource"`
}

type ArtifactLineage struct {
	Name      string `json:"name"`
	Schema    string `json:"schema"`
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	SizeBytes int64  `json:"sizeBytes"`
}

type Ratio struct {
	Numerator   int64 `json:"numerator"`
	Denominator int64 `json:"denominator"`
}

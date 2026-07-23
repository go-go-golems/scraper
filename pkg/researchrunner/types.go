// Package researchrunner bridges Researchctl's domain-neutral process protocol
// to the Workflow V3 product without importing Researchctl Go packages.
package researchrunner

import (
	"encoding/json"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const (
	ProtocolVersion     = "researchctl-runner-stdio/v1"
	Domain              = "scraper-workflow"
	DomainSchemaVersion = "scraper-workflow-execution/v1"
	RunnerName          = "scraper-workflow-runner"
	RunnerVersion       = "v1"
)

type ArtifactRef struct {
	Role          string            `json:"role"`
	Kind          string            `json:"kind"`
	ID            string            `json:"id,omitempty"`
	Digest        string            `json:"digest,omitempty"`
	SizeBytes     *int64            `json:"sizeBytes,omitempty"`
	SchemaVersion string            `json:"schemaVersion,omitempty"`
	URI           string            `json:"uri,omitempty"`
	MediaType     string            `json:"mediaType,omitempty"`
	Catalog       json.RawMessage   `json:"catalog,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

type ExecutionIdentity struct {
	SchemaVersion       string            `json:"schemaVersion"`
	IdentityScheme      string            `json:"identityScheme"`
	Domain              string            `json:"domain"`
	DomainSchemaVersion string            `json:"domainSchemaVersion"`
	Inputs              []ArtifactRef     `json:"inputs"`
	DomainConfig        json.RawMessage   `json:"domainConfig"`
	RequestedMeasures   []json.RawMessage `json:"requestedMeasures"`
	Factors             json.RawMessage   `json:"factors,omitempty"`
}

type Specification struct {
	ID                string            `json:"id"`
	IdentityScheme    string            `json:"identityScheme"`
	CanonicalIdentity ExecutionIdentity `json:"canonicalIdentity"`
	DisplayName       string            `json:"displayName"`
	Provenance        json.RawMessage   `json:"provenance,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
}

type ResearchRun struct {
	ID             string `json:"id"`
	ReplicateIndex int64  `json:"replicateIndex"`
	CreatedAt      string `json:"createdAt"`
	ParentRunID    string `json:"parentRunId,omitempty"`
}

type Attempt struct {
	Specification Specification `json:"specification"`
	Run           ResearchRun   `json:"run"`
	AttemptID     string        `json:"attemptId"`
	AttemptIndex  int64         `json:"attemptIndex"`
}

type ResolvedInput struct {
	Reference ArtifactRef `json:"reference"`
	Path      string      `json:"path,omitempty"`
}

type Request struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Attempt         Attempt         `json:"attempt"`
	Inputs          []ResolvedInput `json:"inputs"`
}

type InputBinding struct {
	Role string `json:"role"`
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type PackageIdentity struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	BundleDigest string `json:"bundleDigest"`
}

type TaskCatalog struct {
	Digest   string            `json:"digest"`
	Packages []PackageIdentity `json:"packages"`
}

type ObservationPolicy struct {
	ExportOutputs               bool `json:"exportOutputs"`
	ExportExternalOperations    bool `json:"exportExternalOperations"`
	ExportCanonicalObservations bool `json:"exportCanonicalObservations"`
}

type WorkflowExecution struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Plan          workflowv3.WorkflowPlan `json:"plan"`
	InputBindings map[string]InputBinding `json:"inputBindings"`
	TaskCatalog   TaskCatalog             `json:"taskCatalog"`
	Observation   ObservationPolicy       `json:"observation"`
}

type RunnerRecord struct {
	Name             string `json:"name"`
	RequestedVersion string `json:"requestedVersion,omitempty"`
	ResolvedVersion  string `json:"resolvedVersion"`
}

type DomainVersion struct {
	Domain        string `json:"domain"`
	SchemaVersion string `json:"schemaVersion"`
}

type Hello struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Runner          RunnerRecord    `json:"runner"`
	Domains         []DomainVersion `json:"domains"`
}

type Event struct {
	Type               string          `json:"type"`
	ProducerSequence   *int64          `json:"producerSequence,omitempty"`
	ProducerOccurredAt string          `json:"producerOccurredAt,omitempty"`
	Payload            json.RawMessage `json:"payload"`
}

type Metric struct {
	Name              string          `json:"name"`
	Scope             string          `json:"scope,omitempty"`
	Value             json.RawMessage `json:"value"`
	NumericProjection *float64        `json:"numericProjection,omitempty"`
	TextProjection    string          `json:"textProjection,omitempty"`
	Unit              string          `json:"unit,omitempty"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	SupersedesOrdinal *int64          `json:"supersedesOrdinal,omitempty"`
}

type Trace struct {
	Kind  string          `json:"kind"`
	Value json.RawMessage `json:"value"`
}

type Artifact struct {
	Role          string          `json:"role"`
	Kind          string          `json:"kind"`
	ID            string          `json:"id,omitempty"`
	Name          string          `json:"name"`
	SchemaVersion string          `json:"schemaVersion,omitempty"`
	MediaType     string          `json:"mediaType,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
	Data          []byte          `json:"data"`
}

type Complete struct {
	Status             string          `json:"status"`
	ProducerFinishedAt string          `json:"producerFinishedAt,omitempty"`
	Payload            json.RawMessage `json:"payload"`
}

type RunnerError struct {
	Code      string `json:"code,omitempty"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable,omitempty"`
}

type Frame struct {
	Type     string       `json:"type"`
	Hello    *Hello       `json:"hello,omitempty"`
	Event    *Event       `json:"event,omitempty"`
	Trace    *Trace       `json:"trace,omitempty"`
	Metric   *Metric      `json:"metric,omitempty"`
	Artifact *Artifact    `json:"artifact,omitempty"`
	Complete *Complete    `json:"complete,omitempty"`
	Error    *RunnerError `json:"error,omitempty"`
}

package model

import (
	"encoding/json"
	"time"
)

type WorkflowID string
type OpID string
type ArtifactID string
type SiteName string
type QueueKey string

type WorkflowStatus string

const (
	WorkflowStatusPending   WorkflowStatus = "pending"
	WorkflowStatusRunning   WorkflowStatus = "running"
	WorkflowStatusSucceeded WorkflowStatus = "succeeded"
	WorkflowStatusFailed    WorkflowStatus = "failed"
	WorkflowStatusCanceled  WorkflowStatus = "canceled"
)

type OpStatus string

const (
	OpStatusPending   OpStatus = "pending"
	OpStatusReady     OpStatus = "ready"
	OpStatusRunning   OpStatus = "running"
	OpStatusSucceeded OpStatus = "succeeded"
	OpStatusFailed    OpStatus = "failed"
	OpStatusCanceled  OpStatus = "canceled"
)

type BackoffKind string

const (
	BackoffKindFixed       BackoffKind = "fixed"
	BackoffKindLinear      BackoffKind = "linear"
	BackoffKindExponential BackoffKind = "exponential"
)

type WorkflowRun struct {
	ID        WorkflowID
	Site      SiteName
	Name      string
	Status    WorkflowStatus
	Input     json.RawMessage
	Metadata  map[string]string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Dependency struct {
	OpID     OpID
	Required bool
}

type RetryPolicy struct {
	MaxAttempts    int
	BackoffKind    BackoffKind
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
}

type RetryState struct {
	Attempt       int
	NextAttemptAt *time.Time
	LastError     string
}

type Lease struct {
	WorkerID   string
	Token      string
	AcquiredAt time.Time
	ExpiresAt  time.Time
}

type OpSpec struct {
	ID         OpID
	WorkflowID WorkflowID
	ParentID   *OpID
	Site       SiteName
	Kind       string
	Queue      QueueKey
	DedupKey   string
	Input      json.RawMessage
	DependsOn  []Dependency
	Retry      RetryPolicy
	Metadata   map[string]string
}

type RecordWrite struct {
	Collection string
	Key        string
	Data       json.RawMessage
}

type ArtifactWrite struct {
	ID          ArtifactID
	Name        string
	Kind        string
	ContentType string
	Metadata    map[string]string
	Body        []byte
}

type OpError struct {
	Code       string
	Message    string
	Retryable  bool
	Details    json.RawMessage
	OccurredAt time.Time
}

type OpResult struct {
	OpID        OpID
	Data        json.RawMessage
	Records     []RecordWrite
	Artifacts   []ArtifactWrite
	Emitted     []OpSpec
	EmittedIDs  []OpID
	Error       *OpError
	CompletedAt time.Time
}

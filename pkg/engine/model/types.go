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
type RateLimitKind string

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

const (
	RateLimitKindTokenBucket RateLimitKind = "token_bucket"
)

type RateLimitPolicy struct {
	Kind          RateLimitKind
	RatePerSecond float64
	Burst         int
}

type QueuePolicy struct {
	MaxInFlight int
	RateLimit   *RateLimitPolicy
}

func DefaultQueuePolicy() QueuePolicy {
	return QueuePolicy{
		MaxInFlight: 1,
	}
}

func (p QueuePolicy) Normalize() QueuePolicy {
	ret := p
	if ret.MaxInFlight <= 0 {
		ret.MaxInFlight = 1
	}
	if ret.RateLimit == nil {
		return ret
	}

	limit := *ret.RateLimit
	if limit.Kind == "" {
		limit.Kind = RateLimitKindTokenBucket
	}
	if limit.Kind != RateLimitKindTokenBucket || limit.RatePerSecond <= 0 || limit.Burst <= 0 {
		ret.RateLimit = nil
		return ret
	}

	ret.RateLimit = &limit
	return ret
}

type Lease struct {
	WorkerID   string
	Token      string
	AcquiredAt time.Time
	ExpiresAt  time.Time
}

type OpSpec struct {
	ID          OpID
	WorkflowID  WorkflowID
	ParentID    *OpID
	Site        SiteName
	Kind        string
	Queue       QueueKey
	DedupKey    string
	Input       json.RawMessage
	DependsOn   []Dependency
	Retry       RetryPolicy
	RetryState  RetryState
	Metadata    map[string]string
	CreatedAt   time.Time  `json:"-"`
	UpdatedAt   time.Time  `json:"-"`
	NextReadyAt *time.Time `json:"-"`
}

func (o OpSpec) ReadyAt() time.Time {
	readyAt := o.UpdatedAt.UTC()
	if o.NextReadyAt != nil && o.NextReadyAt.UTC().After(readyAt) {
		readyAt = o.NextReadyAt.UTC()
	}
	if readyAt.IsZero() {
		readyAt = o.CreatedAt.UTC()
	}
	return readyAt
}

func (o OpSpec) QueueWaitDuration(now time.Time) time.Duration {
	readyAt := o.ReadyAt()
	if readyAt.IsZero() {
		return 0
	}
	wait := now.UTC().Sub(readyAt)
	if wait < 0 {
		return 0
	}
	return wait
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

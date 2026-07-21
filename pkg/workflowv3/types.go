package workflowv3

import "time"

const (
	IRSchema             = "scraper-workflow-ir/v3"
	PlanSchema           = "scraper-workflow-plan/v3"
	TaskABI              = "scraper-js-task/v1"
	ResourceCPUDefault   = "cpu.default"
	ItemManifestSchemaV1 = "scraper-workflow-item-manifest/v1"
)

type RunID string
type NodeKey string

type ArtifactRef struct {
	Schema    string `json:"schema"`
	Digest    string `json:"digest"`
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Locator   string `json:"locator"`
}

type TaskKey struct {
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

type ImplementationIdentity struct {
	TaskKey
	BundleDigest string `json:"bundleDigest"`
	Entrypoint   string `json:"entrypoint"`
	ABI          string `json:"abi"`
}

type RetryPolicy struct {
	MaxAttempts   int   `json:"maxAttempts"`
	BackoffMillis int64 `json:"backoffMillis"`
}

type TaskSpec struct {
	Identity      ImplementationIdentity `json:"identity"`
	Inputs        map[string]string      `json:"inputs"`
	Outputs       map[string]string      `json:"outputs"`
	Modules       []string               `json:"modules,omitempty"`
	ResourceClass string                 `json:"resourceClass"`
	Retry         RetryPolicy            `json:"retry"`
}

type ValueRef struct {
	Source  string  `json:"source"`
	Name    string  `json:"name,omitempty"`
	NodeKey NodeKey `json:"nodeKey,omitempty"`
	MapKey  string  `json:"mapKey,omitempty"`
	Port    string  `json:"port,omitempty"`
	Schema  string  `json:"schema"`
}

type SetRef struct {
	Source         string `json:"source"`
	Name           string `json:"name,omitempty"`
	MapKey         string `json:"mapKey,omitempty"`
	ItemSchema     string `json:"itemSchema"`
	ManifestSchema string `json:"manifestSchema"`
}

type IRInput struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type IRNode struct {
	Key       NodeKey             `json:"key"`
	Task      TaskKey             `json:"task"`
	Bindings  map[string]ValueRef `json:"bindings"`
	DependsOn []NodeKey           `json:"dependsOn,omitempty"`
}

type IRSetInput struct {
	Name           string `json:"name"`
	ItemSchema     string `json:"itemSchema"`
	ManifestSchema string `json:"manifestSchema"`
}

type MapPolicy struct {
	PageSize             int `json:"pageSize"`
	MaxItems             int `json:"maxItems"`
	MaxMaterializedAhead int `json:"maxMaterializedAhead"`
}

type IRMap struct {
	Key      string              `json:"key"`
	Source   SetRef              `json:"source"`
	ItemTask TaskKey             `json:"itemTask"`
	Bindings map[string]ValueRef `json:"bindings"`
	Policy   MapPolicy           `json:"policy"`
}

type IRSetOutput struct {
	Name  string `json:"name"`
	Value SetRef `json:"value"`
}

type IROutput struct {
	Name  string   `json:"name"`
	Value ValueRef `json:"value"`
}

type WorkflowIR struct {
	Schema     string        `json:"schema"`
	Name       string        `json:"name"`
	Inputs     []IRInput     `json:"inputs"`
	SetInputs  []IRSetInput  `json:"setInputs,omitempty"`
	Nodes      []IRNode      `json:"nodes"`
	Maps       []IRMap       `json:"maps,omitempty"`
	Outputs    []IROutput    `json:"outputs"`
	SetOutputs []IRSetOutput `json:"setOutputs,omitempty"`
}

type PlanNode struct {
	Key            NodeKey                `json:"key"`
	Implementation ImplementationIdentity `json:"implementation"`
	Bindings       map[string]ValueRef    `json:"bindings"`
	DependsOn      []NodeKey              `json:"dependsOn,omitempty"`
	InputSchemas   map[string]string      `json:"inputSchemas"`
	OutputSchemas  map[string]string      `json:"outputSchemas"`
	Modules        []string               `json:"modules,omitempty"`
	ResourceClass  string                 `json:"resourceClass"`
	Retry          RetryPolicy            `json:"retry"`
}

type PlanMap struct {
	Key            string                 `json:"key"`
	Source         SetRef                 `json:"source"`
	Implementation ImplementationIdentity `json:"implementation"`
	Bindings       map[string]ValueRef    `json:"bindings"`
	InputSchemas   map[string]string      `json:"inputSchemas"`
	OutputSchemas  map[string]string      `json:"outputSchemas"`
	Modules        []string               `json:"modules,omitempty"`
	ResourceClass  string                 `json:"resourceClass"`
	Retry          RetryPolicy            `json:"retry"`
	Policy         MapPolicy              `json:"policy"`
}

type WorkflowPlan struct {
	Schema        string        `json:"schema"`
	Name          string        `json:"name"`
	IRDigest      string        `json:"irDigest"`
	CatalogDigest string        `json:"catalogDigest"`
	Inputs        []IRInput     `json:"inputs"`
	SetInputs     []IRSetInput  `json:"setInputs,omitempty"`
	Nodes         []PlanNode    `json:"nodes"`
	Maps          []PlanMap     `json:"maps,omitempty"`
	Outputs       []IROutput    `json:"outputs"`
	SetOutputs    []IRSetOutput `json:"setOutputs,omitempty"`
	Digest        string        `json:"digest"`
}

type Failure struct {
	Class     string `json:"class"`
	Code      string `json:"code"`
	Retryable bool   `json:"retryable"`
	Message   string `json:"message"`
}

type Attempt struct {
	RunID              RunID     `json:"runId"`
	NodeKey            NodeKey   `json:"nodeKey"`
	Number             int       `json:"number"`
	Status             string    `json:"status"`
	LeaseToken         string    `json:"-"`
	CancelEpoch        int64     `json:"cancelEpoch"`
	RegistryGeneration string    `json:"registryGeneration"`
	ResourceClass      string    `json:"resourceClass"`
	StartedAt          time.Time `json:"startedAt"`
	FinishedAt         time.Time `json:"finishedAt,omitempty"`
	Failure            *Failure  `json:"failure,omitempty"`
}

type Lease struct {
	RunID              RunID
	NodeKey            NodeKey
	Attempt            int
	Token              string
	CancelEpoch        int64
	ExpiresAt          time.Time
	PlanNode           PlanNode
	RegistryGeneration string
}

type RunSnapshot struct {
	RunID      RunID                  `json:"runId"`
	Status     string                 `json:"status"`
	PlanDigest string                 `json:"planDigest"`
	Outputs    map[string]ArtifactRef `json:"outputs"`
	Attempts   []Attempt              `json:"attempts"`
}

type QueueSnapshot struct {
	Ready            int            `json:"ready"`
	ActiveByResource map[string]int `json:"activeByResource"`
	BlockedByReason  map[string]int `json:"blockedByReason"`
}

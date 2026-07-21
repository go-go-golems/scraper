package workflowv3

import "time"

const (
	IRSchema                   = "scraper-workflow-ir/v3"
	PlanSchema                 = "scraper-workflow-plan/v3"
	TaskABI                    = "scraper-js-task/v1"
	ResourceCPUDefault         = "cpu.default"
	ItemManifestSchemaV1       = "scraper-workflow-item-manifest/v1"
	ReductionPartitionSchemaV1 = "scraper-workflow-reduction-partition/v1"
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

type BudgetAmount struct {
	Dimension string `json:"dimension"`
	Units     int64  `json:"units"`
}

type BudgetAccount struct {
	Account      string         `json:"account"`
	Limits       []BudgetAmount `json:"limits"`
	PolicyDigest string         `json:"policyDigest"`
}

type BudgetClaim struct {
	Account      string         `json:"account"`
	Reserve      []BudgetAmount `json:"reserve"`
	OnExhausted  string         `json:"onExhausted"`
	ApprovalGate NodeKey        `json:"approvalGate,omitempty"`
}

type PlanBudgetClaim struct {
	Account      string         `json:"account"`
	Requested    []BudgetAmount `json:"requested"`
	Effective    []BudgetAmount `json:"effective"`
	OnExhausted  string         `json:"onExhausted"`
	ApprovalGate NodeKey        `json:"approvalGate,omitempty"`
}

type TaskSpec struct {
	Identity      ImplementationIdentity `json:"identity"`
	Inputs        map[string]string      `json:"inputs"`
	Outputs       map[string]string      `json:"outputs"`
	Modules       []string               `json:"modules,omitempty"`
	ResourceClass string                 `json:"resourceClass"`
	Retry         RetryPolicy            `json:"retry"`
	BudgetMaximum *BudgetClaim           `json:"budgetMaximum,omitempty"`
}

type ValueRef struct {
	Source    string  `json:"source"`
	Name      string  `json:"name,omitempty"`
	NodeKey   NodeKey `json:"nodeKey,omitempty"`
	MapKey    string  `json:"mapKey,omitempty"`
	ReduceKey string  `json:"reduceKey,omitempty"`
	GateKey   NodeKey `json:"gateKey,omitempty"`
	Port      string  `json:"port,omitempty"`
	Schema    string  `json:"schema"`
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
	Budget    *BudgetClaim        `json:"budget,omitempty"`
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
	Budget   *BudgetClaim        `json:"budget,omitempty"`
}

type IRSetOutput struct {
	Name  string `json:"name"`
	Value SetRef `json:"value"`
}

type ReducePolicy struct {
	FanIn     int `json:"fanIn"`
	MaxLevels int `json:"maxLevels"`
}

type IRReduce struct {
	Key           string              `json:"key"`
	Source        SetRef              `json:"source"`
	PartitionTask TaskKey             `json:"partitionTask"`
	Bindings      map[string]ValueRef `json:"bindings"`
	Policy        ReducePolicy        `json:"policy"`
	Budget        *BudgetClaim        `json:"budget,omitempty"`
}

type GatePolicy struct {
	DecisionSchema string `json:"decisionSchema"`
	OnReject       string `json:"onReject"`
	OnExpire       string `json:"onExpire"`
	TimeoutMillis  int64  `json:"timeoutMillis,omitempty"`
	RequiredRole   string `json:"requiredRole"`
}

type IRGate struct {
	Key       NodeKey    `json:"key"`
	DependsOn []NodeKey  `json:"dependsOn,omitempty"`
	Policy    GatePolicy `json:"policy"`
}

type PlanGate struct {
	Key              NodeKey    `json:"key"`
	DependsOn        []NodeKey  `json:"dependsOn,omitempty"`
	Policy           GatePolicy `json:"policy"`
	PolicyDigest     string     `json:"policyDigest"`
	BudgetActivation bool       `json:"budgetActivation,omitempty"`
}

type IROutput struct {
	Name  string   `json:"name"`
	Value ValueRef `json:"value"`
}

type WorkflowIR struct {
	Schema     string          `json:"schema"`
	Name       string          `json:"name"`
	Inputs     []IRInput       `json:"inputs"`
	SetInputs  []IRSetInput    `json:"setInputs,omitempty"`
	Budgets    []BudgetAccount `json:"budgets,omitempty"`
	Nodes      []IRNode        `json:"nodes"`
	Maps       []IRMap         `json:"maps,omitempty"`
	Reductions []IRReduce      `json:"reductions,omitempty"`
	Gates      []IRGate        `json:"gates,omitempty"`
	Outputs    []IROutput      `json:"outputs"`
	SetOutputs []IRSetOutput   `json:"setOutputs,omitempty"`
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
	Budget         *PlanBudgetClaim       `json:"budget,omitempty"`
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
	Budget         *PlanBudgetClaim       `json:"budget,omitempty"`
}

type PlanReduce struct {
	Key            string                 `json:"key"`
	Source         SetRef                 `json:"source"`
	Implementation ImplementationIdentity `json:"implementation"`
	Bindings       map[string]ValueRef    `json:"bindings"`
	InputSchemas   map[string]string      `json:"inputSchemas"`
	OutputSchemas  map[string]string      `json:"outputSchemas"`
	Modules        []string               `json:"modules,omitempty"`
	ResourceClass  string                 `json:"resourceClass"`
	Retry          RetryPolicy            `json:"retry"`
	Policy         ReducePolicy           `json:"policy"`
	Budget         *PlanBudgetClaim       `json:"budget,omitempty"`
}

type WorkflowPlan struct {
	Schema        string          `json:"schema"`
	Name          string          `json:"name"`
	IRDigest      string          `json:"irDigest"`
	CatalogDigest string          `json:"catalogDigest"`
	Inputs        []IRInput       `json:"inputs"`
	SetInputs     []IRSetInput    `json:"setInputs,omitempty"`
	Budgets       []BudgetAccount `json:"budgets,omitempty"`
	Nodes         []PlanNode      `json:"nodes"`
	Maps          []PlanMap       `json:"maps,omitempty"`
	Reductions    []PlanReduce    `json:"reductions,omitempty"`
	Gates         []PlanGate      `json:"gates,omitempty"`
	Outputs       []IROutput      `json:"outputs"`
	SetOutputs    []IRSetOutput   `json:"setOutputs,omitempty"`
	Digest        string          `json:"digest"`
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
	FailureCount       int
	Token              string
	CancelEpoch        int64
	ExpiresAt          time.Time
	PlanNode           PlanNode
	RegisteredTask     RegisteredTask
	RegistryGeneration string
	ReleaseGeneration  func()
}

type RunSnapshot struct {
	RunID      RunID                  `json:"runId"`
	Status     string                 `json:"status"`
	PlanDigest string                 `json:"planDigest"`
	Outputs    map[string]ArtifactRef `json:"outputs"`
	Attempts   []Attempt              `json:"attempts"`
}

type MapProgress struct {
	RunID                RunID  `json:"runId"`
	MapKey               string `json:"mapKey"`
	Status               string `json:"status"`
	TotalItems           int    `json:"totalItems"`
	NextIndex            int    `json:"nextIndex"`
	MaterializedItems    int    `json:"materializedItems"`
	TerminalItems        int    `json:"terminalItems"`
	BacklogToMaterialize int    `json:"backlogToMaterialize"`
	BacklogToExecute     int    `json:"backlogToExecute"`
}

type ReductionProgress struct {
	RunID               RunID  `json:"runId"`
	ReduceKey           string `json:"reduceKey"`
	Status              string `json:"status"`
	SourceItems         int    `json:"sourceItems"`
	CurrentLevel        int    `json:"currentLevel"`
	PartitionsTotal     int    `json:"partitionsTotal"`
	PartitionsSucceeded int    `json:"partitionsSucceeded"`
	RootReady           bool   `json:"rootReady"`
}

type RegistryGenerationProgress struct {
	Generation     string `json:"generation"`
	State          string `json:"state"`
	References     int    `json:"references"`
	Failures       int    `json:"failures"`
	QuarantineCode string `json:"quarantineCode,omitempty"`
}

type BudgetProgress struct {
	RunID        RunID  `json:"runId"`
	Account      string `json:"account"`
	Dimension    string `json:"dimension"`
	Limit        int64  `json:"limit"`
	Used         int64  `json:"used"`
	Reserved     int64  `json:"reserved"`
	Remaining    int64  `json:"remaining"`
	Version      int64  `json:"version"`
	PolicyDigest string `json:"policyDigest"`
}

type TerminalRate struct {
	WindowSeconds int64 `json:"windowSeconds"`
	Terminal      int   `json:"terminal"`
	Succeeded     int   `json:"succeeded"`
	Failed        int   `json:"failed"`
}

type OperationalEvent struct {
	Sequence  int64     `json:"sequence"`
	RunID     RunID     `json:"runId"`
	NodeKey   NodeKey   `json:"nodeKey,omitempty"`
	Type      string    `json:"type"`
	DataJSON  string    `json:"dataJson"`
	CreatedAt time.Time `json:"createdAt"`
}

type OperationalSnapshot struct {
	AsOf                time.Time                    `json:"asOf"`
	EventSequence       int64                        `json:"eventSequence"`
	RunStatuses         map[string]int               `json:"runStatuses"`
	NodeStatuses        map[string]int               `json:"nodeStatuses"`
	AttemptStatuses     map[string]int               `json:"attemptStatuses"`
	GateStatuses        map[string]int               `json:"gateStatuses"`
	RetryAttempts       int                          `json:"retryAttempts"`
	LeaseLosses         int                          `json:"leaseLosses"`
	OldestRunningAgeMS  int64                        `json:"oldestRunningAgeMs"`
	Rates               []TerminalRate               `json:"rates"`
	Queue               QueueSnapshot                `json:"queue"`
	Budgets             []BudgetProgress             `json:"budgets,omitempty"`
	Gates               []GateProgress               `json:"gates,omitempty"`
	GatesTruncated      bool                         `json:"gatesTruncated,omitempty"`
	RegistryGenerations []RegistryGenerationProgress `json:"registryGenerations,omitempty"`
}

type QueueSnapshot struct {
	Ready               int                          `json:"ready"`
	ActiveByResource    map[string]int               `json:"activeByResource"`
	BlockedByReason     map[string]int               `json:"blockedByReason"`
	Maps                []MapProgress                `json:"maps,omitempty"`
	Reductions          []ReductionProgress          `json:"reductions,omitempty"`
	RegistryGenerations []RegistryGenerationProgress `json:"registryGenerations,omitempty"`
}

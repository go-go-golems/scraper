package workflowv3

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func gateCatalog(t *testing.T) *Catalog {
	t.Helper()
	catalog, err := NewCatalog(
		TaskSpec{Identity: ImplementationIdentity{TaskKey: TaskKey{Kind: "gate.before", Version: "v1"}, BundleDigest: "sha256:" + strings.Repeat("1", 64), Entrypoint: "tasks.cjs#before", ABI: TaskABI}, Inputs: map[string]string{"input": "input/v1"}, Outputs: map[string]string{"output": "prepared/v1"}, ResourceClass: ResourceCPUDefault, Retry: RetryPolicy{MaxAttempts: 1}, BudgetMaximum: &BudgetClaim{Account: "provider", OnExhausted: BudgetExhaustBlock, Reserve: []BudgetAmount{{Dimension: "requests", Units: 1}}}},
		TaskSpec{Identity: ImplementationIdentity{TaskKey: TaskKey{Kind: "gate.after", Version: "v1"}, BundleDigest: "sha256:" + strings.Repeat("2", 64), Entrypoint: "tasks.cjs#after", ABI: TaskABI}, Inputs: map[string]string{"decision": "decision/v1"}, Outputs: map[string]string{"output": "output/v1"}, ResourceClass: ResourceCPUDefault, Retry: RetryPolicy{MaxAttempts: 1}, BudgetMaximum: &BudgetClaim{Account: "provider", OnExhausted: BudgetExhaustBlock, Reserve: []BudgetAmount{{Dimension: "requests", Units: 1}}}},
	)
	require.NoError(t, err)
	return catalog
}

func validGateIR() WorkflowIR {
	return WorkflowIR{
		Schema: IRSchema, Name: "gate",
		Inputs:  []IRInput{{Name: "input", Schema: "input/v1"}},
		Nodes:   []IRNode{{Key: "before", Task: TaskKey{Kind: "gate.before", Version: "v1"}, Bindings: map[string]ValueRef{"input": {Source: "input", Name: "input", Schema: "input/v1"}}}, {Key: "after", Task: TaskKey{Kind: "gate.after", Version: "v1"}, Bindings: map[string]ValueRef{"decision": {Source: "gate-output", GateKey: "review", Schema: "decision/v1"}}}},
		Gates:   []IRGate{{Key: "review", DependsOn: []NodeKey{"before"}, Policy: GatePolicy{DecisionSchema: "decision/v1", RequiredRole: "reviewer", OnReject: GateFailRun, OnExpire: GateFailRun, TimeoutMillis: 1000}}},
		Outputs: []IROutput{{Name: "output", Value: ValueRef{Source: "node-output", NodeKey: "after", Port: "output", Schema: "output/v1"}}},
	}
}

func TestGateCompilerPinsPolicyAndRejectsInvalidGraphs(t *testing.T) {
	catalog := gateCatalog(t)
	plan, err := Compile(validGateIR(), catalog)
	require.NoError(t, err)
	require.Len(t, plan.Gates, 1)
	require.NotEmpty(t, plan.Gates[0].PolicyDigest)
	require.False(t, plan.Gates[0].BudgetActivation)

	tests := []struct {
		name    string
		mutate  func(*WorkflowIR)
		message string
	}{
		{name: "duplicate", mutate: func(ir *WorkflowIR) { ir.Gates = append(ir.Gates, ir.Gates[0]) }, message: "duplicate gate key"},
		{name: "unknown dependency", mutate: func(ir *WorkflowIR) { ir.Gates[0].DependsOn = []NodeKey{"missing"} }, message: "unknown dependency"},
		{name: "invalid role", mutate: func(ir *WorkflowIR) { ir.Gates[0].Policy.RequiredRole = "secret role" }, message: "required role"},
		{name: "invalid timeout", mutate: func(ir *WorkflowIR) { ir.Gates[0].Policy.TimeoutMillis = -1 }, message: "timeout"},
		{name: "unsupported branch", mutate: func(ir *WorkflowIR) { ir.Gates[0].Policy.OnReject = "cancel-branch" }, message: "branch cancellation"},
		{name: "cycle", mutate: func(ir *WorkflowIR) { ir.Gates[0].DependsOn = []NodeKey{"after"} }, message: "dependency cycle"},
		{name: "shared budget activation gate", mutate: func(ir *WorkflowIR) {
			ir.Budgets = []BudgetAccount{{Account: "provider", PolicyDigest: "sha256:" + strings.Repeat("c", 64), Limits: []BudgetAmount{{Dimension: "requests", Units: 2}}}}
			claim := &BudgetClaim{Account: "provider", OnExhausted: BudgetExhaustRequireApproval, ApprovalGate: "review", Reserve: []BudgetAmount{{Dimension: "requests", Units: 1}}}
			ir.Nodes[0].Budget = claim
			copyClaim := *claim
			copyClaim.Reserve = append([]BudgetAmount(nil), claim.Reserve...)
			ir.Nodes[1].Budget = &copyClaim
		}, message: "each compiled claim requires a dedicated gate"},
		{name: "budget activation data consumer", mutate: func(ir *WorkflowIR) {
			ir.Budgets = []BudgetAccount{{Account: "provider", PolicyDigest: "sha256:" + strings.Repeat("c", 64), Limits: []BudgetAmount{{Dimension: "requests", Units: 1}}}}
			ir.Nodes[0].Budget = &BudgetClaim{Account: "provider", OnExhausted: BudgetExhaustRequireApproval, ApprovalGate: "review", Reserve: []BudgetAmount{{Dimension: "requests", Units: 1}}}
		}, message: "cannot consume budget-activation gate"},
		{name: "budget activation cycle", mutate: func(ir *WorkflowIR) {
			ir.Budgets = []BudgetAccount{{Account: "provider", PolicyDigest: "sha256:" + strings.Repeat("c", 64), Limits: []BudgetAmount{{Dimension: "requests", Units: 1}}}}
			ir.Nodes[0].Budget = &BudgetClaim{Account: "provider", OnExhausted: BudgetExhaustRequireApproval, ApprovalGate: "review", Reserve: []BudgetAmount{{Dimension: "requests", Units: 1}}}
			ir.Nodes = ir.Nodes[:1]
			ir.Outputs[0].Value = ValueRef{Source: "node-output", NodeKey: "before", Port: "output", Schema: "prepared/v1"}
			ir.Gates[0].DependsOn = []NodeKey{"before"}
		}, message: "dependency cycle"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ir := validGateIR()
			test.mutate(&ir)
			_, err := Compile(ir, catalog)
			require.ErrorContains(t, err, test.message)
		})
	}
}

func TestGateDecisionValidationRejectsMalformedAuthorityAndPayload(t *testing.T) {
	ref := ArtifactRef{Schema: "decision/v1", Digest: "sha256:" + strings.Repeat("3", 64), MediaType: "application/json", Size: 2, Locator: "cas://decision"}
	command := GateDecisionCommand{RunID: "run", GateKey: "gate", ExpectedVersion: 1, Decision: "approve", DecisionCode: "APPROVED", ActorID: "actor@example", AuthorizedRole: "reviewer", DecisionRef: &ref}
	require.NoError(t, ValidateGateDecisionCommand(command))
	command.ActorID = strings.Repeat("a", 129)
	require.ErrorContains(t, ValidateGateDecisionCommand(command), "actor")
	command.ActorID = "actor@example"
	command.DecisionCode = "free form comment"
	require.ErrorContains(t, ValidateGateDecisionCommand(command), "decision code")
	command.DecisionCode = "APPROVED"
	command.DecisionRef = nil
	require.ErrorContains(t, ValidateGateDecisionCommand(command), "requires a decision artifact")
}

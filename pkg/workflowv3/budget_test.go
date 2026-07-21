package workflowv3

import (
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBudgetValidationAndCheckedCost(t *testing.T) {
	digest := "sha256:" + strings.Repeat("a", 64)
	require.NoError(t, ValidateBudgetAccount(BudgetAccount{
		Account: "provider.main", PolicyDigest: digest,
		Limits: []BudgetAmount{{Dimension: "cost_microunits", Units: 10}, {Dimension: "requests", Units: 2}},
	}))
	require.ErrorContains(t, ValidateBudgetAccount(BudgetAccount{
		Account: "provider.main", PolicyDigest: digest,
		Limits: []BudgetAmount{{Dimension: "requests", Units: 2}, {Dimension: "cost_microunits", Units: 10}},
	}), "strictly sorted")
	require.ErrorContains(t, ValidateBudgetClaim(BudgetClaim{
		Account: "provider.main", OnExhausted: BudgetExhaustBlock,
		Reserve: []BudgetAmount{{Dimension: "unknown", Units: 1}},
	}), "unknown budget dimension")
	require.ErrorContains(t, ValidateBudgetClaim(BudgetClaim{
		Account: "provider.main", OnExhausted: BudgetExhaustBlock,
		Reserve: []BudgetAmount{{Dimension: "requests", Units: -1}},
	}), "invalid units")

	cost, err := CostMicrounits(1_500_001, 2_000_000)
	require.NoError(t, err)
	require.Equal(t, int64(3_000_002), cost)
	_, err = CostMicrounits(math.MaxInt64, math.MaxInt64)
	require.ErrorContains(t, err, "overflows")
}

func TestCompileBudgetClaimsRequestedAndEffective(t *testing.T) {
	maximum := &BudgetClaim{
		Account: "provider", OnExhausted: BudgetExhaustBlock,
		Reserve: []BudgetAmount{{Dimension: "cost_microunits", Units: 100}, {Dimension: "requests", Units: 2}},
	}
	catalog, err := NewCatalog(TaskSpec{
		Identity: ImplementationIdentity{TaskKey: TaskKey{Kind: "provider.call", Version: "v1"}, BundleDigest: "sha256:" + strings.Repeat("b", 64), Entrypoint: "tasks.cjs#run", ABI: TaskABI},
		Inputs:   map[string]string{"input": "input/v1"}, Outputs: map[string]string{"output": "output/v1"},
		ResourceClass: "network.provider", Retry: RetryPolicy{MaxAttempts: 2}, BudgetMaximum: maximum,
	})
	require.NoError(t, err)
	claim := &BudgetClaim{
		Account: "provider", OnExhausted: BudgetExhaustBlock,
		Reserve: []BudgetAmount{{Dimension: "cost_microunits", Units: 40}, {Dimension: "requests", Units: 1}},
	}
	ir := WorkflowIR{
		Schema: IRSchema, Name: "budgeted",
		Inputs:  []IRInput{{Name: "input", Schema: "input/v1"}},
		Budgets: []BudgetAccount{{Account: "provider", PolicyDigest: "sha256:" + strings.Repeat("c", 64), Limits: []BudgetAmount{{Dimension: "cost_microunits", Units: 80}, {Dimension: "requests", Units: 1}}}},
		Nodes:   []IRNode{{Key: "call", Task: TaskKey{Kind: "provider.call", Version: "v1"}, Bindings: map[string]ValueRef{"input": {Source: "input", Name: "input", Schema: "input/v1"}}, Budget: claim}},
		Outputs: []IROutput{{Name: "output", Value: ValueRef{Source: "node-output", NodeKey: "call", Port: "output", Schema: "output/v1"}}},
	}
	plan, err := Compile(ir, catalog)
	require.NoError(t, err)
	require.Equal(t, claim.Reserve, plan.Nodes[0].Budget.Requested)
	require.Equal(t, claim.Reserve, plan.Nodes[0].Budget.Effective)

	ir.Nodes[0].Budget.Reserve[1].Units = 3
	_, err = Compile(ir, catalog)
	require.ErrorContains(t, err, "exceeds task maximum")
}

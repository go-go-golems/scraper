package workflowv3

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

const (
	BudgetExhaustFailRun         = "fail-run"
	BudgetExhaustBlock           = "block"
	BudgetExhaustRequireApproval = "require-approval"
)

var budgetAccountPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)

var budgetDimensions = map[string]struct{}{
	"requests": {}, "input_tokens": {}, "output_tokens": {},
	"embedding_tokens": {}, "input_bytes": {}, "output_bytes": {},
	"cost_microunits": {},
}

func ValidateBudgetAccount(account BudgetAccount) error {
	if !budgetAccountPattern.MatchString(account.Account) {
		return fmt.Errorf("budget account %q is invalid", account.Account)
	}
	if len(account.Limits) == 0 {
		return fmt.Errorf("budget account %q requires limits", account.Account)
	}
	if err := validateBudgetAmounts(account.Limits, true); err != nil {
		return fmt.Errorf("budget account %q: %w", account.Account, err)
	}
	if strings.TrimSpace(account.PolicyDigest) == "" {
		return fmt.Errorf("budget account %q policy digest is required", account.Account)
	}
	if err := validateSHA256Digest(account.PolicyDigest); err != nil {
		return fmt.Errorf("budget account %q policy digest: %w", account.Account, err)
	}
	return nil
}

func ValidateBudgetUsage(usage []BudgetAmount) error {
	return validateBudgetAmounts(usage, true)
}

func ValidateBudgetClaim(claim BudgetClaim) error {
	if !budgetAccountPattern.MatchString(claim.Account) {
		return fmt.Errorf("budget claim account %q is invalid", claim.Account)
	}
	if len(claim.Reserve) == 0 {
		return fmt.Errorf("budget claim %q requires reservations", claim.Account)
	}
	if err := validateBudgetAmounts(claim.Reserve, false); err != nil {
		return fmt.Errorf("budget claim %q: %w", claim.Account, err)
	}
	switch claim.OnExhausted {
	case BudgetExhaustFailRun, BudgetExhaustBlock:
		if claim.ApprovalGate != "" {
			return fmt.Errorf("budget claim %q has an approval gate without require-approval", claim.Account)
		}
	case BudgetExhaustRequireApproval:
		if strings.TrimSpace(string(claim.ApprovalGate)) == "" {
			return fmt.Errorf("budget claim %q requires an approval gate", claim.Account)
		}
	default:
		return fmt.Errorf("budget claim %q has invalid exhaustion policy %q", claim.Account, claim.OnExhausted)
	}
	return nil
}

func validateBudgetAmounts(amounts []BudgetAmount, allowZero bool) error {
	previous := ""
	for _, amount := range amounts {
		if _, ok := budgetDimensions[amount.Dimension]; !ok {
			return fmt.Errorf("unknown budget dimension %q", amount.Dimension)
		}
		if amount.Dimension <= previous {
			return fmt.Errorf("budget dimensions must be strictly sorted and unique")
		}
		if amount.Units < 0 || (!allowZero && amount.Units == 0) {
			return fmt.Errorf("budget dimension %q has invalid units %d", amount.Dimension, amount.Units)
		}
		previous = amount.Dimension
	}
	return nil
}

func compileBudgetClaim(requested, maximum *BudgetClaim, accounts map[string]BudgetAccount) (*PlanBudgetClaim, error) {
	if requested == nil {
		return nil, nil
	}
	if err := ValidateBudgetClaim(*requested); err != nil {
		return nil, err
	}
	if _, ok := accounts[requested.Account]; !ok {
		return nil, fmt.Errorf("budget claim references unknown account %q", requested.Account)
	}
	if maximum == nil {
		return nil, fmt.Errorf("task does not permit a budget claim")
	}
	if err := ValidateBudgetClaim(*maximum); err != nil {
		return nil, fmt.Errorf("task budget maximum: %w", err)
	}
	if maximum.Account != requested.Account {
		return nil, fmt.Errorf("budget account %q does not match task maximum %q", requested.Account, maximum.Account)
	}
	maxima := make(map[string]int64, len(maximum.Reserve))
	for _, amount := range maximum.Reserve {
		maxima[amount.Dimension] = amount.Units
	}
	for _, amount := range requested.Reserve {
		limit, ok := maxima[amount.Dimension]
		if !ok || amount.Units > limit {
			return nil, fmt.Errorf("budget %s reservation %d exceeds task maximum", amount.Dimension, amount.Units)
		}
	}
	return &PlanBudgetClaim{
		Account: requested.Account, Requested: cloneBudgetAmounts(requested.Reserve),
		Effective: cloneBudgetAmounts(requested.Reserve), OnExhausted: requested.OnExhausted,
		ApprovalGate: requested.ApprovalGate,
	}, nil
}

func cloneBudgetAccounts(accounts []BudgetAccount) []BudgetAccount {
	ret := make([]BudgetAccount, len(accounts))
	for index, account := range accounts {
		ret[index] = account
		ret[index].Limits = cloneBudgetAmounts(account.Limits)
	}
	return ret
}

func cloneBudgetAmounts(amounts []BudgetAmount) []BudgetAmount {
	return append([]BudgetAmount(nil), amounts...)
}

func cloneBudgetClaim(claim *BudgetClaim) *BudgetClaim {
	if claim == nil {
		return nil
	}
	ret := *claim
	ret.Reserve = cloneBudgetAmounts(claim.Reserve)
	return &ret
}

func SortBudgetAmounts(amounts []BudgetAmount) {
	sort.Slice(amounts, func(i, j int) bool { return amounts[i].Dimension < amounts[j].Dimension })
}

// CostMicrounits returns ceil(units*pricePerMillion/1_000_000) with checked
// nonnegative integer arithmetic.
func CostMicrounits(units, pricePerMillion int64) (int64, error) {
	if units < 0 || pricePerMillion < 0 {
		return 0, fmt.Errorf("cost inputs must be nonnegative")
	}
	whole, remainder := units/1_000_000, units%1_000_000
	if whole != 0 && pricePerMillion > math.MaxInt64/whole {
		return 0, fmt.Errorf("cost calculation overflows int64")
	}
	cost := whole * pricePerMillion
	if remainder != 0 && pricePerMillion > math.MaxInt64/remainder {
		return 0, fmt.Errorf("cost calculation overflows int64")
	}
	product := remainder * pricePerMillion
	partial := product / 1_000_000
	if product%1_000_000 != 0 {
		partial++
	}
	if partial > math.MaxInt64-cost {
		return 0, fmt.Errorf("cost calculation overflows int64")
	}
	return cost + partial, nil
}

package workflowv3

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const GateFailRun = "fail-run"

var gateRolePattern = regexp.MustCompile(`^[a-z][a-z0-9._:-]{0,63}$`)
var gateDecisionCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,63}$`)
var gateActorPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._@-]{0,127}$`)

func ValidateGatePolicy(policy GatePolicy) error {
	if strings.TrimSpace(policy.DecisionSchema) == "" {
		return fmt.Errorf("gate decision schema is required")
	}
	if !gateRolePattern.MatchString(policy.RequiredRole) {
		return fmt.Errorf("gate required role %q is invalid", policy.RequiredRole)
	}
	if policy.OnReject != GateFailRun || policy.OnExpire != GateFailRun {
		return fmt.Errorf("gate branch cancellation is not supported; reject and expiry policies must be fail-run")
	}
	if policy.TimeoutMillis < 0 || policy.TimeoutMillis > int64((30*24*time.Hour)/time.Millisecond) {
		return fmt.Errorf("gate timeout is invalid")
	}
	return nil
}

type GateDecisionCommand struct {
	RunID           RunID
	GateKey         NodeKey
	ExpectedVersion int64
	Decision        string
	DecisionCode    string
	ActorID         string
	AuthorizedRole  string
	DecisionRef     *ArtifactRef
}

func ValidateGateDecisionCommand(command GateDecisionCommand) error {
	if strings.TrimSpace(string(command.RunID)) == "" || strings.TrimSpace(string(command.GateKey)) == "" {
		return fmt.Errorf("gate run and key are required")
	}
	if command.ExpectedVersion < 1 {
		return fmt.Errorf("gate expected version must be positive")
	}
	if command.Decision != "approve" && command.Decision != "reject" {
		return fmt.Errorf("gate decision must be approve or reject")
	}
	if !gateDecisionCodePattern.MatchString(command.DecisionCode) {
		return fmt.Errorf("gate decision code is invalid")
	}
	if !gateActorPattern.MatchString(command.ActorID) {
		return fmt.Errorf("gate actor identity is invalid")
	}
	if !gateRolePattern.MatchString(command.AuthorizedRole) {
		return fmt.Errorf("gate authorized role is invalid")
	}
	if command.Decision == "approve" && command.DecisionRef == nil {
		return fmt.Errorf("gate approval requires a decision artifact reference")
	}
	if command.DecisionRef != nil {
		if err := ValidateArtifactRef(*command.DecisionRef); err != nil {
			return fmt.Errorf("gate decision artifact: %w", err)
		}
	}
	return nil
}

type GateProgress struct {
	RunID               RunID      `json:"runId"`
	GateKey             NodeKey    `json:"gateKey"`
	Status              string     `json:"status"`
	Version             int64      `json:"version"`
	RequiredRole        string     `json:"requiredRole"`
	WaitingAgeMS        int64      `json:"waitingAgeMs,omitempty"`
	ExpiresInMS         *int64     `json:"expiresInMs,omitempty"`
	DecisionCode        string     `json:"decisionCode,omitempty"`
	DecidedAt           *time.Time `json:"decidedAt,omitempty"`
	HasDecisionArtifact bool       `json:"hasDecisionArtifact"`
	BudgetActivation    bool       `json:"budgetActivation,omitempty"`
}

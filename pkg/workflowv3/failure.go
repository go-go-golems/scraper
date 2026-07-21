package workflowv3

import (
	"fmt"
	"regexp"
)

var failureCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,63}$`)

var failureClasses = map[string]struct{}{
	"canceled":         {},
	"internal":         {},
	"malformed-output": {},
	"policy":           {},
	"provider-5xx":     {},
	"rate-limit":       {},
	"resource":         {},
	"timeout":          {},
	"transport":        {},
	"validation":       {},
}

func ValidateFailure(failure Failure) error {
	if _, ok := failureClasses[failure.Class]; !ok {
		return fmt.Errorf("unsupported failure class %q", failure.Class)
	}
	if !failureCodePattern.MatchString(failure.Code) {
		return fmt.Errorf("failure code %q must match %s", failure.Code, failureCodePattern)
	}
	if len(failure.Message) > 256 {
		return fmt.Errorf("failure message exceeds 256 bytes")
	}
	return nil
}

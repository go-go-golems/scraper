package workflowv3

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"time"
)

const (
	ExternalOperationOutcomeSucceeded = "succeeded"
	ExternalOperationOutcomeFailed    = "failed"
	ExternalOperationOutcomeCanceled  = "canceled"
	ExternalOperationOutcomeTimedOut  = "timed-out"
	ExternalOperationOutcomeUnknown   = "unknown"

	ExternalOperationAccountingActual       = "actual"
	ExternalOperationAccountingConservative = "conservative"
	ExternalOperationAccountingNone         = "none"
)

var (
	externalOperationNamePattern    = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[.:-][a-z0-9]+)*$`)
	externalOperationVersionPattern = regexp.MustCompile(`^v[1-9][0-9]{0,15}$`)
	externalOperationCounterPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	externalOperationUnitsPattern   = regexp.MustCompile(`^[a-z][a-z0-9_/-]{0,31}$`)
)

type ExternalOperationKind struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ExternalOperationCounterRole string

const (
	ExternalOperationCounterReservation ExternalOperationCounterRole = "reservation"
	ExternalOperationCounterUsage       ExternalOperationCounterRole = "usage"
	ExternalOperationCounterMeasure     ExternalOperationCounterRole = "measure"
)

// ExternalOperationCounterDescriptor declares one bounded scalar that a
// trusted host module may use in an external operation. A counter can have
// several roles: for example, requests is both reserved before a provider call
// and reported as actual usage when it completes.
type ExternalOperationCounterDescriptor struct {
	Name  string                         `json:"name"`
	Unit  string                         `json:"unit"`
	Roles []ExternalOperationCounterRole `json:"roles"`
}

// ExternalOperationDescriptor is immutable host authority. Its digest pins the
// operation kind, provider/tool authority identity, counter schema, and maximum
// number of calls that one Workflow attempt may admit.
type ExternalOperationDescriptor struct {
	Kind            ExternalOperationKind                `json:"kind"`
	AuthorityDigest string                               `json:"authorityDigest"`
	Counters        []ExternalOperationCounterDescriptor `json:"counters"`
	MaxPerAttempt   int                                  `json:"maxPerAttempt"`
	Digest          string                               `json:"digest"`
}

// ExternalOperationCounter is an integer value authorized by an operation
// descriptor. It intentionally has no free-form metadata or text payload.
type ExternalOperationCounter struct {
	Name  string `json:"name"`
	Units int64  `json:"units"`
}

// ExternalOperationSpec is supplied before the effect begins. Reservation
// counters consume portions of the already-reserved Workflow attempt budget;
// measures are non-budget scalar inputs such as chunk_count.
type ExternalOperationSpec struct {
	DescriptorDigest  string                     `json:"descriptorDigest"`
	CorrelationDigest string                     `json:"correlationDigest,omitempty"`
	Reservation       []ExternalOperationCounter `json:"reservation,omitempty"`
	Measures          []ExternalOperationCounter `json:"measures,omitempty"`
}

// ExternalOperationFailure excludes arbitrary provider error text. Its class
// and code use the existing closed Workflow failure vocabulary.
type ExternalOperationFailure struct {
	Class string `json:"class"`
	Code  string `json:"code"`
}

// ExternalOperationCompletion is the immutable result of one admitted effect.
// ProviderStartedAt and ElapsedMicros are data-plane evidence; Workflow attempt
// timing remains separate control-plane evidence.
type ExternalOperationCompletion struct {
	ProviderStartedAt time.Time                  `json:"providerStartedAt"`
	ElapsedMicros     int64                      `json:"elapsedMicros"`
	Outcome           string                     `json:"outcome"`
	Failure           *ExternalOperationFailure  `json:"failure,omitempty"`
	AccountingMode    string                     `json:"accountingMode"`
	Counters          []ExternalOperationCounter `json:"counters,omitempty"`
}

// ExternalOperationTicket authorizes one completion for one prior admission.
// CompletionKey is deliberately omitted from JSON and String so it cannot enter
// canonical evidence or ordinary logs.
type ExternalOperationTicket struct {
	OperationID   string `json:"operationId"`
	CompletionKey string `json:"-"`
}

func (t ExternalOperationTicket) String() string {
	return "external operation ticket " + t.OperationID
}

// ExternalOperation is the public joined read model for one durable admission
// and its optional immutable completion. A nil Completion means the effect was
// admitted but no terminal provider observation was durably recorded.
type ExternalOperation struct {
	OperationID       string                       `json:"operationId"`
	RunID             RunID                        `json:"runId"`
	NodeKey           NodeKey                      `json:"nodeKey"`
	Attempt           int                          `json:"attempt"`
	Ordinal           int                          `json:"ordinal"`
	Kind              ExternalOperationKind        `json:"kind"`
	DescriptorDigest  string                       `json:"descriptorDigest"`
	AuthorityDigest   string                       `json:"authorityDigest"`
	CorrelationDigest string                       `json:"correlationDigest,omitempty"`
	AdmittedAt        time.Time                    `json:"admittedAt"`
	Reservation       []ExternalOperationCounter   `json:"reservation,omitempty"`
	Measures          []ExternalOperationCounter   `json:"measures,omitempty"`
	Completion        *ExternalOperationCompletion `json:"completion,omitempty"`
}

// ExternalOperationRecorder is lease-scoped host authority. It is injected into
// trusted Go task modules only; it must never be exposed through workflow/task
// or ordinary JavaScript require().
type ExternalOperationRecorder interface {
	BeginExternalOperation(context.Context, ExternalOperationSpec) (ExternalOperationTicket, error)
	FinishExternalOperation(context.Context, ExternalOperationTicket, ExternalOperationCompletion) error
}

func ValidateExternalOperationDescriptor(descriptor ExternalOperationDescriptor) error {
	if !externalOperationNamePattern.MatchString(descriptor.Kind.Name) {
		return fmt.Errorf("external operation kind %q is invalid", descriptor.Kind.Name)
	}
	if !externalOperationVersionPattern.MatchString(descriptor.Kind.Version) {
		return fmt.Errorf("external operation version %q is invalid", descriptor.Kind.Version)
	}
	if err := validateSHA256Digest(descriptor.AuthorityDigest); err != nil {
		return fmt.Errorf("external operation authority digest: %w", err)
	}
	if descriptor.MaxPerAttempt < 1 || descriptor.MaxPerAttempt > 100_000 {
		return fmt.Errorf("external operation max per attempt %d is invalid", descriptor.MaxPerAttempt)
	}
	if len(descriptor.Counters) > 32 {
		return fmt.Errorf("external operation has too many counters")
	}
	previous := ""
	for _, counter := range descriptor.Counters {
		if !externalOperationCounterPattern.MatchString(counter.Name) {
			return fmt.Errorf("external operation counter %q is invalid", counter.Name)
		}
		if counter.Name <= previous {
			return fmt.Errorf("external operation counters must be strictly sorted and unique")
		}
		if !externalOperationUnitsPattern.MatchString(counter.Unit) {
			return fmt.Errorf("external operation counter %q unit %q is invalid", counter.Name, counter.Unit)
		}
		if len(counter.Roles) == 0 || len(counter.Roles) > 3 {
			return fmt.Errorf("external operation counter %q roles are invalid", counter.Name)
		}
		rolePrevious := ExternalOperationCounterRole("")
		for _, role := range counter.Roles {
			if !validExternalOperationCounterRole(role) || role <= rolePrevious {
				return fmt.Errorf("external operation counter %q roles must be strictly sorted, unique, and supported", counter.Name)
			}
			rolePrevious = role
		}
		previous = counter.Name
	}
	withoutDigest := cloneExternalOperationDescriptor(descriptor)
	withoutDigest.Digest = ""
	expected, err := Digest(withoutDigest)
	if err != nil {
		return err
	}
	if descriptor.Digest != expected {
		return fmt.Errorf("external operation descriptor digest mismatch: got %s want %s", descriptor.Digest, expected)
	}
	return nil
}

// NewExternalOperationDescriptor returns an immutable descriptor with its
// canonical digest. Callers must provide counters in canonical sorted order.
func NewExternalOperationDescriptor(descriptor ExternalOperationDescriptor) (ExternalOperationDescriptor, error) {
	descriptor = cloneExternalOperationDescriptor(descriptor)
	descriptor.Digest = ""
	if err := validateExternalOperationDescriptorWithoutDigest(descriptor); err != nil {
		return ExternalOperationDescriptor{}, err
	}
	digest, err := Digest(descriptor)
	if err != nil {
		return ExternalOperationDescriptor{}, err
	}
	descriptor.Digest = digest
	return descriptor, nil
}

func validateExternalOperationDescriptorWithoutDigest(descriptor ExternalOperationDescriptor) error {
	if !externalOperationNamePattern.MatchString(descriptor.Kind.Name) {
		return fmt.Errorf("external operation kind %q is invalid", descriptor.Kind.Name)
	}
	if !externalOperationVersionPattern.MatchString(descriptor.Kind.Version) {
		return fmt.Errorf("external operation version %q is invalid", descriptor.Kind.Version)
	}
	if err := validateSHA256Digest(descriptor.AuthorityDigest); err != nil {
		return fmt.Errorf("external operation authority digest: %w", err)
	}
	if descriptor.MaxPerAttempt < 1 || descriptor.MaxPerAttempt > 100_000 {
		return fmt.Errorf("external operation max per attempt %d is invalid", descriptor.MaxPerAttempt)
	}
	if len(descriptor.Counters) > 32 {
		return fmt.Errorf("external operation has too many counters")
	}
	previous := ""
	for _, counter := range descriptor.Counters {
		if !externalOperationCounterPattern.MatchString(counter.Name) {
			return fmt.Errorf("external operation counter %q is invalid", counter.Name)
		}
		if counter.Name <= previous {
			return fmt.Errorf("external operation counters must be strictly sorted and unique")
		}
		if !externalOperationUnitsPattern.MatchString(counter.Unit) {
			return fmt.Errorf("external operation counter %q unit %q is invalid", counter.Name, counter.Unit)
		}
		if len(counter.Roles) == 0 || len(counter.Roles) > 3 {
			return fmt.Errorf("external operation counter %q roles are invalid", counter.Name)
		}
		rolePrevious := ExternalOperationCounterRole("")
		for _, role := range counter.Roles {
			if !validExternalOperationCounterRole(role) || role <= rolePrevious {
				return fmt.Errorf("external operation counter %q roles must be strictly sorted, unique, and supported", counter.Name)
			}
			rolePrevious = role
		}
		previous = counter.Name
	}
	return nil
}

func ValidateExternalOperationSpec(descriptor ExternalOperationDescriptor, spec ExternalOperationSpec) error {
	if err := ValidateExternalOperationDescriptor(descriptor); err != nil {
		return err
	}
	if spec.DescriptorDigest != descriptor.Digest {
		return fmt.Errorf("external operation specification descriptor digest does not match")
	}
	if spec.CorrelationDigest != "" {
		if err := validateSHA256Digest(spec.CorrelationDigest); err != nil {
			return fmt.Errorf("external operation correlation digest: %w", err)
		}
	}
	if err := validateExternalOperationCounters(spec.Reservation, false, descriptor, ExternalOperationCounterReservation); err != nil {
		return fmt.Errorf("external operation reservation: %w", err)
	}
	if err := validateExternalOperationCounters(spec.Measures, true, descriptor, ExternalOperationCounterMeasure); err != nil {
		return fmt.Errorf("external operation measures: %w", err)
	}
	return nil
}

func ValidateExternalOperationCompletion(descriptor ExternalOperationDescriptor, completion ExternalOperationCompletion) error {
	if err := ValidateExternalOperationDescriptor(descriptor); err != nil {
		return err
	}
	if completion.ProviderStartedAt.IsZero() || completion.ProviderStartedAt.Location() != time.UTC {
		return fmt.Errorf("external operation provider start time must be non-zero UTC")
	}
	if completion.ElapsedMicros < 0 {
		return fmt.Errorf("external operation elapsed micros cannot be negative")
	}
	switch completion.Outcome {
	case ExternalOperationOutcomeSucceeded:
		if completion.Failure != nil {
			return fmt.Errorf("successful external operation cannot have failure")
		}
	case ExternalOperationOutcomeFailed, ExternalOperationOutcomeCanceled, ExternalOperationOutcomeTimedOut:
		if completion.Failure == nil {
			return fmt.Errorf("external operation outcome %q requires failure", completion.Outcome)
		}
		if err := ValidateFailure(Failure{Class: completion.Failure.Class, Code: completion.Failure.Code}); err != nil {
			return fmt.Errorf("external operation failure: %w", err)
		}
	case ExternalOperationOutcomeUnknown:
		if completion.Failure != nil {
			return fmt.Errorf("unknown external operation cannot have failure")
		}
	default:
		return fmt.Errorf("external operation outcome %q is invalid", completion.Outcome)
	}
	switch completion.AccountingMode {
	case ExternalOperationAccountingActual:
	case ExternalOperationAccountingConservative:
	case ExternalOperationAccountingNone:
		if len(completion.Counters) != 0 {
			return fmt.Errorf("unaccounted external operation cannot report counters")
		}
	default:
		return fmt.Errorf("external operation accounting mode %q is invalid", completion.AccountingMode)
	}
	if completion.Outcome == ExternalOperationOutcomeUnknown && completion.AccountingMode != ExternalOperationAccountingConservative {
		return fmt.Errorf("unknown external operation requires conservative accounting")
	}
	if err := validateExternalOperationCounters(completion.Counters, true, descriptor, ExternalOperationCounterUsage); err != nil {
		return fmt.Errorf("external operation completion counters: %w", err)
	}
	return nil
}

func validateExternalOperationCounters(counters []ExternalOperationCounter, allowZero bool, descriptor ExternalOperationDescriptor, role ExternalOperationCounterRole) error {
	known := make(map[string]ExternalOperationCounterDescriptor, len(descriptor.Counters))
	for _, counter := range descriptor.Counters {
		known[counter.Name] = counter
	}
	previous := ""
	for _, counter := range counters {
		if !externalOperationCounterPattern.MatchString(counter.Name) {
			return fmt.Errorf("counter %q is invalid", counter.Name)
		}
		if counter.Name <= previous {
			return fmt.Errorf("counters must be strictly sorted and unique")
		}
		if counter.Units < 0 || (!allowZero && counter.Units == 0) {
			return fmt.Errorf("counter %q has invalid units %d", counter.Name, counter.Units)
		}
		described, ok := known[counter.Name]
		if !ok || !externalOperationCounterHasRole(described, role) {
			return fmt.Errorf("counter %q is not allowed for %s", counter.Name, role)
		}
		previous = counter.Name
	}
	return nil
}

func validExternalOperationCounterRole(role ExternalOperationCounterRole) bool {
	switch role {
	case ExternalOperationCounterReservation, ExternalOperationCounterUsage, ExternalOperationCounterMeasure:
		return true
	default:
		return false
	}
}

func externalOperationCounterHasRole(counter ExternalOperationCounterDescriptor, wanted ExternalOperationCounterRole) bool {
	index := sort.Search(len(counter.Roles), func(index int) bool { return counter.Roles[index] >= wanted })
	return index < len(counter.Roles) && counter.Roles[index] == wanted
}

func cloneExternalOperationDescriptor(descriptor ExternalOperationDescriptor) ExternalOperationDescriptor {
	ret := descriptor
	ret.Counters = make([]ExternalOperationCounterDescriptor, len(descriptor.Counters))
	for index, counter := range descriptor.Counters {
		ret.Counters[index] = counter
		ret.Counters[index].Roles = append([]ExternalOperationCounterRole(nil), counter.Roles...)
	}
	return ret
}

func CloneExternalOperationDescriptors(descriptors []ExternalOperationDescriptor) []ExternalOperationDescriptor {
	ret := make([]ExternalOperationDescriptor, len(descriptors))
	for index, descriptor := range descriptors {
		ret[index] = cloneExternalOperationDescriptor(descriptor)
	}
	return ret
}

func CloneExternalOperationCounters(counters []ExternalOperationCounter) []ExternalOperationCounter {
	return append([]ExternalOperationCounter(nil), counters...)
}

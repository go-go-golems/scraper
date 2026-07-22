package workflowv3

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testExternalOperationDescriptor(t *testing.T) ExternalOperationDescriptor {
	t.Helper()
	descriptor, err := NewExternalOperationDescriptor(ExternalOperationDescriptor{
		Kind:            ExternalOperationKind{Name: "provider.generate", Version: "v1"},
		AuthorityDigest: "sha256:" + strings.Repeat("a", 64),
		Counters: []ExternalOperationCounterDescriptor{
			{Name: "chunk_count", Unit: "items", Roles: []ExternalOperationCounterRole{ExternalOperationCounterMeasure}},
			{Name: "cost_microunits", Unit: "microunits", Roles: []ExternalOperationCounterRole{ExternalOperationCounterReservation, ExternalOperationCounterUsage}},
			{Name: "input_tokens", Unit: "tokens", Roles: []ExternalOperationCounterRole{ExternalOperationCounterReservation, ExternalOperationCounterUsage}},
			{Name: "requests", Unit: "requests", Roles: []ExternalOperationCounterRole{ExternalOperationCounterReservation, ExternalOperationCounterUsage}},
		},
		MaxPerAttempt: 4,
	})
	require.NoError(t, err)
	return descriptor
}

func TestExternalOperationDescriptorCanonicalIdentityAndClone(t *testing.T) {
	descriptor := testExternalOperationDescriptor(t)
	require.NoError(t, ValidateExternalOperationDescriptor(descriptor))

	clone := CloneExternalOperationDescriptors([]ExternalOperationDescriptor{descriptor})
	clone[0].Counters[0].Roles[0] = ExternalOperationCounterUsage
	require.Equal(t, ExternalOperationCounterMeasure, descriptor.Counters[0].Roles[0])

	rebuilt, err := NewExternalOperationDescriptor(descriptor)
	require.NoError(t, err)
	require.Equal(t, descriptor.Digest, rebuilt.Digest)
}

func TestExternalOperationDescriptorRejectsMalformedPolicy(t *testing.T) {
	base := testExternalOperationDescriptor(t)
	cases := []struct {
		name   string
		mutate func(*ExternalOperationDescriptor)
	}{
		{name: "invalid kind", mutate: func(value *ExternalOperationDescriptor) { value.Kind.Name = "Provider" }},
		{name: "invalid version", mutate: func(value *ExternalOperationDescriptor) { value.Kind.Version = "1" }},
		{name: "invalid authority", mutate: func(value *ExternalOperationDescriptor) { value.AuthorityDigest = "sha256:no" }},
		{name: "zero maximum", mutate: func(value *ExternalOperationDescriptor) { value.MaxPerAttempt = 0 }},
		{name: "unsorted counters", mutate: func(value *ExternalOperationDescriptor) {
			value.Counters[0], value.Counters[1] = value.Counters[1], value.Counters[0]
		}},
		{name: "unsorted roles", mutate: func(value *ExternalOperationDescriptor) {
			value.Counters[1].Roles = []ExternalOperationCounterRole{ExternalOperationCounterUsage, ExternalOperationCounterReservation}
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			candidate := cloneExternalOperationDescriptor(base)
			test.mutate(&candidate)
			candidate.Digest = ""
			_, err := NewExternalOperationDescriptor(candidate)
			require.Error(t, err)
		})
	}

	base.Digest = "sha256:" + strings.Repeat("b", 64)
	require.Error(t, ValidateExternalOperationDescriptor(base))
}

func TestExternalOperationSpecRequiresDescriptorAuthorizedCounters(t *testing.T) {
	descriptor := testExternalOperationDescriptor(t)
	valid := ExternalOperationSpec{
		DescriptorDigest:  descriptor.Digest,
		CorrelationDigest: "sha256:" + strings.Repeat("c", 64),
		Reservation: []ExternalOperationCounter{
			{Name: "cost_microunits", Units: 10},
			{Name: "input_tokens", Units: 20},
			{Name: "requests", Units: 1},
		},
		Measures: []ExternalOperationCounter{{Name: "chunk_count", Units: 2}},
	}
	require.NoError(t, ValidateExternalOperationSpec(descriptor, valid))

	for _, test := range []struct {
		name   string
		mutate func(*ExternalOperationSpec)
	}{
		{name: "wrong descriptor", mutate: func(value *ExternalOperationSpec) { value.DescriptorDigest = "sha256:" + strings.Repeat("d", 64) }},
		{name: "unknown reservation", mutate: func(value *ExternalOperationSpec) {
			value.Reservation = []ExternalOperationCounter{{Name: "chunk_count", Units: 1}}
		}},
		{name: "unknown measure", mutate: func(value *ExternalOperationSpec) {
			value.Measures = []ExternalOperationCounter{{Name: "requests", Units: 1}}
		}},
		{name: "unsorted reservation", mutate: func(value *ExternalOperationSpec) {
			value.Reservation[0], value.Reservation[1] = value.Reservation[1], value.Reservation[0]
		}},
		{name: "zero reservation", mutate: func(value *ExternalOperationSpec) { value.Reservation[0].Units = 0 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := valid
			candidate.Reservation = CloneExternalOperationCounters(valid.Reservation)
			candidate.Measures = CloneExternalOperationCounters(valid.Measures)
			test.mutate(&candidate)
			require.Error(t, ValidateExternalOperationSpec(descriptor, candidate))
		})
	}
}

func TestExternalOperationCompletionPreservesClosedFailureSemantics(t *testing.T) {
	descriptor := testExternalOperationDescriptor(t)
	started := time.Date(2026, 7, 22, 23, 0, 0, 0, time.UTC)
	success := ExternalOperationCompletion{
		ProviderStartedAt: started, ElapsedMicros: 12, Outcome: ExternalOperationOutcomeSucceeded,
		AccountingMode: ExternalOperationAccountingActual,
		Counters:       []ExternalOperationCounter{{Name: "cost_microunits", Units: 3}, {Name: "input_tokens", Units: 2}, {Name: "requests", Units: 1}},
	}
	require.NoError(t, ValidateExternalOperationCompletion(descriptor, success))

	failure := ExternalOperationCompletion{
		ProviderStartedAt: started, ElapsedMicros: 14, Outcome: ExternalOperationOutcomeFailed,
		Failure:        &ExternalOperationFailure{Class: "timeout", Code: "PROVIDER_TIMEOUT"},
		AccountingMode: ExternalOperationAccountingConservative,
	}
	require.NoError(t, ValidateExternalOperationCompletion(descriptor, failure))

	unknown := ExternalOperationCompletion{
		ProviderStartedAt: started, ElapsedMicros: 0, Outcome: ExternalOperationOutcomeUnknown,
		AccountingMode: ExternalOperationAccountingConservative,
	}
	require.NoError(t, ValidateExternalOperationCompletion(descriptor, unknown))

	for _, test := range []struct {
		name  string
		value ExternalOperationCompletion
	}{
		{name: "success failure", value: ExternalOperationCompletion{ProviderStartedAt: started, Outcome: ExternalOperationOutcomeSucceeded, Failure: &ExternalOperationFailure{Class: "timeout", Code: "PROVIDER_TIMEOUT"}, AccountingMode: ExternalOperationAccountingActual}},
		{name: "failed without failure", value: ExternalOperationCompletion{ProviderStartedAt: started, Outcome: ExternalOperationOutcomeFailed, AccountingMode: ExternalOperationAccountingConservative}},
		{name: "unknown actual", value: ExternalOperationCompletion{ProviderStartedAt: started, Outcome: ExternalOperationOutcomeUnknown, AccountingMode: ExternalOperationAccountingActual}},
		{name: "non utc", value: ExternalOperationCompletion{ProviderStartedAt: started.In(time.FixedZone("test", 3600)), Outcome: ExternalOperationOutcomeSucceeded, AccountingMode: ExternalOperationAccountingActual}},
		{name: "negative elapsed", value: ExternalOperationCompletion{ProviderStartedAt: started, ElapsedMicros: -1, Outcome: ExternalOperationOutcomeSucceeded, AccountingMode: ExternalOperationAccountingActual}},
		{name: "unknown counter", value: ExternalOperationCompletion{ProviderStartedAt: started, Outcome: ExternalOperationOutcomeSucceeded, AccountingMode: ExternalOperationAccountingActual, Counters: []ExternalOperationCounter{{Name: "chunk_count", Units: 1}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Error(t, ValidateExternalOperationCompletion(descriptor, test.value))
		})
	}
}

func TestExternalOperationTicketRedactsCompletionKey(t *testing.T) {
	ticket := ExternalOperationTicket{OperationID: "op-01", CompletionKey: "must-not-persist"}
	body, err := json.Marshal(ticket)
	require.NoError(t, err)
	require.JSONEq(t, `{"operationId":"op-01"}`, string(body))
	require.NotContains(t, ticket.String(), ticket.CompletionKey)
}

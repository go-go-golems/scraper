package workflowv3

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFailureUsesStableVocabulary(t *testing.T) {
	require.NoError(t, ValidateFailure(Failure{
		Class: "validation", Code: "CUSTOMER_DUPLICATE_ID",
		Retryable: false, Message: "task reported CUSTOMER_DUPLICATE_ID",
	}))
	require.Error(t, ValidateFailure(Failure{Class: "whatever", Code: "VALID_CODE"}))
	require.Error(t, ValidateFailure(Failure{Class: "validation", Code: "free form"}))
}

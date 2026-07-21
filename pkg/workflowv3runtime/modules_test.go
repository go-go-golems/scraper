package workflowv3runtime

import (
	"testing"
	"time"

	fetchmod "github.com/go-go-golems/go-go-goja/modules/fetch"
	"github.com/stretchr/testify/require"
)

func TestTaskModuleRegistryRejectsUnsafePublicFetchProfiles(t *testing.T) {
	base := fetchmod.Policy{
		AllowedOrigins:   []string{"https://example.test"},
		Timeout:          time.Second,
		MaxResponseBytes: 1024,
		Credentials: fetchmod.CredentialPolicy{
			AllowEnv: false, AllowFiles: false,
		},
	}
	tests := []struct {
		name   string
		policy fetchmod.Policy
		error  string
	}{
		{name: "empty allowlist", policy: func() fetchmod.Policy {
			value := base
			value.AllowedOrigins = nil
			return value
		}(), error: "requires allowed origins"},
		{name: "wildcard", policy: func() fetchmod.Policy {
			value := base
			value.AllowedOrigins = []string{"*"}
			return value
		}(), error: "wildcard origin"},
		{name: "environment credentials", policy: func() fetchmod.Policy {
			value := base
			value.Credentials.AllowEnv = true
			return value
		}(), error: "credential sources must be disabled"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewTaskModuleRegistry(FetchModule("fetch:public", test.policy, nil))
			require.ErrorContains(t, err, test.error)
		})
	}
}

func TestTaskModuleRegistryRejectsNilDatabaseHandle(t *testing.T) {
	_, err := NewTaskModuleRegistry(DatabaseModule("db:sync", nil))
	require.ErrorContains(t, err, "preconfigured database handle is required")
}

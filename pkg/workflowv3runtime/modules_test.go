package workflowv3runtime

import (
	"strings"
	"testing"
	"time"

	fetchmod "github.com/go-go-golems/go-go-goja/modules/fetch"
	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
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

func TestTaskModuleRegistryScopesClonedOperationDescriptors(t *testing.T) {
	descriptor, err := workflowv3.NewExternalOperationDescriptor(workflowv3.ExternalOperationDescriptor{Kind: workflowv3.ExternalOperationKind{Name: "provider.generate", Version: "v1"}, AuthorityDigest: "sha256:" + strings.Repeat("a", 64), MaxPerAttempt: 1})
	require.NoError(t, err)
	registry, err := NewTaskModuleRegistry(TaskModuleFactory{Alias: "provider:trusted", Operations: []workflowv3.ExternalOperationDescriptor{descriptor}, Build: func(TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error) {
		return gggengine.NativeModuleRegistrar{}, nil
	}})
	require.NoError(t, err)
	operations, err := registry.OperationDescriptors([]string{"provider:trusted"})
	require.NoError(t, err)
	require.Equal(t, []workflowv3.ExternalOperationDescriptor{descriptor}, operations)
	operations[0].AuthorityDigest = "sha256:" + strings.Repeat("b", 64)
	fresh, err := registry.OperationDescriptors([]string{"provider:trusted"})
	require.NoError(t, err)
	require.Equal(t, descriptor.AuthorityDigest, fresh[0].AuthorityDigest)
}

func TestTaskModuleRegistryRejectsNilDatabaseHandle(t *testing.T) {
	_, err := NewTaskModuleRegistry(DatabaseModule("db:sync", nil))
	require.ErrorContains(t, err, "preconfigured database handle is required")
}

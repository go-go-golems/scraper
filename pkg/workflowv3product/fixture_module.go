package workflowv3product

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/scraper/pkg/taskpackages/researchfixture"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3runtime"
)

var researchFixtureOperationDescriptor = newResearchFixtureOperationDescriptor()

func newResearchFixtureOperationDescriptor() workflowv3.ExternalOperationDescriptor {
	authority := sha256.Sum256([]byte("scraper/research-runner-fixture-operation/v1"))
	descriptor, err := workflowv3.NewExternalOperationDescriptor(workflowv3.ExternalOperationDescriptor{
		Kind:            workflowv3.ExternalOperationKind{Name: "fixture.operation", Version: "v1"},
		AuthorityDigest: "sha256:" + hex.EncodeToString(authority[:]),
		MaxPerAttempt:   1,
	})
	if err != nil {
		panic(err)
	}
	return descriptor
}

func fixtureOperationModule() workflowv3runtime.TaskModuleFactory {
	return workflowv3runtime.TaskModuleFactory{
		Alias:      researchfixture.OperationModuleAlias,
		Operations: []workflowv3.ExternalOperationDescriptor{researchFixtureOperationDescriptor},
		Build: func(moduleContext workflowv3runtime.TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error) {
			if moduleContext.ExternalOperations == nil {
				return nil, fmt.Errorf("fixture operation recorder is required")
			}
			loader := func(vm *goja.Runtime, moduleObject *goja.Object) {
				exports := moduleObject.Get("exports").ToObject(vm)
				if err := exports.Set("invoke", func(goja.FunctionCall) goja.Value {
					started := time.Now().UTC()
					ticket, err := moduleContext.ExternalOperations.BeginExternalOperation(moduleContext.Context, workflowv3.ExternalOperationSpec{
						DescriptorDigest: researchFixtureOperationDescriptor.Digest,
					})
					if err != nil {
						panic(vm.NewGoError(err))
					}
					succeeded := moduleContext.Request.Attempt > 1
					completion := workflowv3.ExternalOperationCompletion{
						ProviderStartedAt: started, ElapsedMicros: 1_000,
						Outcome:        workflowv3.ExternalOperationOutcomeSucceeded,
						AccountingMode: workflowv3.ExternalOperationAccountingNone,
					}
					if !succeeded {
						completion.Outcome = workflowv3.ExternalOperationOutcomeFailed
						completion.Failure = &workflowv3.ExternalOperationFailure{Class: "transport", Code: "FIXTURE_TRANSIENT"}
					}
					if err := moduleContext.ExternalOperations.FinishExternalOperation(moduleContext.Context, ticket, completion); err != nil {
						panic(vm.NewGoError(err))
					}
					return vm.ToValue(map[string]bool{"succeeded": succeeded})
				}); err != nil {
					panic(vm.NewGoError(err))
				}
			}
			return gggengine.NativeModuleRegistrar{
				ModuleID: "workflowv3:fixture-operation", ModuleName: researchfixture.OperationModuleAlias, Loader: loader,
			}, nil
		},
	}
}

package workflowv3runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
)

type Engine struct {
	Store                       *workflowv3sqlite.Store
	Registry                    workflowv3.RegistryResolver
	Artifacts                   workflowv3.ArtifactStore
	Modules                     *TaskModuleRegistry
	Isolation                   IsolatedTaskExecutor
	LeaseDuration               time.Duration
	RegistryQuarantineThreshold int
	Now                         func() time.Time
}

func (e *Engine) Submit(ctx context.Context, runID workflowv3.RunID, plan workflowv3.WorkflowPlan, inputs map[string]workflowv3.ArtifactRef) error {
	if err := e.validate(); err != nil {
		return err
	}
	return e.Store.CreateRun(ctx, runID, plan, inputs, e.now())
}

func (e *Engine) ExpandOne(ctx context.Context) (bool, error) {
	if err := e.validate(); err != nil {
		return false, err
	}
	candidates, err := e.Store.ExpansionCandidates(ctx)
	if err != nil {
		return false, err
	}
	for _, candidate := range candidates {
		body, err := workflowv3.ReadArtifact(ctx, e.Artifacts, candidate.Source)
		if err != nil {
			return false, fmt.Errorf("read map manifest %s/%s: %w", candidate.RunID, candidate.MapKey, err)
		}
		manifest, err := workflowv3.DecodeItemManifest(body)
		if err != nil {
			return false, fmt.Errorf("decode map manifest %s/%s: %w", candidate.RunID, candidate.MapKey, err)
		}
		page, err := e.Store.ExpandNextPage(
			ctx, candidate.RunID, candidate.MapKey, candidate.Source, manifest, e.now(),
		)
		if err != nil {
			return false, err
		}
		if page != nil {
			return true, nil
		}
	}
	chained, err := e.Store.ChainedExpansionCandidate(ctx)
	if err != nil {
		return false, err
	}
	if chained == nil {
		return false, nil
	}
	body, err := workflowv3.EncodeItemManifest(chained.Manifest)
	if err != nil {
		return false, err
	}
	ref, err := e.Artifacts.Put(ctx, workflowv3.ItemManifestSchemaV1, "application/json", body)
	if err != nil {
		return false, fmt.Errorf("publish chained map prefix: %w", err)
	}
	page, err := e.Store.ExpandNextChainedPage(ctx, chained.RunID, chained.MapKey, ref, chained.Manifest, chained.Final, e.now())
	if err != nil {
		return false, err
	}
	return page != nil, nil
}

func (e *Engine) FinalizeOneMap(ctx context.Context) (bool, error) {
	if err := e.validate(); err != nil {
		return false, err
	}
	candidates, err := e.Store.ExpansionFinalizationCandidates(ctx)
	if err != nil {
		return false, err
	}
	if len(candidates) == 0 {
		return false, nil
	}
	candidate := candidates[0]
	manifest, err := e.Store.MapOutputManifest(ctx, candidate.RunID, candidate.MapKey)
	if err != nil {
		return false, err
	}
	body, err := workflowv3.EncodeItemManifest(manifest)
	if err != nil {
		return false, err
	}
	ref, err := e.Artifacts.Put(ctx, workflowv3.ItemManifestSchemaV1, "application/json", body)
	if err != nil {
		return false, fmt.Errorf("publish map output artifact: %w", err)
	}
	if err := e.Store.PublishMapOutput(ctx, candidate.RunID, candidate.MapKey, ref, e.now()); err != nil {
		return false, err
	}
	return true, nil
}

func (e *Engine) ReduceOne(ctx context.Context) (bool, error) {
	if err := e.validate(); err != nil {
		return false, err
	}
	candidates, err := e.Store.ReductionCandidates(ctx)
	if err != nil {
		return false, err
	}
	for _, candidate := range candidates {
		if candidate.Status == "pending" {
			body, err := workflowv3.ReadArtifact(ctx, e.Artifacts, candidate.Source)
			if err != nil {
				return e.failReduction(ctx, candidate, "validation", "REDUCTION_SOURCE_ARTIFACT")
			}
			manifest, err := workflowv3.DecodeItemManifest(body)
			if err != nil {
				return e.failReduction(ctx, candidate, "validation", "REDUCTION_SOURCE_MANIFEST")
			}
			if len(manifest.Items) == 0 {
				return e.failReduction(ctx, candidate, "validation", "REDUCTION_SOURCE_EMPTY")
			}
			if len(manifest.Items) == 1 {
				if err := e.Store.PublishReductionRoot(
					ctx, candidate.RunID, candidate.ReduceKey, candidate.Source,
					1, manifest.Items[0].Value, e.now(),
				); err != nil {
					return false, err
				}
				return true, nil
			}
			if err := e.materializeReductionPartitions(ctx, candidate, manifest.Items, 0, len(manifest.Items)); err != nil {
				return false, err
			}
			return true, nil
		}
		members, _, nextLevel, ready, err := e.Store.ReductionLevelMembers(
			ctx, candidate.RunID, candidate.ReduceKey,
		)
		if err != nil {
			return false, err
		}
		if !ready {
			continue
		}
		if len(members) == 1 {
			if err := e.Store.PublishReductionRoot(
				ctx, candidate.RunID, candidate.ReduceKey, candidate.Source,
				candidate.SourceItems, members[0].Value, e.now(),
			); err != nil {
				return false, err
			}
			return true, nil
		}
		if nextLevel >= candidate.MaxLevels {
			return e.failReduction(ctx, candidate, "configuration", "REDUCTION_LEVEL_LIMIT")
		}
		if err := e.materializeReductionPartitions(
			ctx, candidate, members, nextLevel, candidate.SourceItems,
		); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (e *Engine) failReduction(
	ctx context.Context,
	candidate workflowv3sqlite.ReductionCandidate,
	class, code string,
) (bool, error) {
	failure := workflowv3.Failure{
		Class: class, Code: code, Retryable: false,
		Message: "reduction failed validation",
	}
	if err := e.Store.FailReduction(
		ctx, candidate.RunID, candidate.ReduceKey, failure, e.now(),
	); err != nil {
		return false, err
	}
	return true, nil
}

func (e *Engine) materializeReductionPartitions(
	ctx context.Context,
	candidate workflowv3sqlite.ReductionCandidate,
	members []workflowv3.ManifestItem,
	level, sourceItems int,
) error {
	partitions := make([]workflowv3sqlite.ReductionPartitionInput, 0,
		(len(members)+candidate.FanIn-1)/candidate.FanIn)
	for first, ordinal := 0, 0; first < len(members); first, ordinal = first+candidate.FanIn, ordinal+1 {
		last := first + candidate.FanIn
		if last > len(members) {
			last = len(members)
		}
		partition, err := workflowv3.NewReductionPartition(
			candidate.ReduceKey, candidate.Source.Digest, members[first].Value.Schema,
			level, ordinal, candidate.FanIn, members[first:last],
		)
		if err != nil {
			return err
		}
		body, err := workflowv3.EncodeReductionPartition(partition, candidate.FanIn)
		if err != nil {
			return err
		}
		ref, err := e.Artifacts.Put(
			ctx, workflowv3.ReductionPartitionSchemaV1, "application/json", body,
		)
		if err != nil {
			return err
		}
		partitions = append(partitions, workflowv3sqlite.ReductionPartitionInput{
			Partition: partition, Ref: ref,
		})
	}
	return e.Store.MaterializeReductionLevel(
		ctx, candidate.RunID, candidate.ReduceKey, candidate.Source,
		sourceItems, level, partitions, e.now(),
	)
}

func (e *Engine) MaintainGates(ctx context.Context) (bool, error) {
	if err := e.validate(); err != nil {
		return false, err
	}
	expired, err := e.Store.ExpireDueGates(ctx, e.now())
	if err != nil {
		return false, err
	}
	advanced, err := e.Store.AdvanceOneGate(ctx, e.now())
	return expired > 0 || advanced, err
}

func (e *Engine) RunOne(ctx context.Context) (bool, error) {
	if err := e.validate(); err != nil {
		return false, err
	}
	gates, err := e.MaintainGates(ctx)
	if err != nil {
		return false, err
	}
	expanded, err := e.ExpandOne(ctx)
	if err != nil {
		return false, err
	}
	finalized, err := e.FinalizeOneMap(ctx)
	if err != nil {
		return false, err
	}
	reduced, err := e.ReduceOne(ctx)
	if err != nil {
		return false, err
	}
	lease, err := e.Store.LeaseNext(ctx, e.Registry, e.now(), e.leaseDuration())
	if err != nil || lease == nil {
		return gates || expanded || finalized || reduced, err
	}
	return true, e.ExecuteLease(ctx, *lease)
}

func (e *Engine) ExecuteLease(ctx context.Context, lease workflowv3.Lease) error {
	if err := e.validate(); err != nil {
		return err
	}
	if lease.ReleaseGeneration != nil {
		defer lease.ReleaseGeneration()
	}
	registered := lease.RegisteredTask
	if registered.Bundle == nil {
		var err error
		registered, err = e.Registry.ResolveNode(lease.PlanNode)
		if err != nil {
			return fmt.Errorf("resolve leased implementation: %w", err)
		}
	}
	inputs, err := e.Store.ResolveInputs(ctx, lease)
	if err != nil {
		failure := workflowv3.Failure{
			Class: "internal", Code: "WORKFLOW_INPUT_RESOLUTION",
			Retryable: false, Message: "task input resolution failed",
		}
		if persistErr := e.Store.FailWithoutCharge(ctx, lease, failure, e.now()); persistErr != nil {
			return fmt.Errorf("resolve inputs: %v; persist failure: %w", err, persistErr)
		}
		return &AttemptExecutionError{Err: fmt.Errorf("resolve inputs: %w", err)}
	}
	descriptors, err := e.Modules.OperationDescriptors(registered.Spec.Modules)
	if err != nil {
		failure := workflowv3.Failure{Class: "configuration", Code: "WORKFLOW_OPERATION_DESCRIPTOR", Retryable: false, Message: "task operation descriptor is unavailable"}
		if persistErr := e.Store.FailWithoutCharge(ctx, lease, failure, e.now()); persistErr != nil {
			return fmt.Errorf("resolve operation descriptors: %v; persist failure: %w", err, persistErr)
		}
		return &AttemptExecutionError{Err: fmt.Errorf("resolve operation descriptors: %w", err)}
	}
	recorder, err := e.Store.ExternalOperationRecorder(lease, descriptors)
	if err != nil {
		failure := workflowv3.Failure{Class: "internal", Code: "WORKFLOW_OPERATION_RECORDER", Retryable: true, Message: "task operation recorder construction failed"}
		if persistErr := e.Store.InfrastructureFail(ctx, lease, failure, e.now()); persistErr != nil {
			return fmt.Errorf("construct operation recorder: %v; persist failure: %w", err, persistErr)
		}
		return &AttemptExecutionError{Err: fmt.Errorf("construct operation recorder: %w", err)}
	}
	attemptCtx, cancelAttempt := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	go e.watchLease(ctx, lease, cancelAttempt, watchDone)
	defer func() {
		close(watchDone)
		cancelAttempt()
	}()
	taskRequest := TaskRequest{
		RunID: lease.RunID, NodeKey: lease.NodeKey, Attempt: lease.Attempt,
		Task: registered, Inputs: inputs, Artifacts: e.Artifacts,
		Modules: e.Modules, ExternalOperations: recorder,
	}
	isolation := workflowv3.EffectivePlanIsolation(lease.PlanNode.Isolation)
	var result TaskResult
	switch isolation.Effective.Class {
	case workflowv3.IsolationInProcessTrusted:
		result, err = RunTask(attemptCtx, taskRequest)
	case workflowv3.IsolationSubprocessRestricted:
		if e.Isolation == nil {
			err = &IsolationConstructionError{Err: fmt.Errorf("restricted isolation executor is not configured")}
		} else {
			result, err = e.Isolation.Execute(attemptCtx, taskRequest, isolation)
		}
	default:
		err = &IsolationConstructionError{Err: fmt.Errorf("unsupported isolation class %q", isolation.Effective.Class)}
	}
	if err != nil {
		var preparation *TaskPreparationError
		if errors.As(err, &preparation) {
			failure := workflowv3.Failure{
				Class: "internal", Code: "WORKFLOW_TASK_PREPARATION",
				Retryable: false, Message: "task preparation failed",
			}
			if persistErr := e.Store.FailWithoutCharge(ctx, lease, failure, e.now()); persistErr != nil {
				return fmt.Errorf("prepare task: %v; persist failure: %w", err, persistErr)
			}
			return &AttemptExecutionError{Err: fmt.Errorf("prepare task: %w", err)}
		}
		var isolationConstruction *IsolationConstructionError
		if errors.As(err, &isolationConstruction) {
			failure := workflowv3.Failure{
				Class: "configuration", Code: "ISOLATION_PROFILE_UNAVAILABLE",
				Retryable: true, Message: "task isolation profile is unavailable",
			}
			if persistErr := e.Store.InfrastructureFail(ctx, lease, failure, e.now()); persistErr != nil {
				return fmt.Errorf("construct task isolation: %v; persist failure: %w", err, persistErr)
			}
			return &AttemptExecutionError{Err: fmt.Errorf("construct task isolation: %w", err)}
		}
		var construction *RuntimeConstructionError
		if errors.As(err, &construction) {
			if recorder, ok := e.Registry.(interface {
				RecordConstructionFailure(string, string, int) (bool, error)
			}); ok {
				if _, recordErr := recorder.RecordConstructionFailure(
					lease.RegistryGeneration, "TASK_RUNTIME_CONSTRUCTION",
					e.registryQuarantineThreshold(),
				); recordErr != nil {
					return fmt.Errorf("record runtime construction failure: %w", recordErr)
				}
				failure := workflowv3.Failure{
					Class: "configuration", Code: "TASK_RUNTIME_CONSTRUCTION",
					Retryable: true, Message: "task runtime construction failed",
				}
				if persistErr := e.Store.InfrastructureFail(ctx, lease, failure, e.now()); persistErr != nil {
					return fmt.Errorf("construct task runtime: %v; persist failure: %w", err, persistErr)
				}
				return &AttemptExecutionError{Err: fmt.Errorf("construct task runtime: %w", err)}
			}
		}
		failure := workflowv3.Failure{
			Class: "internal", Code: "WORKFLOW_TASK_EXECUTION",
			Retryable: false, Message: "task execution failed",
		}
		var taskFailure *TaskFailureError
		if errors.As(err, &taskFailure) {
			failure = taskFailure.Failure
			failure.Message = "task reported " + failure.Code
		}
		var persistErr error
		if taskFailure != nil && len(taskFailure.Usage) > 0 {
			persistErr = e.Store.FailWithUsage(ctx, lease, failure, taskFailure.Usage, e.now())
		} else {
			persistErr = e.Store.Fail(ctx, lease, failure, e.now())
		}
		if errors.Is(persistErr, workflowv3sqlite.ErrBudgetUsageInvalid) ||
			errors.Is(persistErr, workflowv3sqlite.ErrBudgetUsageExceedsReservation) {
			failure = workflowv3.Failure{
				Class: "budget", Code: "BUDGET_USAGE_INVALID", Retryable: false,
				Message: "failed task reported invalid usage",
			}
			persistErr = e.Store.Fail(ctx, lease, failure, e.now())
		}
		if persistErr != nil {
			return fmt.Errorf("execute task: %v; persist failure: %w", err, persistErr)
		}
		return &AttemptExecutionError{Err: fmt.Errorf("execute task: %w", err)}
	}
	if err := e.Store.CompleteWithUsage(ctx, lease, result.Outputs, result.Usage, e.now()); err != nil {
		var failure workflowv3.Failure
		switch {
		case errors.Is(err, workflowv3sqlite.ErrBudgetUsageExceedsReservation):
			failure = workflowv3.Failure{Class: "budget", Code: "BUDGET_USAGE_EXCEEDS_RESERVATION", Retryable: false, Message: "task usage exceeded reservation"}
		case errors.Is(err, workflowv3sqlite.ErrBudgetUsageInvalid):
			failure = workflowv3.Failure{Class: "budget", Code: "BUDGET_USAGE_INVALID", Retryable: false, Message: "task usage evidence was invalid"}
		default:
			return fmt.Errorf("complete task: %w", err)
		}
		if persistErr := e.Store.Fail(ctx, lease, failure, e.now()); persistErr != nil {
			return fmt.Errorf("complete task: %v; persist budget failure: %w", err, persistErr)
		}
		return &AttemptExecutionError{Err: fmt.Errorf("complete task: %w", err)}
	}
	return nil
}

func (e *Engine) RunUntilIdle(ctx context.Context) error {
	for {
		ran, err := e.RunOne(ctx)
		if err != nil {
			return err
		}
		if !ran {
			return nil
		}
	}
}

func (e *Engine) watchLease(
	ctx context.Context,
	lease workflowv3.Lease,
	cancel context.CancelFunc,
	done <-chan struct{},
) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	expiresAt := lease.ExpiresAt
	renewAfter := expiresAt.Add(-e.leaseDuration() / 2)
	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case <-done:
			return
		case <-ticker.C:
			now := e.now()
			if !now.Before(renewAfter) {
				newExpiry := now.Add(e.leaseDuration())
				renewed, err := e.Store.RenewLease(ctx, lease, now, newExpiry)
				if err == nil && renewed {
					expiresAt = newExpiry
					renewAfter = expiresAt.Add(-e.leaseDuration() / 2)
					continue
				}
				if err == nil || !now.Before(expiresAt) {
					cancel()
					return
				}
			}
			valid, err := e.Store.LeaseValid(ctx, lease, now)
			if err == nil && !valid {
				cancel()
				return
			}
		}
	}
}

func (e *Engine) Snapshot(ctx context.Context, runID workflowv3.RunID) (workflowv3.RunSnapshot, error) {
	if err := e.validate(); err != nil {
		return workflowv3.RunSnapshot{}, err
	}
	return e.Store.Snapshot(ctx, runID)
}

func (e *Engine) validate() error {
	if e == nil || e.Store == nil || e.Registry == nil || e.Artifacts == nil || e.Modules == nil {
		return fmt.Errorf("workflow v3 engine requires store, sealed registry, artifacts, and task modules")
	}
	advertised := e.Registry.ModuleAliases()
	configured := e.Modules.Aliases()
	if len(advertised) != len(configured) {
		return fmt.Errorf("registry advertises modules %v but runtime configures %v", advertised, configured)
	}
	for i := range advertised {
		if advertised[i] != configured[i] {
			return fmt.Errorf("registry advertises modules %v but runtime configures %v", advertised, configured)
		}
	}
	if provider, ok := e.Registry.(interface{ IsolationExecutorDigests() []string }); ok {
		digests := provider.IsolationExecutorDigests()
		if len(digests) > 0 {
			if e.Isolation == nil {
				return fmt.Errorf("registry requires restricted isolation but no executor is configured")
			}
			if err := e.Isolation.Validate(); err != nil {
				return fmt.Errorf("validate restricted isolation executor: %w", err)
			}
			for _, digest := range digests {
				if err := e.Isolation.Supports(digest); err != nil {
					return fmt.Errorf("registry isolation executor is unavailable: %w", err)
				}
			}
		}
	}
	return nil
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func (e *Engine) registryQuarantineThreshold() int {
	if e.RegistryQuarantineThreshold > 0 {
		return e.RegistryQuarantineThreshold
	}
	return 2
}

func (e *Engine) leaseDuration() time.Duration {
	if e.LeaseDuration > 0 {
		return e.LeaseDuration
	}
	return 30 * time.Second
}

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
	return false, nil
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
	attemptCtx, cancelAttempt := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	go e.watchLease(ctx, lease, cancelAttempt, watchDone)
	defer func() {
		close(watchDone)
		cancelAttempt()
	}()
	result, err := RunTask(attemptCtx, TaskRequest{
		RunID: lease.RunID, NodeKey: lease.NodeKey, Attempt: lease.Attempt,
		Task: registered, Inputs: inputs, Artifacts: e.Artifacts,
		Modules: e.Modules,
	})
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
	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case <-done:
			return
		case <-ticker.C:
			valid, err := e.Store.LeaseValid(ctx, lease, e.now())
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

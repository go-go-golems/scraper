package workflowv3runtime

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

// IsolationExecutorSet retains exact executable profiles during rolling
// registry coexistence and routes each attempt by its compiled executor digest.
type IsolationExecutorSet struct {
	executors map[string]IsolatedTaskExecutor
}

func NewIsolationExecutorSet(executors ...*BubblewrapExecutor) (*IsolationExecutorSet, error) {
	set := &IsolationExecutorSet{executors: map[string]IsolatedTaskExecutor{}}
	for _, executor := range executors {
		if executor == nil {
			return nil, fmt.Errorf("isolation executor is required")
		}
		if err := executor.Validate(); err != nil {
			return nil, err
		}
		digest, err := executor.Identity()
		if err != nil {
			return nil, err
		}
		if _, duplicate := set.executors[digest]; duplicate {
			return nil, fmt.Errorf("isolation executor digest %s is duplicated", digest)
		}
		set.executors[digest] = executor
	}
	if len(set.executors) == 0 {
		return nil, fmt.Errorf("isolation executor set cannot be empty")
	}
	return set, nil
}

func (s *IsolationExecutorSet) Validate() error {
	if s == nil || len(s.executors) == 0 {
		return fmt.Errorf("isolation executor set is required")
	}
	for _, executor := range s.executors {
		if err := executor.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (s *IsolationExecutorSet) Supports(digest string) error {
	if s == nil || s.executors[digest] == nil {
		return fmt.Errorf("isolated executor digest %s is unavailable", digest)
	}
	return nil
}

func (s *IsolationExecutorSet) Execute(ctx context.Context, request TaskRequest, isolation workflowv3.PlanIsolation) (TaskResult, error) {
	if err := s.Supports(isolation.ExecutorDigest); err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: err}
	}
	return s.executors[isolation.ExecutorDigest].Execute(ctx, request, isolation)
}

func (s *IsolationExecutorSet) Digests() []string {
	ret := make([]string, 0, len(s.executors))
	for digest := range s.executors {
		ret = append(ret, digest)
	}
	sort.Strings(ret)
	return ret
}

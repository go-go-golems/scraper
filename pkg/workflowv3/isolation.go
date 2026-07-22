package workflowv3

import (
	"fmt"
	"math"
)

const (
	IsolationInProcessTrusted     = "in-process.trusted"
	IsolationSubprocessRestricted = "subprocess.restricted"
)

const (
	maxIsolationWallTimeMillis = int64(24 * 60 * 60 * 1000)
	maxIsolationCPUTimeMillis  = int64(24 * 60 * 60 * 1000)
	maxIsolationMemoryBytes    = int64(1 << 40)
	maxIsolationOutputBytes    = int64(1 << 40)
	maxIsolationProtocolBytes  = int64(64 << 20)
	maxIsolationProcesses      = int64(1024)
	maxIsolationOutputFiles    = 100000
)

func RequiresRestrictedIsolation(modules []string) bool {
	for _, module := range modules {
		if module == "exec:allowlisted" || module == "fs:writable" || module == "process:spawn" {
			return true
		}
	}
	return false
}

func TrustedIsolationPolicy() IsolationPolicy {
	return IsolationPolicy{Class: IsolationInProcessTrusted}
}

func ValidateIsolationPolicy(policy IsolationPolicy) error {
	switch policy.Class {
	case IsolationInProcessTrusted:
		if policy.WallTimeMillis != 0 || policy.CPUTimeMillis != 0 || policy.MemoryBytes != 0 ||
			policy.MaxProcesses != 0 || policy.MaxOutputBytes != 0 ||
			policy.MaxOutputFiles != 0 || policy.MaxProtocolBytes != 0 {
			return fmt.Errorf("trusted in-process isolation cannot declare subprocess limits")
		}
		return nil
	case IsolationSubprocessRestricted:
		if policy.WallTimeMillis < 1 || policy.WallTimeMillis > maxIsolationWallTimeMillis {
			return fmt.Errorf("restricted isolation wall time is invalid")
		}
		if policy.CPUTimeMillis < 1 || policy.CPUTimeMillis > maxIsolationCPUTimeMillis {
			return fmt.Errorf("restricted isolation CPU time limit is invalid")
		}
		if policy.MemoryBytes < 1 || policy.MemoryBytes > maxIsolationMemoryBytes {
			return fmt.Errorf("restricted isolation memory limit is invalid")
		}
		if policy.MaxProcesses < 1 || policy.MaxProcesses > maxIsolationProcesses {
			return fmt.Errorf("restricted isolation process limit is invalid")
		}
		if policy.MaxOutputBytes < 1 || policy.MaxOutputBytes > maxIsolationOutputBytes {
			return fmt.Errorf("restricted isolation output byte limit is invalid")
		}
		if policy.MaxOutputFiles < 1 || policy.MaxOutputFiles > maxIsolationOutputFiles {
			return fmt.Errorf("restricted isolation output file limit is invalid")
		}
		if policy.MaxProtocolBytes < 1 || policy.MaxProtocolBytes > maxIsolationProtocolBytes {
			return fmt.Errorf("restricted isolation protocol byte limit is invalid")
		}
		return nil
	default:
		return fmt.Errorf("isolation class %q is not supported", policy.Class)
	}
}

func CompileIsolation(requested *IsolationPolicy, maximum IsolationPolicy, executorDigests ...string) (PlanIsolation, error) {
	if maximum.Class == "" {
		maximum = TrustedIsolationPolicy()
	}
	if err := ValidateIsolationPolicy(maximum); err != nil {
		return PlanIsolation{}, fmt.Errorf("task isolation maximum: %w", err)
	}
	selected := maximum
	if requested != nil {
		selected = *requested
		if err := ValidateIsolationPolicy(selected); err != nil {
			return PlanIsolation{}, fmt.Errorf("requested isolation: %w", err)
		}
	}
	if selected.Class != maximum.Class {
		return PlanIsolation{}, fmt.Errorf("requested isolation class %q does not match task-required class %q", selected.Class, maximum.Class)
	}
	effective := selected
	if selected.Class == IsolationSubprocessRestricted {
		effective.WallTimeMillis = minPositive(selected.WallTimeMillis, maximum.WallTimeMillis)
		effective.CPUTimeMillis = minPositive(selected.CPUTimeMillis, maximum.CPUTimeMillis)
		effective.MemoryBytes = minPositive(selected.MemoryBytes, maximum.MemoryBytes)
		effective.MaxProcesses = minPositive(selected.MaxProcesses, maximum.MaxProcesses)
		effective.MaxOutputBytes = minPositive(selected.MaxOutputBytes, maximum.MaxOutputBytes)
		effective.MaxOutputFiles = minPositiveInt(selected.MaxOutputFiles, maximum.MaxOutputFiles)
		effective.MaxProtocolBytes = minPositive(selected.MaxProtocolBytes, maximum.MaxProtocolBytes)
	}
	executorDigest := ""
	if len(executorDigests) > 0 {
		executorDigest = executorDigests[0]
	}
	if effective.Class == IsolationSubprocessRestricted {
		if err := validateSHA256Digest(executorDigest); err != nil {
			return PlanIsolation{}, fmt.Errorf("restricted isolation executor digest: %w", err)
		}
	}
	digest, err := Digest(struct {
		Policy         IsolationPolicy `json:"policy"`
		ExecutorDigest string          `json:"executorDigest,omitempty"`
	}{Policy: effective, ExecutorDigest: executorDigest})
	if err != nil {
		return PlanIsolation{}, err
	}
	return PlanIsolation{Requested: selected, Effective: effective, PolicyDigest: digest, ExecutorDigest: executorDigest}, nil
}

func minPositive(left, right int64) int64 {
	if left <= 0 {
		left = math.MaxInt64
	}
	if right <= 0 {
		right = math.MaxInt64
	}
	if left < right {
		return left
	}
	return right
}

func minPositiveInt(left, right int) int {
	if left <= 0 {
		left = math.MaxInt
	}
	if right <= 0 {
		right = math.MaxInt
	}
	if left < right {
		return left
	}
	return right
}

func ValidatePlanIsolation(isolation *PlanIsolation, maximum IsolationPolicy) error {
	if isolation == nil {
		compiled, err := CompileIsolation(nil, maximum)
		if err != nil {
			return err
		}
		if compiled.Effective.Class != IsolationInProcessTrusted {
			return fmt.Errorf("plan omits required isolation class %q", compiled.Effective.Class)
		}
		return nil
	}
	compiled, err := CompileIsolation(&isolation.Requested, maximum, isolation.ExecutorDigest)
	if err != nil {
		return err
	}
	if compiled != *isolation {
		return fmt.Errorf("compiled isolation policy does not match task maximum")
	}
	return nil
}

func EffectivePlanIsolation(isolation *PlanIsolation) PlanIsolation {
	if isolation != nil {
		return *isolation
	}
	trusted := TrustedIsolationPolicy()
	digest, err := Digest(struct {
		Policy         IsolationPolicy `json:"policy"`
		ExecutorDigest string          `json:"executorDigest,omitempty"`
	}{Policy: trusted})
	if err != nil {
		panic(err)
	}
	return PlanIsolation{Requested: trusted, Effective: trusted, PolicyDigest: digest}
}

func isolationMaximumValue(policy *IsolationPolicy) IsolationPolicy {
	if policy == nil {
		return TrustedIsolationPolicy()
	}
	return *policy
}

func cloneIsolationPolicy(policy *IsolationPolicy) *IsolationPolicy {
	if policy == nil {
		return nil
	}
	ret := *policy
	return &ret
}

package workflowv3runtime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type IsolatedTaskExecutor interface {
	Execute(context.Context, TaskRequest, workflowv3.PlanIsolation) (TaskResult, error)
	Supports(string) error
	Validate() error
}

type BubblewrapExecutor struct {
	WorkerExecutable     string
	BubblewrapExecutable string
	LauncherExecutable   string
	ScratchRoot          string
	Tools                map[string]string
	validateOnce         sync.Once
	validateErr          error
}

func (e *BubblewrapExecutor) Identity() (string, error) {
	if e == nil {
		return "", fmt.Errorf("isolated executor is required")
	}
	worker, err := executableDigest(e.WorkerExecutable)
	if err != nil {
		return "", err
	}
	launcher, err := executableDigest(e.launcher())
	if err != nil {
		return "", err
	}
	bubblewrap, err := executableDigest(e.bubblewrap())
	if err != nil {
		return "", err
	}
	type toolIdentity struct {
		ID     string `json:"id"`
		Digest string `json:"digest"`
	}
	tools := make([]toolIdentity, 0, len(e.Tools))
	for id, path := range e.Tools {
		digest, digestErr := executableDigest(path)
		if digestErr != nil {
			return "", digestErr
		}
		tools = append(tools, toolIdentity{ID: id, Digest: digest})
	}
	sort.Slice(tools, func(i, j int) bool { return tools[i].ID < tools[j].ID })
	return workflowv3.Digest(struct {
		Protocol   string         `json:"protocol"`
		Worker     string         `json:"worker"`
		Launcher   string         `json:"launcher"`
		Bubblewrap string         `json:"bubblewrap"`
		Tools      []toolIdentity `json:"tools,omitempty"`
	}{Protocol: IsolatedTaskRequestSchema + "+" + IsolatedTaskResponseSchema, Worker: worker, Launcher: launcher, Bubblewrap: bubblewrap, Tools: tools})
}

func (e *BubblewrapExecutor) Supports(digest string) error {
	identity, err := e.Identity()
	if err != nil {
		return err
	}
	if identity != digest {
		return fmt.Errorf("isolated executor digest %s is unavailable", digest)
	}
	return nil
}

func (e *BubblewrapExecutor) Validate() error {
	if e == nil {
		return fmt.Errorf("isolated executor is required")
	}
	e.validateOnce.Do(func() { e.validateErr = e.validateConfiguration() })
	return e.validateErr
}

func (e *BubblewrapExecutor) validateConfiguration() error {
	if _, err := exactExecutable(e.WorkerExecutable); err != nil {
		return fmt.Errorf("isolated worker: %w", err)
	}
	if _, err := exactExecutable(e.bubblewrap()); err != nil {
		return fmt.Errorf("bubblewrap: %w", err)
	}
	if _, err := exactExecutable(e.launcher()); err != nil {
		return fmt.Errorf("isolation launcher: %w", err)
	}
	for id, path := range e.Tools {
		if !allowlistedToolID.MatchString(id) {
			return fmt.Errorf("invalid isolated tool ID %q", id)
		}
		if _, err := exactExecutable(path); err != nil {
			return fmt.Errorf("isolated tool %q: %w", id, err)
		}
	}
	if _, err := e.Identity(); err != nil {
		return fmt.Errorf("compute isolation executor identity: %w", err)
	}
	probePolicy := workflowv3.IsolationPolicy{
		Class:          workflowv3.IsolationSubprocessRestricted,
		WallTimeMillis: 1000, CPUTimeMillis: 500, MemoryBytes: 64 << 20,
		MaxProcesses: 16, MaxOutputBytes: 1 << 20,
		MaxOutputFiles: 4, MaxProtocolBytes: 1 << 20,
	}
	group, err := createIsolationCgroup(probePolicy)
	if err != nil {
		return fmt.Errorf("cgroup v2 isolation is unavailable: %w", err)
	}
	if err := group.Close(); err != nil {
		return fmt.Errorf("cgroup v2 isolation cleanup failed: %w", err)
	}
	return nil
}

func (e *BubblewrapExecutor) Execute(
	ctx context.Context,
	request TaskRequest,
	isolation workflowv3.PlanIsolation,
) (TaskResult, error) {
	if err := e.Validate(); err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: err}
	}
	if err := workflowv3.ValidatePlanIsolation(&isolation, request.Task.Spec.IsolationMaximum); err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: err}
	}
	executorDigest, err := e.Identity()
	if err != nil || executorDigest != isolation.ExecutorDigest || executorDigest != request.Task.Spec.IsolationExecutorDigest {
		return TaskResult{}, &IsolationConstructionError{Err: fmt.Errorf("isolated executor identity does not match compiled plan and registry")}
	}
	if isolation.Effective.Class != workflowv3.IsolationSubprocessRestricted {
		return TaskResult{}, &IsolationConstructionError{Err: fmt.Errorf("unsupported isolated class %q", isolation.Effective.Class)}
	}
	worker, _ := exactExecutable(e.WorkerExecutable)
	bubblewrap, _ := exactExecutable(e.bubblewrap())
	launcher, _ := exactExecutable(e.launcher())
	root, err := os.MkdirTemp(e.ScratchRoot, "workflowv3-isolated-*")
	if err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: fmt.Errorf("create isolated staging root: %w", err)}
	}
	defer func() { _ = os.RemoveAll(root) }()
	bundleRoot := filepath.Join(root, "bundle")
	inputRoot := filepath.Join(root, "inputs")
	outputRoot := filepath.Join(root, "outputs")
	if err := os.MkdirAll(bundleRoot, 0o700); err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: err}
	}
	inputStore, err := workflowv3.NewFileArtifactStore(inputRoot, maximumInputSize(request.Inputs))
	if err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: err}
	}
	_, err = workflowv3.NewFileArtifactStore(outputRoot, isolation.Effective.MaxOutputBytes)
	if err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: err}
	}
	bundleFiles, err := stageBundle(bundleRoot, request.Task.Bundle)
	if err != nil {
		return TaskResult{}, &TaskPreparationError{Err: err}
	}
	localInputs, err := stageInputs(ctx, request.Artifacts, inputStore, request.Inputs)
	if err != nil {
		return TaskResult{}, &TaskPreparationError{Err: err}
	}
	toolIDs := make([]string, 0, len(e.Tools))
	for id := range e.Tools {
		toolIDs = append(toolIDs, id)
	}
	sort.Strings(toolIDs)
	tools := make([]IsolatedTool, 0, len(toolIDs))
	for index, id := range toolIDs {
		tools = append(tools, IsolatedTool{ID: id, Path: fmt.Sprintf("/tools/tool-%d", index)})
	}
	wireRequest := IsolatedTaskRequest{
		Schema: IsolatedTaskRequestSchema, RunID: request.RunID, NodeKey: request.NodeKey,
		Attempt: request.Attempt, Task: request.Task.Spec.Identity,
		Manifest: request.Task.Bundle.Manifest(), BundleFiles: bundleFiles,
		Isolation: isolation, Inputs: localInputs, Tools: tools,
	}
	body, err := workflowv3.CanonicalJSON(wireRequest)
	if err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: err}
	}
	body = append(body, '\n')
	if int64(len(body)) > isolation.Effective.MaxProtocolBytes {
		return TaskResult{}, &TaskPreparationError{Err: fmt.Errorf("isolated request exceeds protocol limit")}
	}
	attemptCtx, cancel := context.WithTimeout(ctx, time.Duration(isolation.Effective.WallTimeMillis)*time.Millisecond)
	defer cancel()
	stdout := &boundedBuffer{limit: isolation.Effective.MaxProtocolBytes}
	stderr := &boundedDiscard{limit: 64 << 10}
	arguments := []string{
		"--die-with-parent", "--unshare-all", "--new-session", "--clearenv",
		"--ro-bind", worker, "/worker",
		"--ro-bind", bundleRoot, "/bundle",
		"--ro-bind", inputRoot, "/inputs",
		"--bind", outputRoot, "/outputs",
		"--proc", "/proc", "--dev", "/dev", "--tmpfs", "/tmp",
	}
	if len(toolIDs) > 0 {
		arguments = append(arguments, "--dir", "/tools")
		for index, id := range toolIDs {
			tool, _ := exactExecutable(e.Tools[id])
			arguments = append(arguments, "--ro-bind", tool, fmt.Sprintf("/tools/tool-%d", index))
		}
	}
	arguments = append(arguments,
		"--chdir", "/", "--setenv", "LANG", "C.UTF-8", "--setenv", "PATH", "/nonexistent",
		"/worker", "--bundle-root", "/bundle", "--input-root", "/inputs", "--output-root", "/outputs",
	)
	cgroup, err := createIsolationCgroup(isolation.Effective)
	if err != nil {
		return TaskResult{}, &IsolationConstructionError{Err: fmt.Errorf("create isolation cgroup: %w", err)}
	}
	defer func() { _ = cgroup.Close() }()
	launcherArguments := append([]string{"--cgroup", cgroup.path, bubblewrap}, arguments...)
	command := exec.CommandContext(attemptCtx, launcher, launcherArguments...)
	command.Stdin = bytes.NewReader(body)
	command.Stdout = stdout
	command.Stderr = stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pdeathsig: syscall.SIGKILL}
	processDone := make(chan struct{})
	monitorDone := make(chan struct{})
	var cpuExceeded bool
	var monitorErr error
	go func() {
		defer close(monitorDone)
		ticker := time.NewTicker(5 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-attemptCtx.Done():
				_ = cgroup.Kill()
				return
			case <-processDone:
				return
			case <-ticker.C:
				usage, err := cgroup.CPUUsageMicros()
				if err != nil {
					monitorErr = err
					_ = cgroup.Kill()
					return
				}
				if usage > isolation.Effective.CPUTimeMillis*1000 {
					cpuExceeded = true
					_ = cgroup.Kill()
					return
				}
			}
		}
	}()
	runErr := command.Run()
	close(processDone)
	<-monitorDone
	if monitorErr != nil {
		return TaskResult{}, &IsolationConstructionError{Err: fmt.Errorf("monitor isolation CPU usage: %w", monitorErr)}
	}
	cpuUsage, cpuErr := cgroup.CPUUsageMicros()
	if cpuErr != nil {
		return TaskResult{}, &IsolationConstructionError{Err: fmt.Errorf("read isolation CPU evidence: %w", cpuErr)}
	}
	if cpuUsage > isolation.Effective.CPUTimeMillis*1000 {
		cpuExceeded = true
	}
	oomKills, oomErr := cgroup.OOMKills()
	if oomErr != nil {
		return TaskResult{}, &IsolationConstructionError{Err: fmt.Errorf("read isolation memory evidence: %w", oomErr)}
	}
	if cpuExceeded {
		return TaskResult{}, &TaskFailureError{Failure: workflowv3.Failure{Class: "resource", Code: "ISOLATION_CPU_LIMIT", Retryable: true, Message: "isolated task exceeded CPU limit"}}
	}
	if oomKills > 0 {
		return TaskResult{}, &TaskFailureError{Failure: workflowv3.Failure{Class: "resource", Code: "ISOLATION_MEMORY_LIMIT", Retryable: true, Message: "isolated task exceeded memory limit"}}
	}
	if attemptCtx.Err() != nil {
		code := "ISOLATION_WALL_TIME"
		class := "resource"
		if ctx.Err() != nil {
			code, class = "ISOLATION_CANCELED", "canceled"
		}
		return TaskResult{}, &TaskFailureError{Failure: workflowv3.Failure{Class: class, Code: code, Retryable: class == "resource", Message: "isolated task terminated"}}
	}
	if stdout.overflow || (int64(stdout.Len()) == isolation.Effective.MaxProtocolBytes &&
		(stdout.Len() == 0 || stdout.Bytes()[stdout.Len()-1] != '\n')) {
		return TaskResult{}, protocolTaskError("ISOLATION_FRAME_TOO_LARGE")
	}
	if runErr != nil {
		return TaskResult{}, &TaskFailureError{Failure: workflowv3.Failure{Class: "execution", Code: "ISOLATION_CHILD_EXIT", Retryable: true, Message: "isolated worker exited"}}
	}
	responseBody, err := readBoundedFrame(bytes.NewReader(stdout.Bytes()), isolation.Effective.MaxProtocolBytes)
	if err != nil {
		return TaskResult{}, protocolTaskError("ISOLATION_FRAME_INVALID")
	}
	var response IsolatedTaskResponse
	if err := workflowv3.StrictDecode(responseBody, &response); err != nil {
		return TaskResult{}, protocolTaskError("ISOLATION_FRAME_INVALID")
	}
	canonicalResponse, canonicalErr := workflowv3.CanonicalJSON(response)
	if canonicalErr != nil || !bytes.Equal(canonicalResponse, responseBody) {
		return TaskResult{}, protocolTaskError("ISOLATION_FRAME_INVALID")
	}
	if err := validateIsolatedResponse(wireRequest, response); err != nil {
		return TaskResult{}, protocolTaskError("ISOLATION_IDENTITY_MISMATCH")
	}
	if response.Failure != nil {
		return TaskResult{}, &TaskFailureError{Failure: *response.Failure, Usage: response.Usage}
	}
	outputs, err := publishCandidates(ctx, outputRoot, request.Artifacts, request.Task.Spec.Outputs, response.Outputs, isolation.Effective)
	if err != nil {
		return TaskResult{}, &TaskFailureError{Failure: workflowv3.Failure{Class: "resource", Code: "ISOLATION_OUTPUT_INVALID", Message: "isolated candidate output failed validation"}}
	}
	return TaskResult{Outputs: outputs, Usage: response.Usage}, nil
}

func (e *BubblewrapExecutor) launcher() string {
	if strings.TrimSpace(e.LauncherExecutable) != "" {
		return e.LauncherExecutable
	}
	return filepath.Join(filepath.Dir(e.WorkerExecutable), "workflowv3-isolation-launcher")
}

func (e *BubblewrapExecutor) bubblewrap() string {
	if strings.TrimSpace(e.BubblewrapExecutable) == "" {
		return "/usr/bin/bwrap"
	}
	return e.BubblewrapExecutable
}

type IsolationConstructionError struct{ Err error }

func (e *IsolationConstructionError) Error() string { return e.Err.Error() }
func (e *IsolationConstructionError) Unwrap() error { return e.Err }

func executableDigest(path string) (string, error) {
	resolved, err := exactExecutable(path)
	if err != nil {
		return "", err
	}
	body, err := os.ReadFile(resolved)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(body)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func exactExecutable(path string) (string, error) {
	if strings.TrimSpace(path) == "" || !filepath.IsAbs(path) {
		return "", fmt.Errorf("executable path must be absolute")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", fmt.Errorf("path is not an executable regular file")
	}
	return resolved, nil
}

func stageBundle(root string, bundle *workflowv3.Bundle) ([]string, error) {
	if bundle == nil {
		return nil, fmt.Errorf("bundle is required")
	}
	files := bundle.Files()
	paths := make([]string, 0, len(files))
	for name, body := range files {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return nil, err
		}
		if err := os.WriteFile(target, body, 0o400); err != nil {
			return nil, err
		}
		paths = append(paths, name)
	}
	sort.Strings(paths)
	return paths, nil
}

func stageInputs(ctx context.Context, source workflowv3.ArtifactStore, target *workflowv3.FileArtifactStore, inputs map[string]workflowv3.ArtifactRef) (map[string]workflowv3.ArtifactRef, error) {
	ret := make(map[string]workflowv3.ArtifactRef, len(inputs))
	for name, ref := range inputs {
		body, err := workflowv3.ReadArtifact(ctx, source, ref)
		if err != nil {
			return nil, err
		}
		local, err := target.Put(ctx, ref.Schema, ref.MediaType, body)
		if err != nil {
			return nil, err
		}
		if local.Digest != ref.Digest || local.Size != ref.Size {
			return nil, fmt.Errorf("staged input identity changed")
		}
		ret[name] = local
	}
	return ret, nil
}

func validateIsolatedResponse(request IsolatedTaskRequest, response IsolatedTaskResponse) error {
	if response.Schema != IsolatedTaskResponseSchema || response.RunID != request.RunID ||
		response.NodeKey != request.NodeKey || response.Attempt != request.Attempt ||
		response.Task != request.Task || response.IsolationPolicyDigest != request.Isolation.PolicyDigest {
		return fmt.Errorf("isolated response identity mismatch")
	}
	if response.Failure != nil && len(response.Outputs) != 0 {
		return fmt.Errorf("isolated failure cannot publish outputs")
	}
	if response.Failure != nil {
		if err := workflowv3.ValidateFailure(*response.Failure); err != nil {
			return fmt.Errorf("isolated failure is invalid: %w", err)
		}
	}
	if response.Failure == nil && response.Outputs == nil {
		return fmt.Errorf("isolated success requires outputs")
	}
	return nil
}

func publishCandidates(ctx context.Context, root string, destination workflowv3.ArtifactStore, expected map[string]string, candidates map[string]workflowv3.ArtifactRef, policy workflowv3.IsolationPolicy) (map[string]workflowv3.ArtifactRef, error) {
	if err := validateOutputTree(root, policy); err != nil {
		return nil, err
	}
	if len(candidates) != len(expected) || len(candidates) > policy.MaxOutputFiles {
		return nil, fmt.Errorf("isolated output cardinality exceeds policy")
	}
	var total int64
	published := make(map[string]workflowv3.ArtifactRef, len(candidates))
	for port, schema := range expected {
		candidate, ok := candidates[port]
		if !ok || candidate.Schema != schema {
			return nil, fmt.Errorf("isolated output schema mismatch")
		}
		body, err := readRegularCandidate(root, candidate)
		if err != nil {
			return nil, err
		}
		if candidate.Size > policy.MaxOutputBytes-total {
			return nil, fmt.Errorf("isolated output byte limit exceeded")
		}
		total += candidate.Size
		ref, err := destination.Put(ctx, candidate.Schema, candidate.MediaType, body)
		if err != nil {
			return nil, err
		}
		if ref.Digest != candidate.Digest || ref.Size != candidate.Size {
			return nil, fmt.Errorf("published candidate identity changed")
		}
		published[port] = ref
	}
	return published, nil
}

func validateOutputTree(root string, policy workflowv3.IsolationPolicy) error {
	count := 0
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("isolated output tree contains a non-regular file")
		}
		count++
		if count > policy.MaxOutputFiles || info.Size() > policy.MaxOutputBytes-total {
			return fmt.Errorf("isolated output tree exceeds compiled limits")
		}
		total += info.Size()
		return nil
	})
	return err
}

func readRegularCandidate(root string, ref workflowv3.ArtifactRef) ([]byte, error) {
	if err := workflowv3.ValidateArtifactRef(ref); err != nil {
		return nil, err
	}
	clean := filepath.Clean(filepath.FromSlash(ref.Locator))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.Dir(clean) != "objects" {
		return nil, fmt.Errorf("isolated output locator is invalid")
	}
	path := filepath.Join(root, clean)
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("isolated output is not a regular file")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(resolved, resolvedRoot+string(filepath.Separator)) {
		return nil, fmt.Errorf("isolated output escapes staging root")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Nlink != 1 {
		return nil, fmt.Errorf("isolated output has multiple hard links")
	}
	body, err := os.ReadFile(resolved)
	if err != nil {
		return nil, err
	}
	if int64(len(body)) != ref.Size {
		return nil, fmt.Errorf("isolated output size mismatch")
	}
	sum := sha256.Sum256(body)
	if "sha256:"+hex.EncodeToString(sum[:]) != ref.Digest {
		return nil, fmt.Errorf("isolated output digest mismatch")
	}
	return body, nil
}

func protocolTaskError(code string) error {
	return &TaskFailureError{Failure: workflowv3.Failure{Class: "protocol", Code: code, Message: "isolated worker protocol failed"}}
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int64
	overflow bool
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	remaining := b.limit - int64(b.Len())
	if remaining <= 0 {
		b.overflow = true
		return len(value), nil
	}
	write := value
	if int64(len(write)) > remaining {
		write = write[:remaining]
		b.overflow = true
	}
	_, _ = b.Buffer.Write(write)
	return len(value), nil
}

type boundedDiscard struct{ limit, written int64 }

func (b *boundedDiscard) Write(value []byte) (int, error) {
	remaining := b.limit - b.written
	if remaining > 0 {
		if int64(len(value)) < remaining {
			b.written += int64(len(value))
		} else {
			b.written = b.limit
		}
	}
	return len(value), nil
}

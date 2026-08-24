package workflowv3runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

const (
	IsolatedTaskRequestSchema  = "scraper-workflow-isolated-task-request/v1"
	IsolatedTaskResponseSchema = "scraper-workflow-isolated-task-response/v1"
	maxBootstrapFrameBytes     = int64(64 << 20)
)

type IsolatedTool struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type IsolatedTaskRequest struct {
	Schema      string                            `json:"schema"`
	RunID       workflowv3.RunID                  `json:"runId"`
	NodeKey     workflowv3.NodeKey                `json:"nodeKey"`
	Attempt     int                               `json:"attempt"`
	CancelEpoch int64                             `json:"cancelEpoch"`
	Task        workflowv3.ImplementationIdentity `json:"task"`
	Manifest    workflowv3.BundleManifest         `json:"manifest"`
	BundleFiles []string                          `json:"bundleFiles"`
	Isolation   workflowv3.PlanIsolation          `json:"isolation"`
	Inputs      map[string]workflowv3.ArtifactRef `json:"inputs"`
	Tools       []IsolatedTool                    `json:"tools,omitempty"`
}

type IsolatedTaskResponse struct {
	Schema                string                            `json:"schema"`
	RunID                 workflowv3.RunID                  `json:"runId"`
	NodeKey               workflowv3.NodeKey                `json:"nodeKey"`
	Attempt               int                               `json:"attempt"`
	Task                  workflowv3.ImplementationIdentity `json:"task"`
	IsolationPolicyDigest string                            `json:"isolationPolicyDigest"`
	Outputs               map[string]workflowv3.ArtifactRef `json:"outputs,omitempty"`
	Usage                 []workflowv3.BudgetAmount         `json:"usage,omitempty"`
	Failure               *workflowv3.Failure               `json:"failure,omitempty"`
}

type IsolatedWorkerOptions struct {
	BundleRoot string
	InputRoot  string
	OutputRoot string
}

func ServeIsolatedTask(ctx context.Context, input io.Reader, output io.Writer, options IsolatedWorkerOptions) error {
	requestBody, err := readBoundedFrame(input, maxBootstrapFrameBytes)
	if err != nil {
		return err
	}
	var request IsolatedTaskRequest
	if err := workflowv3.StrictDecode(requestBody, &request); err != nil {
		return fmt.Errorf("decode isolated request: %w", err)
	}
	canonicalRequest, err := workflowv3.CanonicalJSON(request)
	if err != nil || !bytes.Equal(canonicalRequest, requestBody) {
		return fmt.Errorf("isolated request is not canonical JSON")
	}
	if err := validateIsolatedRequest(request); err != nil {
		return err
	}
	if int64(len(requestBody)) > request.Isolation.Effective.MaxProtocolBytes {
		return fmt.Errorf("isolated request exceeds compiled protocol limit")
	}
	if err := applyIsolationLimits(request.Isolation.Effective); err != nil {
		return fmt.Errorf("apply isolation limits: %w", err)
	}
	files, err := readBundleFiles(options.BundleRoot, request.BundleFiles)
	if err != nil {
		return err
	}
	bundle, err := workflowv3.NewBundle(request.Manifest, files)
	if err != nil {
		return fmt.Errorf("reconstruct isolated bundle: %w", err)
	}
	if bundle.Digest() != request.Task.BundleDigest {
		return fmt.Errorf("isolated bundle digest mismatch")
	}
	var registered workflowv3.RegisteredTask
	found := false
	for _, spec := range bundle.TaskSpecs() {
		if spec.Identity == request.Task {
			registered = workflowv3.RegisteredTask{Spec: spec, Bundle: bundle}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("isolated task identity is not in bundle")
	}
	inputStore, err := workflowv3.NewFileArtifactStore(options.InputRoot, maximumInputSize(request.Inputs))
	if err != nil {
		return fmt.Errorf("open isolated input store: %w", err)
	}
	outputStore, err := workflowv3.NewFileArtifactStore(options.OutputRoot, request.Isolation.Effective.MaxOutputBytes)
	if err != nil {
		return fmt.Errorf("open isolated output store: %w", err)
	}
	factories := []TaskModuleFactory{FSInputModule()}
	if len(request.Tools) > 0 {
		tools := make(map[string]string, len(request.Tools))
		for _, tool := range request.Tools {
			if _, duplicate := tools[tool.ID]; duplicate {
				return fmt.Errorf("duplicate isolated tool ID")
			}
			if !strings.HasPrefix(tool.Path, "/tools/") || filepath.Clean(tool.Path) != tool.Path {
				return fmt.Errorf("invalid isolated tool path")
			}
			tools[tool.ID] = tool.Path
		}
		factories = append(factories, AllowlistedExecModule(tools))
	}
	modules, err := NewTaskModuleRegistry(factories...)
	if err != nil {
		return err
	}
	result, runErr := RunTask(ctx, TaskRequest{
		RunID: request.RunID, NodeKey: request.NodeKey, Attempt: request.Attempt,
		Task: registered, Inputs: request.Inputs, Artifacts: &splitArtifactStore{read: inputStore, write: outputStore}, Modules: modules,
	})
	response := IsolatedTaskResponse{
		Schema: IsolatedTaskResponseSchema, RunID: request.RunID, NodeKey: request.NodeKey,
		Attempt: request.Attempt, Task: request.Task, IsolationPolicyDigest: request.Isolation.PolicyDigest,
	}
	if runErr == nil {
		response.Outputs, response.Usage = result.Outputs, result.Usage
	} else {
		response.Failure = isolatedFailure(runErr)
	}
	body, err := workflowv3.CanonicalJSON(response)
	if err != nil {
		return err
	}
	if int64(len(body)+1) > request.Isolation.Effective.MaxProtocolBytes {
		return fmt.Errorf("isolated response exceeds compiled protocol limit")
	}
	body = append(body, '\n')
	_, err = output.Write(body)
	return err
}

func validateIsolatedRequest(request IsolatedTaskRequest) error {
	if request.Schema != IsolatedTaskRequestSchema {
		return fmt.Errorf("unsupported isolated request schema %q", request.Schema)
	}
	if request.RunID == "" || request.NodeKey == "" || request.Attempt < 1 {
		return fmt.Errorf("isolated attempt identity is invalid")
	}
	if request.Isolation.Effective.Class != workflowv3.IsolationSubprocessRestricted {
		return fmt.Errorf("isolated request requires restricted subprocess policy")
	}
	if err := workflowv3.ValidatePlanIsolation(&request.Isolation, request.Isolation.Effective); err != nil {
		return fmt.Errorf("isolated policy identity: %w", err)
	}
	if !sort.StringsAreSorted(request.BundleFiles) {
		return fmt.Errorf("isolated bundle files must be sorted")
	}
	previousTool := ""
	for _, tool := range request.Tools {
		if tool.ID <= previousTool {
			return fmt.Errorf("isolated tools must be strictly sorted")
		}
		previousTool = tool.ID
	}
	if request.Manifest.ABI != workflowv3.TaskABI || request.Task.ABI != workflowv3.TaskABI {
		return fmt.Errorf("isolated task ABI mismatch")
	}
	for _, ref := range request.Inputs {
		if err := workflowv3.ValidateArtifactRef(ref); err != nil {
			return fmt.Errorf("isolated input ref: %w", err)
		}
	}
	return nil
}

func readBoundedFrame(reader io.Reader, limit int64) ([]byte, error) {
	if limit < 1 || limit > maxBootstrapFrameBytes {
		return nil, fmt.Errorf("invalid protocol frame limit")
	}
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("protocol frame exceeds limit")
	}
	if len(body) == 0 || body[len(body)-1] != '\n' {
		return nil, fmt.Errorf("protocol frame must end with newline")
	}
	body = body[:len(body)-1]
	if bytes.ContainsRune(body, '\n') || len(bytes.TrimSpace(body)) != len(body) {
		return nil, fmt.Errorf("protocol requires exactly one canonical JSON line")
	}
	return body, nil
}

func maximumInputSize(inputs map[string]workflowv3.ArtifactRef) int64 {
	maximum := workflowv3.DefaultMaxArtifactBytes
	for _, ref := range inputs {
		if ref.Size > maximum {
			maximum = ref.Size
		}
	}
	return maximum
}

func readBundleFiles(root string, paths []string) (map[string][]byte, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{}
	for _, modulePath := range paths {
		clean := filepath.ToSlash(filepath.Clean(modulePath))
		if clean != modulePath || clean == "." || strings.HasPrefix(clean, "../") ||
			filepath.IsAbs(modulePath) || len(modulePath) > 256 {
			return nil, fmt.Errorf("invalid isolated bundle path")
		}
		if _, exists := files[modulePath]; exists {
			return nil, fmt.Errorf("duplicate isolated bundle path")
		}
		body, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(modulePath)))
		if readErr != nil {
			return nil, fmt.Errorf("read isolated bundle file: %w", readErr)
		}
		files[modulePath] = body
	}
	return files, nil
}

func isolatedFailure(err error) *workflowv3.Failure {
	var taskFailure *TaskFailureError
	if errors.As(err, &taskFailure) {
		failure := taskFailure.Failure
		failure.Message = "isolated task reported " + failure.Code
		return &failure
	}
	var preparation *TaskPreparationError
	if errors.As(err, &preparation) {
		return &workflowv3.Failure{Class: "internal", Code: "WORKFLOW_TASK_PREPARATION", Message: "isolated task preparation failed"}
	}
	var construction *RuntimeConstructionError
	if errors.As(err, &construction) {
		return &workflowv3.Failure{Class: "configuration", Code: "TASK_RUNTIME_CONSTRUCTION", Message: "isolated task runtime construction failed"}
	}
	return &workflowv3.Failure{Class: "internal", Code: "WORKFLOW_TASK_EXECUTION", Message: "isolated task execution failed"}
}

type splitArtifactStore struct {
	read  workflowv3.ArtifactStore
	write workflowv3.ArtifactStore
}

func (s *splitArtifactStore) Put(ctx context.Context, schema, mediaType string, body []byte) (workflowv3.ArtifactRef, error) {
	return s.write.Put(ctx, schema, mediaType, body)
}

func (s *splitArtifactStore) Open(ctx context.Context, ref workflowv3.ArtifactRef) (io.ReadCloser, error) {
	return s.read.Open(ctx, ref)
}

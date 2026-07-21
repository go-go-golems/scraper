package workflowv3runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	fsmod "github.com/go-go-golems/go-go-goja/modules/fs"
	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type TaskRequest struct {
	RunID     workflowv3.RunID
	NodeKey   workflowv3.NodeKey
	Attempt   int
	Task      workflowv3.RegisteredTask
	Inputs    map[string]workflowv3.ArtifactRef
	Artifacts workflowv3.ArtifactStore
}

type TaskResult struct {
	Outputs map[string]workflowv3.ArtifactRef
}

type TaskFailureError struct {
	Failure workflowv3.Failure
}

func (e *TaskFailureError) Error() string {
	return fmt.Sprintf("task failure %s/%s", e.Failure.Class, e.Failure.Code)
}

type outputState struct {
	ctx      context.Context
	store    workflowv3.ArtifactStore
	expected map[string]string
	outputs  map[string]workflowv3.ArtifactRef
}

var safePort = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

func RunTask(ctx context.Context, request TaskRequest) (TaskResult, error) {
	if request.Task.Bundle == nil || request.Artifacts == nil {
		return TaskResult{}, fmt.Errorf("task bundle and artifact store are required")
	}
	if err := validateInputRefs(request.Task.Spec, request.Inputs); err != nil {
		return TaskResult{}, err
	}
	workspace, inputValues, err := materializeInputs(ctx, request.Artifacts, request.Inputs)
	if err != nil {
		return TaskResult{}, err
	}
	defer func() { _ = os.RemoveAll(workspace) }()

	state := &outputState{
		ctx: ctx, store: request.Artifacts,
		expected: request.Task.Spec.Outputs,
		outputs:  map[string]workflowv3.ArtifactRef{},
	}
	modules := []gggengine.RuntimeModuleRegistrar{
		gggengine.NativeModuleRegistrar{
			ModuleID: "workflowv3:task", ModuleName: "workflow/task",
			Loader: taskModuleLoader(state),
		},
	}
	for _, module := range request.Task.Spec.Modules {
		switch module {
		case "fs:input":
			inputFS := fsmod.New(
				fsmod.WithName("fs:input"),
				fsmod.WithBackend(fsmod.NewReadOnlyFSBackend(fsmod.FSMount{
					FS: os.DirFS(workspace), Root: ".", Mount: "/",
				})),
			)
			modules = append(modules, gggengine.NativeModuleRegistrar{
				ModuleID: "workflowv3:fs-input", ModuleName: "fs:input",
				Loader: inputFS.Loader,
			})
		default:
			return TaskResult{}, fmt.Errorf("task requests unsupported module %q", module)
		}
	}
	loader := bundleLoader(request.Task.Bundle)
	factory, err := gggengine.NewRuntimeFactoryBuilder().
		WithRequireOptions(require.WithLoader(loader)).
		WithModules(modules...).
		Build()
	if err != nil {
		return TaskResult{}, fmt.Errorf("build task runtime: %w", err)
	}
	runtime, err := factory.NewRuntime(gggengine.WithStartupContext(ctx))
	if err != nil {
		return TaskResult{}, fmt.Errorf("create task runtime: %w", err)
	}
	defer func() { _ = runtime.Close(context.Background()) }()

	modulePath, exportName, err := splitEntrypoint(request.Task.Spec.Identity.Entrypoint)
	if err != nil {
		return TaskResult{}, err
	}
	returned, err := runtime.Owner.Call(ctx, "workflowv3.task", func(_ context.Context, vm *goja.Runtime) (any, error) {
		moduleValue, err := runtime.Require.Require("./" + modulePath)
		if err != nil {
			return nil, fmt.Errorf("require task module: %w", err)
		}
		functionValue := moduleValue.ToObject(vm).Get(exportName)
		function, ok := goja.AssertFunction(functionValue)
		if !ok {
			return nil, fmt.Errorf("entrypoint export %q is not a function", exportName)
		}
		contextObject := taskContextObject(vm, request, inputValues, state)
		value, err := function(goja.Undefined(), contextObject)
		if err != nil {
			if failure := exportedTaskFailure(err); failure != nil {
				return nil, failure
			}
			return nil, err
		}
		if promise, ok := value.Export().(*goja.Promise); ok {
			return promise, nil
		}
		return value.Export(), nil
	})
	if err != nil {
		return TaskResult{}, fmt.Errorf("execute task: %w", err)
	}
	if promise, ok := returned.(*goja.Promise); ok {
		returned, err = waitForTaskPromise(ctx, runtime, promise)
		if err != nil {
			return TaskResult{}, fmt.Errorf("await task: %w", err)
		}
	}
	if err := validateReturnedOutputs(returned, state); err != nil {
		return TaskResult{}, err
	}
	return TaskResult{Outputs: cloneRefs(state.outputs)}, nil
}

func taskModuleLoader(state *outputState) require.ModuleLoader {
	return func(vm *goja.Runtime, moduleObject *goja.Object) {
		exports := moduleObject.Get("exports").ToObject(vm)
		mustSet(vm, exports, "implementation", func(call goja.FunctionCall) goja.Value {
			if _, ok := goja.AssertFunction(call.Argument(0)); !ok {
				panic(vm.NewTypeError("task.implementation requires a function"))
			}
			return call.Argument(0)
		})
		mustSet(vm, exports, "success", func(call goja.FunctionCall) goja.Value {
			return call.Argument(0)
		})
		mustSet(vm, exports, "failure", func(call goja.FunctionCall) goja.Value {
			value := call.Argument(0).ToObject(vm)
			failure := workflowv3.Failure{
				Class:     strings.TrimSpace(value.Get("class").String()),
				Code:      strings.TrimSpace(value.Get("code").String()),
				Retryable: value.Get("retryable").ToBoolean(),
				Message:   strings.TrimSpace(value.Get("message").String()),
			}
			if err := workflowv3.ValidateFailure(failure); err != nil {
				panic(vm.NewTypeError("invalid task failure: %s", err))
			}
			object := vm.NewObject()
			mustSet(vm, object, "class", failure.Class)
			mustSet(vm, object, "code", failure.Code)
			mustSet(vm, object, "retryable", failure.Retryable)
			mustSet(vm, object, "message", failure.Message)
			mustSet(vm, object, "__workflowTaskFailure", true)
			return object
		})
	}
}

func taskContextObject(vm *goja.Runtime, request TaskRequest, inputs map[string]any, state *outputState) *goja.Object {
	object := vm.NewObject()
	mustSet(vm, object, "input", func() map[string]any { return inputs })
	mustSet(vm, object, "identity", func() map[string]any {
		return map[string]any{
			"runId": string(request.RunID), "nodeKey": string(request.NodeKey),
			"attempt": request.Attempt,
		}
	})
	mustSet(vm, object, "checkpoint", func() {
		if err := state.ctx.Err(); err != nil {
			panic(vm.NewGoError(err))
		}
	})
	outputs := vm.NewObject()
	mustSet(vm, outputs, "putJSON", func(call goja.FunctionCall) goja.Value {
		port := strings.TrimSpace(call.Argument(0).String())
		options := call.Argument(1).ToObject(vm)
		schema := strings.TrimSpace(options.Get("schema").String())
		expected, ok := state.expected[port]
		if !ok {
			panic(vm.NewTypeError("undeclared output port %s", port))
		}
		if schema != expected {
			panic(vm.NewTypeError("output %s schema %s does not match %s", port, schema, expected))
		}
		body, err := json.Marshal(options.Get("value").Export())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		ref, err := state.store.Put(state.ctx, schema, "application/json", body)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		state.outputs[port] = ref
		return jsonValue(vm, ref)
	})
	mustSet(vm, object, "outputs", outputs)
	return object
}

func materializeInputs(ctx context.Context, store workflowv3.ArtifactStore, refs map[string]workflowv3.ArtifactRef) (string, map[string]any, error) {
	workspace, err := os.MkdirTemp("", "workflowv3-input-*")
	if err != nil {
		return "", nil, fmt.Errorf("create task input workspace: %w", err)
	}
	values := make(map[string]any, len(refs))
	for port, ref := range refs {
		if !safePort.MatchString(port) {
			_ = os.RemoveAll(workspace)
			return "", nil, fmt.Errorf("invalid input port %q", port)
		}
		body, err := workflowv3.ReadArtifact(ctx, store, ref)
		if err != nil {
			_ = os.RemoveAll(workspace)
			return "", nil, fmt.Errorf("materialize input %s: %w", port, err)
		}
		name := port + ".artifact"
		if err := os.WriteFile(filepath.Join(workspace, name), body, 0o400); err != nil {
			_ = os.RemoveAll(workspace)
			return "", nil, fmt.Errorf("write task input %s: %w", port, err)
		}
		values[port] = map[string]any{
			"schema": ref.Schema, "digest": ref.Digest, "mediaType": ref.MediaType,
			"size": ref.Size, "path": "/" + name,
		}
	}
	return workspace, values, nil
}

func validateInputRefs(spec workflowv3.TaskSpec, refs map[string]workflowv3.ArtifactRef) error {
	if len(refs) != len(spec.Inputs) {
		return fmt.Errorf("task has %d input refs, expected %d", len(refs), len(spec.Inputs))
	}
	for port, schema := range spec.Inputs {
		ref, ok := refs[port]
		if !ok {
			return fmt.Errorf("task input %q is missing", port)
		}
		if err := workflowv3.ValidateArtifactRef(ref); err != nil {
			return fmt.Errorf("task input %q: %w", port, err)
		}
		if ref.Schema != schema {
			return fmt.Errorf("task input %q schema %q does not match %q", port, ref.Schema, schema)
		}
	}
	return nil
}

func exportedTaskFailure(err error) *TaskFailureError {
	var exception *goja.Exception
	if !errors.As(err, &exception) {
		return nil
	}
	return taskFailureFromValue(exception.Value())
}

func taskFailureFromValue(value goja.Value) *TaskFailureError {
	object, ok := value.(*goja.Object)
	if !ok {
		return nil
	}
	marker := object.Get("__workflowTaskFailure")
	if marker == nil || !marker.ToBoolean() {
		return nil
	}
	return &TaskFailureError{Failure: workflowv3.Failure{
		Class:     strings.TrimSpace(object.Get("class").String()),
		Code:      strings.TrimSpace(object.Get("code").String()),
		Retryable: object.Get("retryable").ToBoolean(),
		Message:   strings.TrimSpace(object.Get("message").String()),
	}}
}

type taskPromiseSnapshot struct {
	state   goja.PromiseState
	value   any
	failure *TaskFailureError
	message string
}

func waitForTaskPromise(ctx context.Context, runtime *gggengine.Runtime, promise *goja.Promise) (any, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		result, err := runtime.Owner.Call(ctx, "workflowv3.task.promise", func(_ context.Context, _ *goja.Runtime) (any, error) {
			snapshot := taskPromiseSnapshot{state: promise.State()}
			switch snapshot.state {
			case goja.PromiseStatePending:
				// The outer loop waits without exporting a Goja value.
			case goja.PromiseStateFulfilled:
				value := promise.Result()
				if value != nil && !goja.IsUndefined(value) && !goja.IsNull(value) {
					snapshot.value = value.Export()
				}
			case goja.PromiseStateRejected:
				value := promise.Result()
				snapshot.failure = taskFailureFromValue(value)
				if snapshot.failure == nil && value != nil {
					snapshot.message = value.String()
				}
			}
			return snapshot, nil
		})
		if err != nil {
			return nil, err
		}
		snapshot := result.(taskPromiseSnapshot)
		switch snapshot.state {
		case goja.PromiseStatePending:
			time.Sleep(5 * time.Millisecond)
		case goja.PromiseStateFulfilled:
			return snapshot.value, nil
		case goja.PromiseStateRejected:
			if snapshot.failure != nil {
				return nil, snapshot.failure
			}
			if snapshot.message == "" {
				return nil, fmt.Errorf("promise rejected")
			}
			return nil, fmt.Errorf("promise rejected: %s", snapshot.message)
		default:
			return nil, fmt.Errorf("unknown promise state %v", snapshot.state)
		}
	}
}

func validateReturnedOutputs(returned any, state *outputState) error {
	values, ok := returned.(map[string]any)
	if !ok {
		return fmt.Errorf("task must return task.success({outputs})")
	}
	if len(values) != len(state.expected) || len(state.outputs) != len(state.expected) {
		return fmt.Errorf("task returned %d outputs and wrote %d, expected %d", len(values), len(state.outputs), len(state.expected))
	}
	for port := range state.expected {
		if _, ok := values[port]; !ok {
			return fmt.Errorf("task return is missing output %q", port)
		}
		if _, ok := state.outputs[port]; !ok {
			return fmt.Errorf("task did not write output %q", port)
		}
	}
	return nil
}

func bundleLoader(bundle *workflowv3.Bundle) func(string) ([]byte, error) {
	return func(modulePath string) ([]byte, error) {
		clean := filepath.ToSlash(filepath.Clean(strings.TrimPrefix(strings.TrimPrefix(modulePath, "./"), "/")))
		for _, candidate := range []string{clean, clean + ".js", clean + ".cjs"} {
			if body, ok := bundle.File(candidate); ok {
				return body, nil
			}
		}
		return nil, require.ModuleFileDoesNotExistError
	}
}

func splitEntrypoint(entrypoint string) (string, string, error) {
	modulePath, exportName, ok := strings.Cut(entrypoint, "#")
	if !ok || strings.TrimSpace(modulePath) == "" || strings.TrimSpace(exportName) == "" {
		return "", "", fmt.Errorf("invalid entrypoint %q", entrypoint)
	}
	return strings.TrimPrefix(modulePath, "./"), exportName, nil
}

func cloneRefs(input map[string]workflowv3.ArtifactRef) map[string]workflowv3.ArtifactRef {
	ret := make(map[string]workflowv3.ArtifactRef, len(input))
	for key, value := range input {
		ret[key] = value
	}
	return ret
}

func mustSet(vm *goja.Runtime, object *goja.Object, name string, value any) {
	if err := object.Set(name, value); err != nil {
		panic(vm.NewGoError(err))
	}
}

func jsonValue(vm *goja.Runtime, value any) goja.Value {
	body, err := json.Marshal(value)
	if err != nil {
		panic(vm.NewGoError(err))
	}
	var plain any
	if err := json.Unmarshal(body, &plain); err != nil {
		panic(vm.NewGoError(err))
	}
	return vm.ToValue(plain)
}

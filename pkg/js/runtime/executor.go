package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/rs/zerolog/log"
)

const (
	MetadataScript = "script"
)

type DependencyResolver interface {
	Result(ctx context.Context, workflowID model.WorkflowID, opID model.OpID) (*model.OpResult, error)
}

type ExecutorConfig struct {
	ScriptsFS               fs.FS
	ScriptsRoot             string
	Modules                 []gggengine.ModuleSpec
	RuntimeModuleRegistrars []gggengine.RuntimeModuleRegistrar
	ScraperDB               databasemod.QueryExecer
	SiteDB                  databasemod.QueryExecer
}

type ExecutionRequest struct {
	Workflow     model.WorkflowRun
	Op           model.OpSpec
	Lease        model.Lease
	Now          time.Time
	Dependencies DependencyResolver
}

type Executor struct {
	config ExecutorConfig
}

func NewExecutor(config ExecutorConfig) *Executor {
	return &Executor{config: config}
}

func (e *Executor) Execute(ctx context.Context, req ExecutionRequest) (*model.OpResult, error) {
	scriptPath := strings.TrimSpace(req.Op.Metadata[MetadataScript])
	if scriptPath == "" {
		return nil, fmt.Errorf("op metadata %q is required for js execution", MetadataScript)
	}

	loader := moduleLoader(e.config.ScriptsFS, e.config.ScriptsRoot)
	builder := gggengine.NewBuilder().
		WithRequireOptions(require.WithLoader(loader)).
		WithModules(e.config.Modules...).
		WithRuntimeModuleRegistrars(NewDatabaseRegistrar(DatabaseRegistrarConfig{
			ScraperDB: e.config.ScraperDB,
			SiteDB:    e.config.SiteDB,
		}))
	if len(e.config.RuntimeModuleRegistrars) > 0 {
		builder = builder.WithRuntimeModuleRegistrars(e.config.RuntimeModuleRegistrars...)
	}

	factory, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("build js runtime: %w", err)
	}

	runtime, err := factory.NewRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("create js runtime: %w", err)
	}
	defer func() {
		_ = runtime.Close(context.Background())
	}()

	state := &executionState{
		now: req.Now,
	}

	resultValue, err := runtime.Owner.Call(ctx, "scraper.js.execute", func(_ context.Context, vm *goja.Runtime) (any, error) {
		moduleValue, err := runtime.Require.Require("./" + scriptPath)
		if err != nil {
			return nil, fmt.Errorf("require script %s: %w", scriptPath, err)
		}

		fn, err := resolveScriptFunction(vm, moduleValue, scriptPath)
		if err != nil {
			return nil, err
		}

		jsCtx, err := buildJSContext(vm, req, state)
		if err != nil {
			return nil, err
		}

		result, err := fn(goja.Undefined(), jsCtx)
		if err != nil {
			return nil, err
		}
		if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
			return nil, nil
		}
		if promise, ok := result.Export().(*goja.Promise); ok {
			return promise, nil
		}
		return result.Export(), nil
	})
	if err != nil {
		return nil, fmt.Errorf("execute js script %s: %w", scriptPath, err)
	}

	if promise, ok := resultValue.(*goja.Promise); ok {
		resultValue, err = WaitForPromise(ctx, runtime, promise)
		if err != nil {
			return nil, fmt.Errorf("await js script %s: %w", scriptPath, err)
		}
	}

	result, err := buildOpResult(req, state, resultValue)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

type executionState struct {
	now            time.Time
	records        []model.RecordWrite
	artifacts      []model.ArtifactWrite
	emitted        []model.OpSpec
	emittedCount   int
	artifactCount  int
	dependencyMemo map[model.OpID]*model.OpResult
	logEntries     []LogEntry
}

func buildJSContext(vm *goja.Runtime, req ExecutionRequest, state *executionState) (goja.Value, error) {
	workflowInput, err := decodeRawMessage(req.Workflow.Input)
	if err != nil {
		return nil, fmt.Errorf("decode workflow input: %w", err)
	}
	opInput, err := decodeRawMessage(req.Op.Input)
	if err != nil {
		return nil, fmt.Errorf("decode op input: %w", err)
	}

	ctxObj := vm.NewObject()
	_ = ctxObj.Set("site", string(req.Op.Site))
	_ = ctxObj.Set("now", req.Now.UTC().Format(time.RFC3339Nano))
	_ = ctxObj.Set("workflow", map[string]any{
		"id":       string(req.Workflow.ID),
		"site":     string(req.Workflow.Site),
		"name":     req.Workflow.Name,
		"status":   string(req.Workflow.Status),
		"input":    workflowInput,
		"metadata": cloneStringMap(req.Workflow.Metadata),
	})
	_ = ctxObj.Set("op", map[string]any{
		"id":         string(req.Op.ID),
		"workflowID": string(req.Op.WorkflowID),
		"site":       string(req.Op.Site),
		"kind":       req.Op.Kind,
		"queue":      string(req.Op.Queue),
		"dedupKey":   req.Op.DedupKey,
		"metadata":   cloneStringMap(req.Op.Metadata),
	})
	_ = ctxObj.Set("lease", map[string]any{
		"workerID":   req.Lease.WorkerID,
		"token":      req.Lease.Token,
		"acquiredAt": req.Lease.AcquiredAt.UTC().Format(time.RFC3339Nano),
		"expiresAt":  req.Lease.ExpiresAt.UTC().Format(time.RFC3339Nano),
	})
	_ = ctxObj.Set("input", opInput)

	_ = ctxObj.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			parts = append(parts, fmt.Sprint(arg.Export()))
		}
		msg := strings.Join(parts, " ")
		log.Info().
			Str("component", "js-runner").
			Str("site", string(req.Op.Site)).
			Str("workflow_id", string(req.Workflow.ID)).
			Str("op_id", string(req.Op.ID)).
			Msg(msg)
		state.logEntries = append(state.logEntries, LogEntry{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Message:   msg,
		})
		return goja.Undefined()
	})

	_ = ctxObj.Set("dep", func(call goja.FunctionCall) goja.Value {
		opID := model.OpID(call.Argument(0).String())
		depResult, err := resolveDependency(context.Background(), req, state, opID)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		if depResult == nil {
			return goja.Null()
		}

		exported, err := exportOpResult(depResult)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(exported)
	})

	_ = ctxObj.Set("emit", func(call goja.FunctionCall) goja.Value {
		spec, err := emittedOpFromJS(req, state, call.Argument(0).Export())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		state.emitted = append(state.emitted, spec)
		return vm.ToValue(string(spec.ID))
	})

	_ = ctxObj.Set("writeRecord", func(call goja.FunctionCall) goja.Value {
		record, err := recordWriteFromJS(call.Argument(0).Export(), call.Argument(1).Export(), call.Argument(2).Export())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		state.records = append(state.records, record)
		return goja.Undefined()
	})

	_ = ctxObj.Set("writeArtifact", func(call goja.FunctionCall) goja.Value {
		artifact, err := artifactWriteFromJS(req, state, call.Argument(0).Export())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		state.artifacts = append(state.artifacts, artifact)
		return vm.ToValue(string(artifact.ID))
	})

	return ctxObj, nil
}

func resolveScriptFunction(vm *goja.Runtime, value goja.Value, scriptPath string) (goja.Callable, error) {
	if fn, ok := goja.AssertFunction(value); ok {
		return fn, nil
	}

	obj := value.ToObject(vm)
	defaultValue := obj.Get("default")
	if fn, ok := goja.AssertFunction(defaultValue); ok {
		return fn, nil
	}

	return nil, fmt.Errorf("script %s must export a function or default function", scriptPath)
}

func buildOpResult(req ExecutionRequest, state *executionState, returned any) (*model.OpResult, error) {
	result := &model.OpResult{
		OpID:        req.Op.ID,
		Records:     append([]model.RecordWrite(nil), state.records...),
		Artifacts:   append([]model.ArtifactWrite(nil), state.artifacts...),
		Emitted:     append([]model.OpSpec(nil), state.emitted...),
		CompletedAt: req.Now,
	}
	for _, emitted := range state.emitted {
		result.EmittedIDs = append(result.EmittedIDs, emitted.ID)
	}

	// Persist captured log entries as a special artifact
	if len(state.logEntries) > 0 {
		logBody, err := json.Marshal(state.logEntries)
		if err == nil {
			result.Artifacts = append(result.Artifacts, model.ArtifactWrite{
				ID:          model.ArtifactID(fmt.Sprintf("%s:execution-log", req.Op.ID)),
				Name:        "execution-log",
				Kind:        "execution-log",
				ContentType: "application/json",
				Body:        logBody,
			})
		}
	}

	if returned == nil {
		return result, nil
	}

	envelope, isEnvelope := returned.(map[string]any)
	if isEnvelope {
		if data, ok := envelope["data"]; ok {
			raw, err := marshalJSON(data)
			if err != nil {
				return nil, fmt.Errorf("marshal result data: %w", err)
			}
			result.Data = raw
		}
		if errorValue, ok := envelope["error"]; ok && errorValue != nil {
			opErr, err := opErrorFromJS(errorValue, req.Now)
			if err != nil {
				return nil, err
			}
			result.Error = opErr
		}
		return result, nil
	}

	raw, err := marshalJSON(returned)
	if err != nil {
		return nil, fmt.Errorf("marshal result value: %w", err)
	}
	result.Data = raw
	return result, nil
}

func emittedOpFromJS(req ExecutionRequest, state *executionState, value any) (model.OpSpec, error) {
	m, ok := value.(map[string]any)
	if !ok {
		return model.OpSpec{}, fmt.Errorf("emit expects an object")
	}

	kind := strings.TrimSpace(asString(m["kind"]))
	if kind == "" {
		return model.OpSpec{}, fmt.Errorf("emitted op kind is required")
	}

	workflowID := req.Workflow.ID
	if text := strings.TrimSpace(asString(m["workflowID"])); text != "" {
		workflowID = model.WorkflowID(text)
	}

	site := req.Op.Site
	if text := strings.TrimSpace(asString(m["site"])); text != "" {
		site = model.SiteName(text)
	}

	parentID := req.Op.ID
	if text := strings.TrimSpace(asString(m["parentID"])); text != "" {
		parentID = model.OpID(text)
	}

	state.emittedCount++
	opID := model.OpID(strings.TrimSpace(asString(m["id"])))
	if opID == "" {
		opID = model.OpID(fmt.Sprintf("%s:emit:%03d", req.Op.ID, state.emittedCount))
	}

	input, err := marshalJSON(m["input"])
	if err != nil {
		return model.OpSpec{}, fmt.Errorf("marshal emitted op input: %w", err)
	}
	retry, err := retryPolicyFromJS(m["retry"])
	if err != nil {
		return model.OpSpec{}, err
	}
	dependsOn, err := dependenciesFromJS(m["dependsOn"])
	if err != nil {
		return model.OpSpec{}, err
	}

	return model.OpSpec{
		ID:         opID,
		WorkflowID: workflowID,
		ParentID:   &parentID,
		Site:       site,
		Kind:       kind,
		Queue:      model.QueueKey(asString(m["queue"])),
		DedupKey:   asString(m["dedupKey"]),
		Input:      input,
		DependsOn:  dependsOn,
		Retry:      retry,
		Metadata:   stringMapFromJS(m["metadata"]),
	}, nil
}

func recordWriteFromJS(collectionValue, keyValue, dataValue any) (model.RecordWrite, error) {
	collection := strings.TrimSpace(asString(collectionValue))
	if collection == "" {
		return model.RecordWrite{}, fmt.Errorf("record collection is required")
	}

	key := strings.TrimSpace(asString(keyValue))
	if key == "" {
		rawKey, err := marshalJSON(keyValue)
		if err != nil {
			return model.RecordWrite{}, fmt.Errorf("marshal record key: %w", err)
		}
		key = string(rawKey)
	}

	data, err := marshalJSON(dataValue)
	if err != nil {
		return model.RecordWrite{}, fmt.Errorf("marshal record data: %w", err)
	}

	return model.RecordWrite{
		Collection: collection,
		Key:        key,
		Data:       data,
	}, nil
}

func artifactWriteFromJS(req ExecutionRequest, state *executionState, value any) (model.ArtifactWrite, error) {
	m, ok := value.(map[string]any)
	if !ok {
		return model.ArtifactWrite{}, fmt.Errorf("artifact payload must be an object")
	}

	state.artifactCount++
	artifactID := model.ArtifactID(strings.TrimSpace(asString(m["id"])))
	if artifactID == "" {
		artifactID = model.ArtifactID(fmt.Sprintf("%s:artifact:%03d", req.Op.ID, state.artifactCount))
	}

	return model.ArtifactWrite{
		ID:          artifactID,
		Name:        asString(m["name"]),
		Kind:        asString(m["kind"]),
		ContentType: asString(m["contentType"]),
		Metadata:    stringMapFromJS(m["metadata"]),
		Body:        bytesFromJS(m["body"]),
	}, nil
}

func opErrorFromJS(value any, now time.Time) (*model.OpError, error) {
	m, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("result error must be an object")
	}

	details, err := marshalJSON(m["details"])
	if err != nil {
		return nil, fmt.Errorf("marshal result error details: %w", err)
	}

	return &model.OpError{
		Code:       asString(m["code"]),
		Message:    asString(m["message"]),
		Retryable:  asBool(m["retryable"]),
		Details:    details,
		OccurredAt: now,
	}, nil
}

func resolveDependency(
	ctx context.Context,
	req ExecutionRequest,
	state *executionState,
	opID model.OpID,
) (*model.OpResult, error) {
	if state.dependencyMemo == nil {
		state.dependencyMemo = map[model.OpID]*model.OpResult{}
	}
	if result, ok := state.dependencyMemo[opID]; ok {
		return result, nil
	}
	if req.Dependencies == nil {
		return nil, fmt.Errorf("dependency resolver is not configured")
	}

	result, err := req.Dependencies.Result(ctx, req.Workflow.ID, opID)
	if err != nil {
		return nil, fmt.Errorf("resolve dependency %s: %w", opID, err)
	}
	state.dependencyMemo[opID] = result
	return result, nil
}

func exportOpResult(result *model.OpResult) (map[string]any, error) {
	if result == nil {
		return nil, nil
	}

	data, err := decodeRawMessage(result.Data)
	if err != nil {
		return nil, fmt.Errorf("decode dependency data: %w", err)
	}

	ret := map[string]any{
		"opID":        string(result.OpID),
		"data":        data,
		"records":     exportRecords(result.Records),
		"artifacts":   exportArtifacts(result.Artifacts),
		"emittedIDs":  exportOpIDs(result.EmittedIDs),
		"completedAt": result.CompletedAt.UTC().Format(time.RFC3339Nano),
	}
	if result.Error != nil {
		details, err := decodeRawMessage(result.Error.Details)
		if err != nil {
			return nil, fmt.Errorf("decode dependency error details: %w", err)
		}
		ret["error"] = map[string]any{
			"code":       result.Error.Code,
			"message":    result.Error.Message,
			"retryable":  result.Error.Retryable,
			"details":    details,
			"occurredAt": result.Error.OccurredAt.UTC().Format(time.RFC3339Nano),
		}
	}

	return ret, nil
}

func exportRecords(records []model.RecordWrite) []map[string]any {
	ret := make([]map[string]any, 0, len(records))
	for _, record := range records {
		data, err := decodeRawMessage(record.Data)
		if err != nil {
			data = nil
		}
		ret = append(ret, map[string]any{
			"collection": record.Collection,
			"key":        record.Key,
			"data":       data,
		})
	}
	return ret
}

func exportArtifacts(artifacts []model.ArtifactWrite) []map[string]any {
	ret := make([]map[string]any, 0, len(artifacts))
	for _, artifact := range artifacts {
		ret = append(ret, map[string]any{
			"id":          string(artifact.ID),
			"name":        artifact.Name,
			"kind":        artifact.Kind,
			"contentType": artifact.ContentType,
			"metadata":    cloneStringMap(artifact.Metadata),
			"bodyText":    string(artifact.Body),
		})
	}
	return ret
}

func exportOpIDs(ids []model.OpID) []string {
	ret := make([]string, 0, len(ids))
	for _, id := range ids {
		ret = append(ret, string(id))
	}
	return ret
}

func decodeRawMessage(raw json.RawMessage) (any, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return nil, nil
	}

	var ret any
	if err := json.Unmarshal(raw, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func marshalJSON(value any) (json.RawMessage, error) {
	if value == nil {
		return nil, nil
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func stringMapFromJS(value any) map[string]string {
	ret := map[string]string{}
	m, ok := value.(map[string]any)
	if !ok {
		return ret
	}
	for key, v := range m {
		ret[key] = asString(v)
	}
	return ret
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	ret := make(map[string]string, len(in))
	for key, value := range in {
		ret[key] = value
	}
	return ret
}

func dependenciesFromJS(value any) ([]model.Dependency, error) {
	if value == nil {
		return nil, nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("dependsOn must be an array")
	}

	ret := make([]model.Dependency, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("dependency entries must be objects")
		}
		opID := strings.TrimSpace(asString(m["opID"]))
		if opID == "" {
			return nil, fmt.Errorf("dependency opID is required")
		}
		required := true
		if rawRequired, ok := m["required"]; ok {
			required = asBool(rawRequired)
		}
		ret = append(ret, model.Dependency{
			OpID:     model.OpID(opID),
			Required: required,
		})
	}

	return ret, nil
}

func retryPolicyFromJS(value any) (model.RetryPolicy, error) {
	if value == nil {
		return model.RetryPolicy{}, nil
	}
	m, ok := value.(map[string]any)
	if !ok {
		return model.RetryPolicy{}, fmt.Errorf("retry must be an object")
	}

	initialBackoff, err := durationFromJS(m["initialBackoff"])
	if err != nil {
		return model.RetryPolicy{}, fmt.Errorf("parse retry.initialBackoff: %w", err)
	}
	maxBackoff, err := durationFromJS(m["maxBackoff"])
	if err != nil {
		return model.RetryPolicy{}, fmt.Errorf("parse retry.maxBackoff: %w", err)
	}

	return model.RetryPolicy{
		MaxAttempts:    asInt(m["maxAttempts"]),
		BackoffKind:    model.BackoffKind(asString(m["backoffKind"])),
		InitialBackoff: initialBackoff,
		MaxBackoff:     maxBackoff,
		Multiplier:     asFloat64(m["multiplier"]),
	}, nil
}

func durationFromJS(value any) (time.Duration, error) {
	switch typed := value.(type) {
	case nil:
		return 0, nil
	case string:
		if strings.TrimSpace(typed) == "" {
			return 0, nil
		}
		return time.ParseDuration(typed)
	case int64:
		return time.Duration(typed), nil
	case int32:
		return time.Duration(typed), nil
	case int:
		return time.Duration(typed), nil
	case float64:
		return time.Duration(typed), nil
	default:
		return 0, fmt.Errorf("unsupported duration type %T", value)
	}
}

func bytesFromJS(value any) []byte {
	switch typed := value.(type) {
	case nil:
		return nil
	case string:
		return []byte(typed)
	case []byte:
		return append([]byte(nil), typed...)
	default:
		return []byte(asString(value))
	}
}

func asString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case json.RawMessage:
		return string(typed)
	case []byte:
		return string(typed)
	case bool:
		return strconv.FormatBool(typed)
	case int:
		return strconv.Itoa(typed)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(value)
	}
}

func asBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		ret, _ := strconv.ParseBool(typed)
		return ret
	default:
		return false
	}
}

func asInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		ret, _ := strconv.Atoi(typed)
		return ret
	default:
		return 0
	}
}

func asFloat64(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		ret, _ := strconv.ParseFloat(typed, 64)
		return ret
	default:
		return 0
	}
}

func moduleLoader(fsys fs.FS, root string) func(modulePath string) ([]byte, error) {
	scriptsFS := subFS(fsys, root)
	return func(modulePath string) ([]byte, error) {
		candidates := []string{
			cleanModulePath(modulePath),
			cleanModulePath(modulePath) + ".js",
		}
		for _, candidate := range candidates {
			body, err := fs.ReadFile(scriptsFS, candidate)
			if err == nil {
				return body, nil
			}
			if errorsIsNotExist(err) {
				continue
			}
			return nil, err
		}
		return nil, require.ModuleFileDoesNotExistError
	}
}

func cleanModulePath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = strings.TrimPrefix(path, "/")
	return filepath.Clean(path)
}

func subFS(fsys fs.FS, root string) fs.FS {
	if fsys == nil {
		return nilFS{}
	}
	root = strings.TrimSpace(root)
	if root == "" || root == "." {
		return fsys
	}
	sub, err := fs.Sub(fsys, root)
	if err != nil {
		return fsys
	}
	return sub
}

type nilFS struct{}

func (nilFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func errorsIsNotExist(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "not exist"))
}

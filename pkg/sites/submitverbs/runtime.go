package submitverbs

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	scraperjs "github.com/go-go-golems/scraper/pkg/js/runtime"
	"github.com/rs/zerolog/log"
)

type ExecutorConfig struct {
	Registry                *jsverbs.Registry
	VerbsFS                 fs.FS
	VerbsRoot               string
	Modules                 []gggengine.ModuleSpec
	RuntimeModuleRegistrars []gggengine.RuntimeModuleRegistrar
	ScraperDB               databasemod.QueryExecer
	SiteDB                  databasemod.QueryExecer
}

type ExecutionRequest struct {
	Site         model.SiteName
	Verb         *jsverbs.VerbSpec
	ParsedValues *values.Values
	Workflow     model.WorkflowRun
	Now          time.Time
}

type ExecutionResult struct {
	Workflow   model.WorkflowRun
	Submitted  []model.OpSpec
	TargetOpID model.OpID
	VerbData   json.RawMessage
}

type Executor struct {
	config ExecutorConfig
	files  map[string]*jsverbs.FileSpec
}

func NewExecutor(config ExecutorConfig) *Executor {
	files := map[string]*jsverbs.FileSpec{}
	if config.Registry != nil {
		for _, file := range config.Registry.Files {
			for _, alias := range moduleAliases(file.ModulePath) {
				files[alias] = file
			}
		}
	}
	return &Executor{
		config: config,
		files:  files,
	}
}

func (e *Executor) Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error) {
	if req.Verb == nil {
		return nil, fmt.Errorf("submission verb is required")
	}

	loader := e.moduleLoader()
	builder := gggengine.NewBuilder().
		WithRequireOptions(require.WithLoader(loader)).
		WithModules(e.config.Modules...).
		WithRuntimeModuleRegistrars(scraperjs.NewDatabaseRegistrar(scraperjs.DatabaseRegistrarConfig{
			ScraperDB: e.config.ScraperDB,
			SiteDB:    e.config.SiteDB,
		}))
	if len(e.config.RuntimeModuleRegistrars) > 0 {
		builder = builder.WithRuntimeModuleRegistrars(e.config.RuntimeModuleRegistrars...)
	}

	factory, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("build submit runtime: %w", err)
	}
	runtime, err := factory.NewRuntime(ctx)
	if err != nil {
		return nil, fmt.Errorf("create submit runtime: %w", err)
	}
	defer func() {
		_ = runtime.Close(context.Background())
	}()

	state := &submissionState{
		workflow: req.Workflow,
		now:      req.Now,
	}

	resultValue, err := runtime.Owner.Call(ctx, "scraper.submit.execute", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if _, err := runtime.Require.Require(req.Verb.File.ModulePath); err != nil {
			return nil, fmt.Errorf("require verb module %s: %w", req.Verb.File.ModulePath, err)
		}

		registryValue := vm.Get("__scraperSubmitVerbRegistry")
		if registryValue == nil || goja.IsUndefined(registryValue) || goja.IsNull(registryValue) {
			return nil, fmt.Errorf("submit verb registry not initialized")
		}
		entryValue := registryValue.ToObject(vm).Get(req.Verb.File.ModulePath)
		if entryValue == nil || goja.IsUndefined(entryValue) || goja.IsNull(entryValue) {
			return nil, fmt.Errorf("submit verb module entry missing for %s", req.Verb.File.ModulePath)
		}
		fnValue := entryValue.ToObject(vm).Get(req.Verb.FunctionName)
		fn, ok := goja.AssertFunction(fnValue)
		if !ok {
			return nil, fmt.Errorf("submit verb %s not captured in %s", req.Verb.FunctionName, req.Verb.File.RelPath)
		}

		jsCtx, err := buildSubmitContext(vm, req, state)
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
		return nil, fmt.Errorf("execute submit verb %s: %w", req.Verb.SourceRef(), err)
	}

	if promise, ok := resultValue.(*goja.Promise); ok {
		resultValue, err = scraperjs.WaitForPromise(ctx, runtime, promise)
		if err != nil {
			return nil, fmt.Errorf("await submit verb %s: %w", req.Verb.SourceRef(), err)
		}
	}

	if err := applySubmitReturn(state, resultValue); err != nil {
		return nil, err
	}

	return &ExecutionResult{
		Workflow:   state.workflow,
		Submitted:  append([]model.OpSpec(nil), state.emitted...),
		TargetOpID: state.targetOpID,
		VerbData:   append(json.RawMessage(nil), state.verbData...),
	}, nil
}

type submissionState struct {
	workflow     model.WorkflowRun
	now          time.Time
	emitted      []model.OpSpec
	emittedCount int
	targetOpID   model.OpID
	verbData     json.RawMessage
}

func buildSubmitContext(vm *goja.Runtime, req ExecutionRequest, state *submissionState) (goja.Value, error) {
	allValues, sectionValues := parseSubmitValues(req.ParsedValues)
	workflowInput, err := decodeRawMessage(state.workflow.Input)
	if err != nil {
		return nil, fmt.Errorf("decode workflow input: %w", err)
	}

	ctxObj := vm.NewObject()
	_ = ctxObj.Set("site", string(req.Site))
	_ = ctxObj.Set("now", req.Now.UTC().Format(time.RFC3339Nano))
	_ = ctxObj.Set("values", allValues)
	_ = ctxObj.Set("sections", sectionValues)
	_ = ctxObj.Set("command", map[string]any{
		"name":       req.Verb.Name,
		"fullPath":   req.Verb.FullPath(),
		"function":   req.Verb.FunctionName,
		"module":     req.Verb.File.ModulePath,
		"sourceFile": req.Verb.File.RelPath,
	})
	_ = ctxObj.Set("workflow", map[string]any{
		"id":       string(state.workflow.ID),
		"site":     string(state.workflow.Site),
		"name":     state.workflow.Name,
		"status":   string(state.workflow.Status),
		"input":    workflowInput,
		"metadata": cloneStringMap(state.workflow.Metadata),
	})

	_ = ctxObj.Set("log", func(call goja.FunctionCall) goja.Value {
		parts := make([]string, 0, len(call.Arguments))
		for _, arg := range call.Arguments {
			parts = append(parts, fmt.Sprint(arg.Export()))
		}
		log.Info().
			Str("component", "submit-verb").
			Str("site", string(req.Site)).
			Str("workflow_id", string(state.workflow.ID)).
			Str("verb", req.Verb.FullPath()).
			Msg(strings.Join(parts, " "))
		return goja.Undefined()
	})

	_ = ctxObj.Set("emit", func(call goja.FunctionCall) goja.Value {
		spec, err := emittedOpFromSubmitJS(req, state, call.Argument(0).Export())
		if err != nil {
			panic(vm.NewGoError(err))
		}
		state.emitted = append(state.emitted, spec)
		return vm.ToValue(string(spec.ID))
	})

	_ = ctxObj.Set("setTargetOpID", func(call goja.FunctionCall) goja.Value {
		state.targetOpID = model.OpID(strings.TrimSpace(call.Argument(0).String()))
		return goja.Undefined()
	})

	_ = ctxObj.Set("setWorkflowName", func(call goja.FunctionCall) goja.Value {
		state.workflow.Name = strings.TrimSpace(call.Argument(0).String())
		return goja.Undefined()
	})

	_ = ctxObj.Set("setWorkflowMetadata", func(call goja.FunctionCall) goja.Value {
		state.workflow.Metadata = stringMapFromJS(call.Argument(0).Export())
		return goja.Undefined()
	})

	return ctxObj, nil
}

func applySubmitReturn(state *submissionState, returned any) error {
	if returned == nil {
		return nil
	}

	if envelope, ok := returned.(map[string]any); ok {
		if target, ok := envelope["targetOpID"]; ok && state.targetOpID == "" {
			state.targetOpID = model.OpID(strings.TrimSpace(asString(target)))
		}
		if name, ok := envelope["workflowName"]; ok {
			state.workflow.Name = strings.TrimSpace(asString(name))
		}
		if metadata, ok := envelope["workflowMetadata"]; ok {
			state.workflow.Metadata = stringMapFromJS(metadata)
		}
		if data, ok := envelope["data"]; ok {
			raw, err := marshalJSON(data)
			if err != nil {
				return fmt.Errorf("marshal submit verb data: %w", err)
			}
			state.verbData = raw
			return nil
		}
	}

	raw, err := marshalJSON(returned)
	if err != nil {
		return fmt.Errorf("marshal submit verb result: %w", err)
	}
	state.verbData = raw
	return nil
}

func emittedOpFromSubmitJS(req ExecutionRequest, state *submissionState, value any) (model.OpSpec, error) {
	m, ok := value.(map[string]any)
	if !ok {
		return model.OpSpec{}, fmt.Errorf("emit expects an object")
	}

	kind := strings.TrimSpace(asString(m["kind"]))
	if kind == "" {
		return model.OpSpec{}, fmt.Errorf("emitted op kind is required")
	}

	workflowID := state.workflow.ID
	if text := strings.TrimSpace(asString(m["workflowID"])); text != "" {
		workflowID = model.WorkflowID(text)
	}

	site := req.Site
	if text := strings.TrimSpace(asString(m["site"])); text != "" {
		site = model.SiteName(text)
	}

	state.emittedCount++
	opID := model.OpID(strings.TrimSpace(asString(m["id"])))
	if opID == "" {
		opID = model.OpID(fmt.Sprintf("%s:submit:%03d", state.workflow.ID, state.emittedCount))
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

	var parentID *model.OpID
	if text := strings.TrimSpace(asString(m["parentID"])); text != "" {
		op := model.OpID(text)
		parentID = &op
	}

	return model.OpSpec{
		ID:         opID,
		WorkflowID: workflowID,
		ParentID:   parentID,
		Site:       site,
		Kind:       kind,
		Queue:      model.QueueKey(strings.TrimSpace(asString(m["queue"]))),
		DedupKey:   asString(m["dedupKey"]),
		Input:      input,
		DependsOn:  dependsOn,
		Retry:      retry,
		Metadata:   stringMapFromJS(m["metadata"]),
	}, nil
}

func parseSubmitValues(parsedValues *values.Values) (map[string]any, map[string]map[string]any) {
	allValues := map[string]any{}
	sections := map[string]map[string]any{}
	if parsedValues == nil {
		return allValues, sections
	}

	allValues = parsedValues.GetDataMap()
	parsedValues.ForEach(func(slug string, value *values.SectionValues) {
		sectionMap := map[string]any{}
		value.Fields.ForEach(func(_ string, fieldValue *fields.FieldValue) {
			if fieldValue == nil || fieldValue.Definition == nil {
				return
			}
			sectionMap[fieldValue.Definition.Name] = fieldValue.Value
		})
		sections[slug] = sectionMap
	})

	return allValues, sections
}

func (e *Executor) moduleLoader() func(modulePath string) ([]byte, error) {
	return func(modulePath string) ([]byte, error) {
		moduleKey := modulePath
		if file := e.files[modulePath]; file != nil {
			candidate := file.RelPath
			if e.config.VerbsRoot != "" {
				candidate = filepath.ToSlash(filepath.Join(e.config.VerbsRoot, file.RelPath))
			}
			source, err := fs.ReadFile(e.config.VerbsFS, candidate)
			if err != nil {
				return nil, err
			}
			return []byte(injectVerbOverlay(moduleKey, file, string(source))), nil
		}

		trimmed := strings.TrimPrefix(strings.TrimPrefix(modulePath, "./"), "/")
		candidates := []string{trimmed}
		if e.config.VerbsRoot != "" {
			candidates = append(candidates, filepath.ToSlash(filepath.Join(e.config.VerbsRoot, trimmed)))
		}
		var (
			source []byte
			err    error
		)
		for _, candidate := range candidates {
			source, err = fs.ReadFile(e.config.VerbsFS, candidate)
			if err == nil {
				file := e.files[moduleKey]
				return []byte(injectVerbOverlay(moduleKey, file, string(source))), nil
			}
		}
		return nil, require.ModuleFileDoesNotExistError
	}
}

func injectVerbOverlay(moduleKey string, file *jsverbs.FileSpec, source string) string {
	functionNames := []string{}
	if file != nil {
		for _, fn := range file.Functions {
			functionNames = append(functionNames, fn.Name)
		}
		sort.Strings(functionNames)
	}

	var suffix strings.Builder
	suffix.WriteString("\n")
	suffix.WriteString(`globalThis.__scraperSubmitVerbRegistry = globalThis.__scraperSubmitVerbRegistry || {};` + "\n")
	suffix.WriteString(`globalThis.__scraperSubmitVerbRegistry["`)
	suffix.WriteString(moduleKey)
	suffix.WriteString(`"] = {`)
	for i, name := range functionNames {
		if i > 0 {
			suffix.WriteString(",")
		}
		suffix.WriteString(name)
		suffix.WriteString(`: typeof `)
		suffix.WriteString(name)
		suffix.WriteString(` === "function" ? `)
		suffix.WriteString(name)
		suffix.WriteString(` : undefined`)
	}
	suffix.WriteString("};\n")

	prelude := strings.Join([]string{
		`globalThis.__scraperSubmitVerbRegistry = globalThis.__scraperSubmitVerbRegistry || {};`,
		`globalThis.__package__ = globalThis.__package__ || function() {};`,
		`globalThis.__section__ = globalThis.__section__ || function() {};`,
		`globalThis.__verb__ = globalThis.__verb__ || function() {};`,
		`globalThis.doc = globalThis.doc || function() { return ""; };`,
		"",
	}, "\n")

	return injectPrelude(source, prelude) + suffix.String()
}

func moduleAliases(modulePath string) []string {
	cleaned := filepath.ToSlash(strings.TrimSpace(modulePath))
	if cleaned == "" {
		return nil
	}
	base := strings.TrimPrefix(cleaned, "./")
	base = strings.TrimPrefix(base, "/")
	withoutExt := strings.TrimSuffix(base, filepath.Ext(base))

	candidates := []string{
		cleaned,
		"/" + base,
		base,
		"./" + base,
		withoutExt,
		"/" + withoutExt,
		"./" + withoutExt,
	}

	seen := map[string]struct{}{}
	ret := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		ret = append(ret, candidate)
	}
	return ret
}

func injectPrelude(source, prelude string) string {
	trimmed := strings.TrimLeft(source, "\ufeff \t\r\n")
	if strings.HasPrefix(trimmed, `"use strict";`) || strings.HasPrefix(trimmed, `'use strict';`) {
		if idx := strings.Index(source, "\n"); idx >= 0 {
			return source[:idx+1] + prelude + source[idx+1:]
		}
	}
	return prelude + source
}

func decodeRawMessage(raw json.RawMessage) (any, error) {
	if len(raw) == 0 {
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
		return []byte("null"), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func stringMapFromJS(value any) map[string]string {
	if value == nil {
		return nil
	}
	m, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	ret := map[string]string{}
	for key, candidate := range m {
		ret[key] = asString(candidate)
	}
	return ret
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	ret := make(map[string]string, len(in))
	for key, value := range in {
		ret[key] = value
	}
	return ret
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func retryPolicyFromJS(value any) (model.RetryPolicy, error) {
	if value == nil {
		return model.RetryPolicy{}, nil
	}
	m, ok := value.(map[string]any)
	if !ok {
		return model.RetryPolicy{}, fmt.Errorf("retry must be an object")
	}

	ret := model.RetryPolicy{
		MaxAttempts: intValue(m["maxAttempts"]),
		BackoffKind: model.BackoffKind(strings.TrimSpace(asString(m["backoffKind"]))),
		Multiplier:  floatValue(m["multiplier"]),
	}
	if ret.InitialBackoff, _ = time.ParseDuration(strings.TrimSpace(asString(m["initialBackoff"]))); ret.InitialBackoff < 0 {
		ret.InitialBackoff = 0
	}
	if ret.MaxBackoff, _ = time.ParseDuration(strings.TrimSpace(asString(m["maxBackoff"]))); ret.MaxBackoff < 0 {
		ret.MaxBackoff = 0
	}
	return ret, nil
}

func dependenciesFromJS(value any) ([]model.Dependency, error) {
	if value == nil {
		return nil, nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("dependsOn must be an array")
	}
	ret := make([]model.Dependency, 0, len(list))
	for _, candidate := range list {
		m, ok := candidate.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("dependency entry must be an object")
		}
		opID := strings.TrimSpace(asString(m["opID"]))
		if opID == "" {
			return nil, fmt.Errorf("dependency opID is required")
		}
		ret = append(ret, model.Dependency{
			OpID:     model.OpID(opID),
			Required: boolValue(m["required"], true),
		})
	}
	return ret, nil
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func floatValue(value any) float64 {
	switch v := value.(type) {
	case float32:
		return float64(v)
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func boolValue(value any, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	b, ok := value.(bool)
	if !ok {
		return defaultValue
	}
	return b
}

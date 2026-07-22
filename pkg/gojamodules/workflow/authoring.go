package workflowmodule

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type DescriptorModule struct {
	Name      string
	Factories map[string]workflowv3.TaskKey
}

type AuthoringResult struct {
	IR   workflowv3.WorkflowIR
	Plan workflowv3.WorkflowPlan
}

type authoringState struct {
	catalog     *workflowv3.Catalog
	refs        map[*goja.Object]workflowv3.ValueRef
	sets        map[*goja.Object]workflowv3.SetRef
	tasks       map[*goja.Object]taskInvocation
	jobs        map[*goja.Object]workflowv3.NodeKey
	workflows   map[*goja.Object]workflowv3.WorkflowIR
	plans       map[*goja.Object]workflowv3.WorkflowPlan
	planIR      map[*goja.Object]workflowv3.WorkflowIR
	activeBuild *planBuilder
}

type taskInvocation struct {
	key      workflowv3.TaskKey
	bindings map[string]workflowv3.ValueRef
}

type planBuilder struct {
	state  *authoringState
	ir     workflowv3.WorkflowIR
	closed bool
}

func Author(ctx context.Context, source string, catalog *workflowv3.Catalog, modules ...DescriptorModule) (AuthoringResult, error) {
	if catalog == nil {
		return AuthoringResult{}, fmt.Errorf("workflow catalog is required")
	}
	state := &authoringState{
		catalog:   catalog,
		refs:      map[*goja.Object]workflowv3.ValueRef{},
		sets:      map[*goja.Object]workflowv3.SetRef{},
		tasks:     map[*goja.Object]taskInvocation{},
		jobs:      map[*goja.Object]workflowv3.NodeKey{},
		workflows: map[*goja.Object]workflowv3.WorkflowIR{},
		plans:     map[*goja.Object]workflowv3.WorkflowPlan{},
		planIR:    map[*goja.Object]workflowv3.WorkflowIR{},
	}
	vm := goja.New()
	registry := require.NewRegistry()
	registry.RegisterNativeModule("workflow", state.workflowLoader)
	for _, module := range modules {
		module := module
		if strings.TrimSpace(module.Name) == "" {
			return AuthoringResult{}, fmt.Errorf("descriptor module name is required")
		}
		registry.RegisterNativeModule(module.Name, state.descriptorLoader(module))
	}
	registry.Enable(vm)

	moduleObject := vm.NewObject()
	if err := moduleObject.Set("exports", vm.NewObject()); err != nil {
		return AuthoringResult{}, err
	}
	wrapped, err := vm.RunString("(function(module, exports, require) {\n" + source + "\n})")
	if err != nil {
		return AuthoringResult{}, fmt.Errorf("compile workflow script: %w", err)
	}
	call, ok := goja.AssertFunction(wrapped)
	if !ok {
		return AuthoringResult{}, fmt.Errorf("workflow script wrapper is not callable")
	}
	if err := ctx.Err(); err != nil {
		return AuthoringResult{}, err
	}
	if _, err := call(goja.Undefined(), moduleObject, moduleObject.Get("exports"), vm.Get("require")); err != nil {
		return AuthoringResult{}, fmt.Errorf("execute workflow script: %w", err)
	}
	exported := moduleObject.Get("exports")
	if exported == nil || goja.IsUndefined(exported) || goja.IsNull(exported) {
		return AuthoringResult{}, fmt.Errorf("workflow script must export workflow.compile(...) result")
	}
	object := exported.ToObject(vm)
	plan, ok := state.plans[object]
	if !ok {
		return AuthoringResult{}, fmt.Errorf("workflow script export is not a compiled workflow plan")
	}
	return AuthoringResult{IR: state.planIR[object], Plan: plan}, nil
}

func (s *authoringState) workflowLoader(vm *goja.Runtime, moduleObject *goja.Object) {
	exports := moduleObject.Get("exports").ToObject(vm)
	mustSet(vm, exports, "define", func(call goja.FunctionCall) goja.Value {
		name := strings.TrimSpace(call.Argument(0).String())
		build, ok := goja.AssertFunction(call.Argument(1))
		if name == "" || !ok {
			panic(vm.NewTypeError("workflow.define(name, build) requires a name and callback"))
		}
		if s.activeBuild != nil {
			panic(vm.NewTypeError("nested workflow.define is not allowed"))
		}
		builder := &planBuilder{state: s, ir: workflowv3.WorkflowIR{
			Schema: workflowv3.IRSchema, Name: name,
			Inputs: []workflowv3.IRInput{}, Nodes: []workflowv3.IRNode{}, Outputs: []workflowv3.IROutput{},
		}}
		s.activeBuild = builder
		_, err := build(goja.Undefined(), builder.object(vm))
		s.activeBuild = nil
		builder.closed = true
		if err != nil {
			panic(err)
		}
		object := vm.NewObject()
		s.workflows[object] = builder.ir
		return object
	})
	mustSet(vm, exports, "toIR", func(call goja.FunctionCall) goja.Value {
		ir := s.mustWorkflow(vm, call.Argument(0))
		return jsonValue(vm, ir)
	})
	mustSet(vm, exports, "validate", func(call goja.FunctionCall) goja.Value {
		ir := s.mustWorkflow(vm, call.Argument(0))
		if err := workflowv3.ValidateIR(ir, s.catalog); err != nil {
			return vm.ToValue(map[string]any{"ok": false, "errors": []string{err.Error()}})
		}
		return vm.ToValue(map[string]any{"ok": true, "errors": []string{}})
	})
	mustSet(vm, exports, "digest", func(call goja.FunctionCall) goja.Value {
		ir := s.mustWorkflow(vm, call.Argument(0))
		digest, err := workflowv3.Digest(ir)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		return vm.ToValue(digest)
	})
	mustSet(vm, exports, "compile", func(call goja.FunctionCall) goja.Value {
		ir := s.mustWorkflow(vm, call.Argument(0))
		plan, err := workflowv3.Compile(ir, s.catalog)
		if err != nil {
			panic(vm.NewGoError(err))
		}
		object := jsonValue(vm, plan).ToObject(vm)
		s.plans[object] = plan
		s.planIR[object] = ir
		return object
	})
}

func (s *authoringState) descriptorLoader(module DescriptorModule) require.ModuleLoader {
	return func(vm *goja.Runtime, moduleObject *goja.Object) {
		exports := moduleObject.Get("exports").ToObject(vm)
		names := make([]string, 0, len(module.Factories))
		for name := range module.Factories {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			name := name
			key := module.Factories[name]
			spec, ok := s.catalog.Lookup(key)
			if !ok {
				panic(vm.NewGoError(fmt.Errorf("descriptor %s references unknown task %s@%s", name, key.Kind, key.Version)))
			}
			mustSet(vm, exports, name, func(call goja.FunctionCall) goja.Value {
				if s.activeBuild == nil {
					panic(vm.NewTypeError("task descriptors may only be created inside workflow.define"))
				}
				options := call.Argument(0).ToObject(vm)
				bindings := make(map[string]workflowv3.ValueRef, len(spec.Inputs))
				for input := range spec.Inputs {
					value := options.Get(input)
					if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
						panic(vm.NewTypeError("task %s requires input %s", name, input))
					}
					ref, ok := s.refs[value.ToObject(vm)]
					if !ok {
						panic(vm.NewTypeError("task %s input %s must be a workflow value", name, input))
					}
					bindings[input] = ref
				}
				for _, option := range options.Keys() {
					if _, ok := spec.Inputs[option]; !ok {
						panic(vm.NewTypeError("task %s has unknown input %s", name, option))
					}
				}
				object := vm.NewObject()
				s.tasks[object] = taskInvocation{key: key, bindings: bindings}
				return object
			})
		}
	}
}

func (b *planBuilder) object(vm *goja.Runtime) *goja.Object {
	object := vm.NewObject()
	mustSet(vm, object, "budget", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		account := strings.TrimSpace(call.Argument(0).String())
		options := call.Argument(1).ToObject(vm)
		limits := budgetAmountsFromValue(vm, options.Get("limits"))
		policyDigest := strings.TrimSpace(options.Get("policyDigest").String())
		for _, existing := range b.ir.Budgets {
			if existing.Account == account {
				panic(vm.NewTypeError("budget account %s is already defined", account))
			}
		}
		b.ir.Budgets = append(b.ir.Budgets, workflowv3.BudgetAccount{
			Account: account, Limits: limits, PolicyDigest: policyDigest,
		})
		sort.Slice(b.ir.Budgets, func(i, j int) bool { return b.ir.Budgets[i].Account < b.ir.Budgets[j].Account })
		return object
	})
	mustSet(vm, object, "input", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		name := strings.TrimSpace(call.Argument(0).String())
		options := call.Argument(1).ToObject(vm)
		schema := strings.TrimSpace(options.Get("schema").String())
		if name == "" || schema == "" {
			panic(vm.NewTypeError("input name and schema are required"))
		}
		for _, input := range b.ir.Inputs {
			if input.Name == name {
				panic(vm.NewTypeError("input %s is already defined", name))
			}
		}
		b.ir.Inputs = append(b.ir.Inputs, workflowv3.IRInput{Name: name, Schema: schema})
		return b.newRef(vm, workflowv3.ValueRef{Source: "input", Name: name, Schema: schema})
	})
	mustSet(vm, object, "inputSet", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		name := strings.TrimSpace(call.Argument(0).String())
		options := call.Argument(1).ToObject(vm)
		itemSchema := strings.TrimSpace(options.Get("itemSchema").String())
		manifestSchema := strings.TrimSpace(options.Get("manifestSchema").String())
		if name == "" || itemSchema == "" || manifestSchema == "" {
			panic(vm.NewTypeError("set input name, itemSchema, and manifestSchema are required"))
		}
		for _, input := range b.ir.Inputs {
			if input.Name == name {
				panic(vm.NewTypeError("input %s is already defined", name))
			}
		}
		for _, input := range b.ir.SetInputs {
			if input.Name == name {
				panic(vm.NewTypeError("set input %s is already defined", name))
			}
		}
		b.ir.SetInputs = append(b.ir.SetInputs, workflowv3.IRSetInput{
			Name: name, ItemSchema: itemSchema, ManifestSchema: manifestSchema,
		})
		return b.newSet(vm, workflowv3.SetRef{
			Source: "set-input", Name: name, ItemSchema: itemSchema, ManifestSchema: manifestSchema,
		})
	})
	mustSet(vm, object, "gate", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		key := workflowv3.NodeKey(strings.TrimSpace(call.Argument(0).String()))
		options := call.Argument(1).ToObject(vm)
		policy := workflowv3.GatePolicy{
			DecisionSchema: strings.TrimSpace(options.Get("schema").String()),
			RequiredRole:   strings.TrimSpace(options.Get("requiredRole").String()),
			OnReject:       workflowv3.GateFailRun, OnExpire: workflowv3.GateFailRun,
		}
		if value := options.Get("timeoutMs"); value != nil && !goja.IsUndefined(value) && !goja.IsNull(value) {
			policy.TimeoutMillis = value.ToInteger()
		}
		if value := options.Get("onReject"); value != nil && !goja.IsUndefined(value) && !goja.IsNull(value) {
			policy.OnReject = strings.TrimSpace(value.String())
		}
		if value := options.Get("onExpire"); value != nil && !goja.IsUndefined(value) && !goja.IsNull(value) {
			policy.OnExpire = strings.TrimSpace(value.String())
		}
		if err := workflowv3.ValidateGatePolicy(policy); err != nil {
			panic(vm.NewTypeError("invalid gate policy: %s", err))
		}
		for _, node := range b.ir.Nodes {
			if node.Key == key {
				panic(vm.NewTypeError("gate key %s conflicts with a node", key))
			}
		}
		for _, gate := range b.ir.Gates {
			if gate.Key == key {
				panic(vm.NewTypeError("gate %s is already defined", key))
			}
		}
		gate := workflowv3.IRGate{Key: key, Policy: policy}
		b.ir.Gates = append(b.ir.Gates, gate)
		index := len(b.ir.Gates) - 1
		if configure, ok := goja.AssertFunction(call.Argument(2)); ok {
			gateBuilder := vm.NewObject()
			mustSet(vm, gateBuilder, "after", func(afterCall goja.FunctionCall) goja.Value {
				dependency, ok := b.state.jobs[afterCall.Argument(0).ToObject(vm)]
				if !ok {
					panic(vm.NewTypeError("gate after requires a job from this workflow"))
				}
				b.ir.Gates[index].DependsOn = appendUniqueNode(b.ir.Gates[index].DependsOn, dependency)
				return gateBuilder
			})
			if _, err := configure(goja.Undefined(), gateBuilder); err != nil {
				panic(err)
			}
		}
		return b.newRef(vm, workflowv3.ValueRef{
			Source: "gate-output", GateKey: key, Schema: policy.DecisionSchema,
		})
	})
	mustSet(vm, object, "task", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		key := workflowv3.NodeKey(strings.TrimSpace(call.Argument(0).String()))
		if key == "" {
			panic(vm.NewTypeError("node key is required"))
		}
		for _, node := range b.ir.Nodes {
			if node.Key == key {
				panic(vm.NewTypeError("node %s is already defined", key))
			}
		}
		invocation, ok := b.state.tasks[call.Argument(1).ToObject(vm)]
		if !ok {
			panic(vm.NewTypeError("task requires a descriptor from an authoring module"))
		}
		node := workflowv3.IRNode{Key: key, Task: invocation.key, Bindings: invocation.bindings}
		b.ir.Nodes = append(b.ir.Nodes, node)
		index := len(b.ir.Nodes) - 1
		job := b.jobObject(vm, key, invocation.key)
		if build, ok := goja.AssertFunction(call.Argument(2)); ok {
			jobBuilder := vm.NewObject()
			mustSet(vm, jobBuilder, "after", func(afterCall goja.FunctionCall) goja.Value {
				dependency, ok := b.state.jobs[afterCall.Argument(0).ToObject(vm)]
				if !ok {
					panic(vm.NewTypeError("after requires a job from this workflow"))
				}
				b.ir.Nodes[index].DependsOn = appendUniqueNode(b.ir.Nodes[index].DependsOn, dependency)
				return jobBuilder
			})
			mustSet(vm, jobBuilder, "budget", func(option goja.FunctionCall) goja.Value {
				b.ir.Nodes[index].Budget = budgetClaimFromValue(vm, option.Argument(0))
				return jobBuilder
			})
			mustSet(vm, jobBuilder, "isolation", func(option goja.FunctionCall) goja.Value {
				b.ir.Nodes[index].Isolation = isolationPolicyFromValue(vm, option.Argument(0))
				return jobBuilder
			})
			if _, err := build(goja.Undefined(), jobBuilder); err != nil {
				panic(err)
			}
		}
		return job
	})
	mustSet(vm, object, "map", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		key := strings.TrimSpace(call.Argument(0).String())
		source, ok := b.state.sets[call.Argument(1).ToObject(vm)]
		build, buildOK := goja.AssertFunction(call.Argument(2))
		if key == "" || !ok || !buildOK {
			panic(vm.NewTypeError("map requires a key, set, and item task callback"))
		}
		for _, node := range b.ir.Nodes {
			if string(node.Key) == key {
				panic(vm.NewTypeError("map key %s conflicts with a node", key))
			}
		}
		for _, mapped := range b.ir.Maps {
			if mapped.Key == key {
				panic(vm.NewTypeError("map %s is already defined", key))
			}
		}
		item := b.newRef(vm, workflowv3.ValueRef{
			Source: "map-item", MapKey: key, Schema: source.ItemSchema,
		})
		value, err := build(goja.Undefined(), item)
		if err != nil {
			panic(err)
		}
		invocation, ok := b.state.tasks[value.ToObject(vm)]
		if !ok {
			panic(vm.NewTypeError("map item callback must return a task descriptor"))
		}
		policy := workflowv3.MapPolicy{
			PageSize: 64, MaxItems: 10000, MaxMaterializedAhead: 128,
		}
		var budget *workflowv3.BudgetClaim
		var isolation *workflowv3.IsolationPolicy
		if configure, configureOK := goja.AssertFunction(call.Argument(3)); configureOK {
			mapBuilder := vm.NewObject()
			mustSet(vm, mapBuilder, "pageSize", func(option goja.FunctionCall) goja.Value {
				policy.PageSize = int(option.Argument(0).ToInteger())
				return mapBuilder
			})
			mustSet(vm, mapBuilder, "maxItems", func(option goja.FunctionCall) goja.Value {
				policy.MaxItems = int(option.Argument(0).ToInteger())
				return mapBuilder
			})
			mustSet(vm, mapBuilder, "maxMaterializedAhead", func(option goja.FunctionCall) goja.Value {
				policy.MaxMaterializedAhead = int(option.Argument(0).ToInteger())
				return mapBuilder
			})
			mustSet(vm, mapBuilder, "budget", func(option goja.FunctionCall) goja.Value {
				budget = budgetClaimFromValue(vm, option.Argument(0))
				return mapBuilder
			})
			mustSet(vm, mapBuilder, "isolation", func(option goja.FunctionCall) goja.Value {
				isolation = isolationPolicyFromValue(vm, option.Argument(0))
				return mapBuilder
			})
			if _, err := configure(goja.Undefined(), mapBuilder); err != nil {
				panic(err)
			}
		}
		b.ir.Maps = append(b.ir.Maps, workflowv3.IRMap{
			Key: key, Source: source, ItemTask: invocation.key,
			Bindings: invocation.bindings, Policy: policy, Budget: budget, Isolation: isolation,
		})
		spec, found := b.state.catalog.Lookup(invocation.key)
		if !found || len(spec.Outputs) != 1 {
			panic(vm.NewTypeError("map item task must declare exactly one output"))
		}
		outputSchema := ""
		for _, schema := range spec.Outputs {
			outputSchema = schema
		}
		return b.newSet(vm, workflowv3.SetRef{
			Source: "map-output", MapKey: key, ItemSchema: outputSchema,
			ManifestSchema: workflowv3.ItemManifestSchemaV1,
		})
	})
	mustSet(vm, object, "reduce", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		key := strings.TrimSpace(call.Argument(0).String())
		source, ok := b.state.sets[call.Argument(1).ToObject(vm)]
		build, buildOK := goja.AssertFunction(call.Argument(2))
		if key == "" || !ok || !buildOK {
			panic(vm.NewTypeError("reduce requires a key, set, and partition task callback"))
		}
		for _, node := range b.ir.Nodes {
			if string(node.Key) == key {
				panic(vm.NewTypeError("reduction key %s conflicts with a node", key))
			}
		}
		for _, mapped := range b.ir.Maps {
			if mapped.Key == key {
				panic(vm.NewTypeError("reduction key %s conflicts with a map", key))
			}
		}
		for _, reduced := range b.ir.Reductions {
			if reduced.Key == key {
				panic(vm.NewTypeError("reduction %s is already defined", key))
			}
		}
		partition := b.newRef(vm, workflowv3.ValueRef{
			Source: "reduction-partition", ReduceKey: key,
			Schema: workflowv3.ReductionPartitionSchemaV1,
		})
		value, err := build(goja.Undefined(), partition)
		if err != nil {
			panic(err)
		}
		invocation, ok := b.state.tasks[value.ToObject(vm)]
		if !ok {
			panic(vm.NewTypeError("reduce callback must return a task descriptor"))
		}
		policy := workflowv3.ReducePolicy{FanIn: 16, MaxLevels: 8}
		var budget *workflowv3.BudgetClaim
		var isolation *workflowv3.IsolationPolicy
		if configure, configureOK := goja.AssertFunction(call.Argument(3)); configureOK {
			reduceBuilder := vm.NewObject()
			mustSet(vm, reduceBuilder, "fanIn", func(option goja.FunctionCall) goja.Value {
				policy.FanIn = int(option.Argument(0).ToInteger())
				return reduceBuilder
			})
			mustSet(vm, reduceBuilder, "maxLevels", func(option goja.FunctionCall) goja.Value {
				policy.MaxLevels = int(option.Argument(0).ToInteger())
				return reduceBuilder
			})
			mustSet(vm, reduceBuilder, "budget", func(option goja.FunctionCall) goja.Value {
				budget = budgetClaimFromValue(vm, option.Argument(0))
				return reduceBuilder
			})
			mustSet(vm, reduceBuilder, "isolation", func(option goja.FunctionCall) goja.Value {
				isolation = isolationPolicyFromValue(vm, option.Argument(0))
				return reduceBuilder
			})
			if _, err := configure(goja.Undefined(), reduceBuilder); err != nil {
				panic(err)
			}
		}
		b.ir.Reductions = append(b.ir.Reductions, workflowv3.IRReduce{
			Key: key, Source: source, PartitionTask: invocation.key,
			Bindings: invocation.bindings, Policy: policy, Budget: budget, Isolation: isolation,
		})
		spec, found := b.state.catalog.Lookup(invocation.key)
		if !found || len(spec.Outputs) != 1 {
			panic(vm.NewTypeError("reduction task must declare exactly one output"))
		}
		outputSchema := ""
		for _, schema := range spec.Outputs {
			outputSchema = schema
		}
		return b.newRef(vm, workflowv3.ValueRef{
			Source: "reduction-output", ReduceKey: key, Schema: outputSchema,
		})
	})
	mustSet(vm, object, "output", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		name := strings.TrimSpace(call.Argument(0).String())
		ref, ok := b.state.refs[call.Argument(1).ToObject(vm)]
		if name == "" || !ok {
			panic(vm.NewTypeError("output requires a name and workflow value"))
		}
		for _, output := range b.ir.Outputs {
			if output.Name == name {
				panic(vm.NewTypeError("output %s is already defined", name))
			}
		}
		b.ir.Outputs = append(b.ir.Outputs, workflowv3.IROutput{Name: name, Value: ref})
		return object
	})
	mustSet(vm, object, "outputSet", func(call goja.FunctionCall) goja.Value {
		b.ensureOpen(vm)
		name := strings.TrimSpace(call.Argument(0).String())
		set, ok := b.state.sets[call.Argument(1).ToObject(vm)]
		if name == "" || !ok {
			panic(vm.NewTypeError("set output requires a name and workflow set"))
		}
		for _, output := range b.ir.Outputs {
			if output.Name == name {
				panic(vm.NewTypeError("output %s is already defined", name))
			}
		}
		for _, output := range b.ir.SetOutputs {
			if output.Name == name {
				panic(vm.NewTypeError("set output %s is already defined", name))
			}
		}
		b.ir.SetOutputs = append(b.ir.SetOutputs, workflowv3.IRSetOutput{Name: name, Value: set})
		return object
	})
	return object
}

func (b *planBuilder) jobObject(vm *goja.Runtime, key workflowv3.NodeKey, taskKey workflowv3.TaskKey) *goja.Object {
	object := vm.NewObject()
	b.state.jobs[object] = key
	mustSet(vm, object, "output", func(call goja.FunctionCall) goja.Value {
		name := strings.TrimSpace(call.Argument(0).String())
		spec, _ := b.state.catalog.Lookup(taskKey)
		schema, ok := spec.Outputs[name]
		if !ok {
			panic(vm.NewTypeError("task %s@%s has no output %s", taskKey.Kind, taskKey.Version, name))
		}
		return b.newRef(vm, workflowv3.ValueRef{Source: "node-output", NodeKey: key, Port: name, Schema: schema})
	})
	return object
}

func (b *planBuilder) newRef(vm *goja.Runtime, ref workflowv3.ValueRef) *goja.Object {
	object := vm.NewObject()
	b.state.refs[object] = ref
	return object
}

func (b *planBuilder) newSet(vm *goja.Runtime, ref workflowv3.SetRef) *goja.Object {
	object := vm.NewObject()
	b.state.sets[object] = ref
	return object
}

func (b *planBuilder) ensureOpen(vm *goja.Runtime) {
	if b.closed {
		panic(vm.NewTypeError("workflow builder is closed"))
	}
}

func (s *authoringState) mustWorkflow(vm *goja.Runtime, value goja.Value) workflowv3.WorkflowIR {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		panic(vm.NewTypeError("workflow value is required"))
	}
	ir, ok := s.workflows[value.ToObject(vm)]
	if !ok {
		panic(vm.NewTypeError("value is not a workflow from this runtime"))
	}
	return ir
}

func isolationPolicyFromValue(vm *goja.Runtime, value goja.Value) *workflowv3.IsolationPolicy {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		panic(vm.NewTypeError("isolation policy is required"))
	}
	object := value.ToObject(vm)
	policy := &workflowv3.IsolationPolicy{
		Class:            strings.TrimSpace(object.Get("class").String()),
		WallTimeMillis:   safeIntegerProperty(vm, object, "wallTimeMillis"),
		CPUTimeMillis:    safeIntegerProperty(vm, object, "cpuTimeMillis"),
		MemoryBytes:      safeIntegerProperty(vm, object, "memoryBytes"),
		MaxProcesses:     safeIntegerProperty(vm, object, "maxProcesses"),
		MaxOutputBytes:   safeIntegerProperty(vm, object, "maxOutputBytes"),
		MaxOutputFiles:   int(safeIntegerProperty(vm, object, "maxOutputFiles")),
		MaxProtocolBytes: safeIntegerProperty(vm, object, "maxProtocolBytes"),
	}
	if err := workflowv3.ValidateIsolationPolicy(*policy); err != nil {
		panic(vm.NewTypeError("invalid isolation policy: %s", err))
	}
	return policy
}

func safeIntegerProperty(vm *goja.Runtime, object *goja.Object, name string) int64 {
	value := object.Get(name)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return 0
	}
	number := value.ToFloat()
	if number < 0 || number > 9007199254740991 || math.Trunc(number) != number {
		panic(vm.NewTypeError("isolation %s must be a nonnegative safe integer", name))
	}
	return int64(number)
}

func budgetClaimFromValue(vm *goja.Runtime, value goja.Value) *workflowv3.BudgetClaim {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		panic(vm.NewTypeError("budget claim is required"))
	}
	object := value.ToObject(vm)
	claim := &workflowv3.BudgetClaim{
		Account:     strings.TrimSpace(object.Get("account").String()),
		Reserve:     budgetAmountsFromValue(vm, object.Get("reserve")),
		OnExhausted: strings.TrimSpace(object.Get("onExhausted").String()),
	}
	if approval := object.Get("approvalGate"); approval != nil &&
		!goja.IsUndefined(approval) && !goja.IsNull(approval) {
		claim.ApprovalGate = workflowv3.NodeKey(strings.TrimSpace(approval.String()))
	}
	if err := workflowv3.ValidateBudgetClaim(*claim); err != nil {
		panic(vm.NewTypeError("invalid budget claim: %s", err))
	}
	return claim
}

func budgetAmountsFromValue(vm *goja.Runtime, value goja.Value) []workflowv3.BudgetAmount {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		panic(vm.NewTypeError("budget dimensions are required"))
	}
	object := value.ToObject(vm)
	keys := object.Keys()
	sort.Strings(keys)
	amounts := make([]workflowv3.BudgetAmount, 0, len(keys))
	for _, dimension := range keys {
		number := object.Get(dimension).ToFloat()
		if number < 0 || number > 9007199254740991 || math.Trunc(number) != number {
			panic(vm.NewTypeError("budget dimension %s must be a nonnegative safe integer", dimension))
		}
		amounts = append(amounts, workflowv3.BudgetAmount{Dimension: dimension, Units: int64(number)})
	}
	return amounts
}

func appendUniqueNode(nodes []workflowv3.NodeKey, key workflowv3.NodeKey) []workflowv3.NodeKey {
	for _, existing := range nodes {
		if existing == key {
			return nodes
		}
	}
	return append(nodes, key)
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

func TypeScript() string {
	return `declare module "workflow" {
  export interface ValueRef<T = unknown> { readonly schema?: string }
  export interface SetRef<T = unknown> { readonly itemSchema?: string }
  export interface JobRef<T = unknown> {
    output(name: string): ValueRef<T>;
  }
  export type BudgetDimension =
    | "requests" | "input_tokens" | "output_tokens"
    | "embedding_tokens" | "input_bytes" | "output_bytes"
    | "cost_microunits";
  export type BudgetAmounts = Partial<Record<BudgetDimension, number>>;
  export interface BudgetClaim {
    account: string;
    reserve: BudgetAmounts;
    onExhausted: "fail-run" | "block" | "require-approval";
    approvalGate?: string;
  }
  export interface IsolationPolicy {
    class: "in-process.trusted" | "subprocess.restricted";
    wallTimeMillis?: number;
    cpuTimeMillis?: number;
    memoryBytes?: number;
    maxProcesses?: number;
    maxOutputBytes?: number;
    maxOutputFiles?: number;
    maxProtocolBytes?: number;
  }
  export interface GateBuilder {
    after(job: JobRef): GateBuilder;
  }
  export interface JobBuilder {
    after(job: JobRef): JobBuilder;
    budget(claim: BudgetClaim): JobBuilder;
    isolation(policy: IsolationPolicy): JobBuilder;
  }
  export interface MapBuilder {
    pageSize(value: number): MapBuilder;
    maxItems(value: number): MapBuilder;
    maxMaterializedAhead(value: number): MapBuilder;
    budget(claim: BudgetClaim): MapBuilder;
    isolation(policy: IsolationPolicy): MapBuilder;
  }
  export interface ReduceBuilder {
    fanIn(value: number): ReduceBuilder;
    maxLevels(value: number): ReduceBuilder;
    budget(claim: BudgetClaim): ReduceBuilder;
    isolation(policy: IsolationPolicy): ReduceBuilder;
  }
  export interface PlanBuilder {
    budget(
      account: string,
      options: {limits: BudgetAmounts; policyDigest: string},
    ): PlanBuilder;
    gate<TDecision = unknown>(
      name: string,
      options: {
        schema: string;
        timeoutMs?: number;
        requiredRole: string;
        onReject?: "fail-run" | "cancel-branch";
        onExpire?: "fail-run" | "cancel-branch";
      },
      configure?: (gate: GateBuilder) => void,
    ): ValueRef<TDecision>;
    input<T = unknown>(
      name: string,
      options: {schema: string},
    ): ValueRef<T>;
    inputSet<T = unknown>(
      name: string,
      options: {itemSchema: string; manifestSchema: string},
    ): SetRef<T>;
    task(
      name: string,
      task: unknown,
      build?: (job: JobBuilder) => void,
    ): JobRef;
    map<I, O>(
      name: string,
      source: SetRef<I>,
      task: (item: ValueRef<I>) => unknown,
      build?: (map: MapBuilder) => void,
    ): SetRef<O>;
    reduce<I, O>(
      name: string,
      source: SetRef<I>,
      task: (partition: ValueRef<readonly I[]>) => unknown,
      build?: (reduce: ReduceBuilder) => void,
    ): ValueRef<O>;
    output(name: string, value: ValueRef): PlanBuilder;
    outputSet(name: string, value: SetRef): PlanBuilder;
  }
  export interface Workflow {}
  export interface WorkflowPlanV3 { readonly schema: "scraper-workflow-plan/v3" }
  export function define(
    name: string,
    build: (plan: PlanBuilder) => void,
  ): Workflow;
  export function toIR(value: Workflow): unknown;
  export function validate(value: Workflow): {ok: boolean; errors: string[]};
  export function digest(value: Workflow): string;
  export function compile(value: Workflow): WorkflowPlanV3;
}
`
}

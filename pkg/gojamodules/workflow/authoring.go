package workflowmodule

import (
	"context"
	"encoding/json"
	"fmt"
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
		builder := &planBuilder{state: s, ir: workflowv3.WorkflowIR{Schema: workflowv3.IRSchema, Name: name}}
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
			if _, err := build(goja.Undefined(), jobBuilder); err != nil {
				panic(err)
			}
		}
		return job
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
  export interface JobRef<T = unknown> {
    output(name: string): ValueRef<T>;
  }
  export interface JobBuilder { after(job: JobRef): JobBuilder }
  export interface PlanBuilder {
    input(name: string, options: {schema: string}): ValueRef;
    task(
      name: string,
      task: unknown,
      build?: (job: JobBuilder) => void,
    ): JobRef;
    output(name: string, value: ValueRef): PlanBuilder;
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

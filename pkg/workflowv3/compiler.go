package workflowv3

import (
	"fmt"
	"sort"
	"strings"
)

func ValidateIR(ir WorkflowIR, catalog *Catalog) error {
	if ir.Schema != IRSchema {
		return fmt.Errorf("workflow IR schema must be %q", IRSchema)
	}
	if strings.TrimSpace(ir.Name) == "" {
		return fmt.Errorf("workflow name is required")
	}
	if catalog == nil {
		return fmt.Errorf("task catalog is required")
	}

	budgetAccounts := make(map[string]BudgetAccount, len(ir.Budgets))
	previousAccount := ""
	for _, account := range ir.Budgets {
		if err := ValidateBudgetAccount(account); err != nil {
			return err
		}
		if account.Account <= previousAccount {
			return fmt.Errorf("budget accounts must be strictly sorted and unique")
		}
		budgetAccounts[account.Account] = account
		previousAccount = account.Account
	}

	budgetGateKeys := map[NodeKey]struct{}{}
	budgetGateOwners := map[NodeKey]string{}
	registerBudgetGate := func(gate NodeKey, owner string) error {
		if gate == "" {
			return nil
		}
		if existing, duplicate := budgetGateOwners[gate]; duplicate {
			return fmt.Errorf("budget approval gate %q is shared by %s and %s; each compiled claim requires a dedicated gate", gate, existing, owner)
		}
		budgetGateOwners[gate] = owner
		budgetGateKeys[gate] = struct{}{}
		return nil
	}
	for _, node := range ir.Nodes {
		if node.Budget != nil {
			if err := registerBudgetGate(node.Budget.ApprovalGate, "node "+string(node.Key)); err != nil {
				return err
			}
		}
	}
	for _, mapped := range ir.Maps {
		if mapped.Budget != nil {
			if err := registerBudgetGate(mapped.Budget.ApprovalGate, "map "+mapped.Key); err != nil {
				return err
			}
		}
	}
	for _, reduced := range ir.Reductions {
		if reduced.Budget != nil {
			if err := registerBudgetGate(reduced.Budget.ApprovalGate, "reduction "+reduced.Key); err != nil {
				return err
			}
		}
	}

	inputSchemas := make(map[string]string, len(ir.Inputs))
	for _, input := range ir.Inputs {
		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Schema) == "" {
			return fmt.Errorf("workflow input name and schema are required")
		}
		if _, exists := inputSchemas[input.Name]; exists {
			return fmt.Errorf("duplicate workflow input %q", input.Name)
		}
		inputSchemas[input.Name] = input.Schema
	}

	setInputs := make(map[string]SetRef, len(ir.SetInputs))
	for _, input := range ir.SetInputs {
		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.ItemSchema) == "" ||
			strings.TrimSpace(input.ManifestSchema) == "" {
			return fmt.Errorf("workflow set input name, item schema, and manifest schema are required")
		}
		if _, exists := inputSchemas[input.Name]; exists {
			return fmt.Errorf("workflow input %q is already defined as a value input", input.Name)
		}
		if _, exists := setInputs[input.Name]; exists {
			return fmt.Errorf("duplicate workflow set input %q", input.Name)
		}
		setInputs[input.Name] = SetRef{
			Source: "set-input", Name: input.Name, ItemSchema: input.ItemSchema,
			ManifestSchema: input.ManifestSchema,
		}
	}

	nodeSpecs := make(map[NodeKey]TaskSpec, len(ir.Nodes))
	for _, node := range ir.Nodes {
		if strings.TrimSpace(string(node.Key)) == "" {
			return fmt.Errorf("node key is required")
		}
		if _, exists := nodeSpecs[node.Key]; exists {
			return fmt.Errorf("duplicate node key %q", node.Key)
		}
		spec, ok := catalog.Lookup(node.Task)
		if !ok {
			return fmt.Errorf("node %q references unknown task %s@%s", node.Key, node.Task.Kind, node.Task.Version)
		}
		nodeSpecs[node.Key] = spec
	}

	reductionSchemas := make(map[string]string, len(ir.Reductions))
	for _, reduced := range ir.Reductions {
		if strings.TrimSpace(reduced.Key) == "" {
			return fmt.Errorf("reduction key is required")
		}
		if _, exists := reductionSchemas[reduced.Key]; exists {
			return fmt.Errorf("duplicate reduction key %q", reduced.Key)
		}
		spec, ok := catalog.Lookup(reduced.PartitionTask)
		if !ok {
			return fmt.Errorf("reduction %q references unknown task %s@%s", reduced.Key, reduced.PartitionTask.Kind, reduced.PartitionTask.Version)
		}
		if len(spec.Outputs) != 1 {
			return fmt.Errorf("reduction %q reducer must declare one output", reduced.Key)
		}
		for _, schema := range spec.Outputs {
			reductionSchemas[reduced.Key] = schema
		}
	}

	gateSchemas := make(map[NodeKey]string, len(ir.Gates))
	for _, gate := range ir.Gates {
		if strings.TrimSpace(string(gate.Key)) == "" {
			return fmt.Errorf("gate key is required")
		}
		if _, exists := nodeSpecs[gate.Key]; exists {
			return fmt.Errorf("gate key %q conflicts with a node key", gate.Key)
		}
		if _, exists := gateSchemas[gate.Key]; exists {
			return fmt.Errorf("duplicate gate key %q", gate.Key)
		}
		if err := ValidateGatePolicy(gate.Policy); err != nil {
			return fmt.Errorf("gate %q policy: %w", gate.Key, err)
		}
		seenDependencies := map[NodeKey]struct{}{}
		for _, dependency := range gate.DependsOn {
			if _, ok := nodeSpecs[dependency]; !ok {
				return fmt.Errorf("gate %q has unknown dependency %q", gate.Key, dependency)
			}
			if _, exists := seenDependencies[dependency]; exists {
				return fmt.Errorf("gate %q repeats dependency %q", gate.Key, dependency)
			}
			seenDependencies[dependency] = struct{}{}
		}
		gateSchemas[gate.Key] = gate.Policy.DecisionSchema
	}

	for _, node := range ir.Nodes {
		spec := nodeSpecs[node.Key]
		if _, err := compileBudgetClaim(node.Budget, spec.BudgetMaximum, budgetAccounts); err != nil {
			return fmt.Errorf("node %q budget: %w", node.Key, err)
		}
		compiledIsolation, err := CompileIsolation(node.Isolation, spec.IsolationMaximum, spec.IsolationExecutorDigest)
		if err != nil {
			return fmt.Errorf("node %q isolation: %w", node.Key, err)
		}
		if RequiresRestrictedIsolation(spec.Modules) && compiledIsolation.Effective.Class != IsolationSubprocessRestricted {
			return fmt.Errorf("node %q modules require restricted isolation", node.Key)
		}
		if node.Budget != nil && node.Budget.ApprovalGate != "" {
			if _, ok := gateSchemas[node.Budget.ApprovalGate]; !ok {
				return fmt.Errorf("node %q budget references unknown gate %q", node.Key, node.Budget.ApprovalGate)
			}
		}
		if len(node.Bindings) != len(spec.Inputs) {
			return fmt.Errorf("node %q has %d bindings, task requires %d", node.Key, len(node.Bindings), len(spec.Inputs))
		}
		for port, expectedSchema := range spec.Inputs {
			binding, ok := node.Bindings[port]
			if !ok {
				return fmt.Errorf("node %q is missing binding %q", node.Key, port)
			}
			if binding.Source == "gate-output" {
				if _, budgetOnly := budgetGateKeys[binding.GateKey]; budgetOnly {
					return fmt.Errorf("node %q cannot consume budget-activation gate %q", node.Key, binding.GateKey)
				}
			}
			actualSchema, err := refSchema(binding, inputSchemas, nodeSpecs, gateSchemas, reductionSchemas)
			if err != nil {
				return fmt.Errorf("node %q binding %q: %w", node.Key, port, err)
			}
			if actualSchema != expectedSchema || binding.Schema != expectedSchema {
				return fmt.Errorf("node %q binding %q schema %q does not match %q", node.Key, port, actualSchema, expectedSchema)
			}
		}
		seenDeps := map[NodeKey]struct{}{}
		for _, dependency := range node.DependsOn {
			if dependency == node.Key {
				return fmt.Errorf("node %q cannot depend on itself", node.Key)
			}
			if _, ok := nodeSpecs[dependency]; !ok {
				return fmt.Errorf("node %q has unknown dependency %q", node.Key, dependency)
			}
			if _, exists := seenDeps[dependency]; exists {
				return fmt.Errorf("node %q repeats dependency %q", node.Key, dependency)
			}
			seenDeps[dependency] = struct{}{}
		}
	}
	if err := validateAcyclic(ir.Nodes); err != nil {
		return err
	}
	if err := validateGateAcyclic(ir.Nodes, ir.Gates); err != nil {
		return err
	}

	mapOutputs := make(map[string]SetRef, len(ir.Maps))
	for _, mapped := range ir.Maps {
		if strings.TrimSpace(mapped.Key) == "" {
			return fmt.Errorf("map key is required")
		}
		if _, exists := nodeSpecs[NodeKey(mapped.Key)]; exists {
			return fmt.Errorf("map key %q conflicts with a node key", mapped.Key)
		}
		if _, exists := gateSchemas[NodeKey(mapped.Key)]; exists {
			return fmt.Errorf("map key %q conflicts with a gate key", mapped.Key)
		}
		if _, exists := mapOutputs[mapped.Key]; exists {
			return fmt.Errorf("duplicate map key %q", mapped.Key)
		}
		source, err := setRefSchema(mapped.Source, setInputs, mapOutputs)
		if err != nil {
			return fmt.Errorf("map %q source: %w", mapped.Key, err)
		}
		if mapped.Policy.PageSize < 1 || mapped.Policy.MaxItems < 1 ||
			mapped.Policy.MaxMaterializedAhead < mapped.Policy.PageSize ||
			mapped.Policy.PageSize > mapped.Policy.MaxItems {
			return fmt.Errorf("map %q has invalid expansion policy", mapped.Key)
		}
		spec, ok := catalog.Lookup(mapped.ItemTask)
		if !ok {
			return fmt.Errorf("map %q references unknown task %s@%s", mapped.Key, mapped.ItemTask.Kind, mapped.ItemTask.Version)
		}
		if len(spec.Outputs) != 1 {
			return fmt.Errorf("map %q item task must declare exactly one output", mapped.Key)
		}
		if _, err := compileBudgetClaim(mapped.Budget, spec.BudgetMaximum, budgetAccounts); err != nil {
			return fmt.Errorf("map %q budget: %w", mapped.Key, err)
		}
		compiledIsolation, err := CompileIsolation(mapped.Isolation, spec.IsolationMaximum, spec.IsolationExecutorDigest)
		if err != nil {
			return fmt.Errorf("map %q isolation: %w", mapped.Key, err)
		}
		if RequiresRestrictedIsolation(spec.Modules) && compiledIsolation.Effective.Class != IsolationSubprocessRestricted {
			return fmt.Errorf("map %q modules require restricted isolation", mapped.Key)
		}
		if mapped.Budget != nil && mapped.Budget.ApprovalGate != "" {
			if _, ok := gateSchemas[mapped.Budget.ApprovalGate]; !ok {
				return fmt.Errorf("map %q budget references unknown gate %q", mapped.Key, mapped.Budget.ApprovalGate)
			}
		}
		if len(mapped.Bindings) != len(spec.Inputs) {
			return fmt.Errorf("map %q has %d bindings, task requires %d", mapped.Key, len(mapped.Bindings), len(spec.Inputs))
		}
		itemBindings := 0
		for port, expectedSchema := range spec.Inputs {
			binding, ok := mapped.Bindings[port]
			if !ok {
				return fmt.Errorf("map %q is missing binding %q", mapped.Key, port)
			}
			var actualSchema string
			if binding.Source == "map-item" {
				if binding.MapKey != mapped.Key {
					return fmt.Errorf("map %q binding %q has wrong item owner %q", mapped.Key, port, binding.MapKey)
				}
				actualSchema = source.ItemSchema
				itemBindings++
			} else {
				if binding.Source == "gate-output" {
					if _, budgetOnly := budgetGateKeys[binding.GateKey]; budgetOnly {
						return fmt.Errorf("map %q cannot consume budget-activation gate %q", mapped.Key, binding.GateKey)
					}
				}
				actualSchema, err = refSchema(binding, inputSchemas, nodeSpecs, gateSchemas, reductionSchemas)
				if err != nil {
					return fmt.Errorf("map %q binding %q: %w", mapped.Key, port, err)
				}
			}
			if actualSchema != expectedSchema || binding.Schema != expectedSchema {
				return fmt.Errorf("map %q binding %q schema %q does not match %q", mapped.Key, port, actualSchema, expectedSchema)
			}
		}
		if itemBindings != 1 {
			return fmt.Errorf("map %q item task requires exactly one map-item binding", mapped.Key)
		}
		outputSchema := ""
		for _, schema := range spec.Outputs {
			outputSchema = schema
		}
		mapOutputs[mapped.Key] = SetRef{
			Source: "map-output", MapKey: mapped.Key, ItemSchema: outputSchema,
			ManifestSchema: ItemManifestSchemaV1,
		}
	}

	reductionOutputs := make(map[string]string, len(ir.Reductions))
	for _, reduced := range ir.Reductions {
		if strings.TrimSpace(reduced.Key) == "" {
			return fmt.Errorf("reduction key is required")
		}
		if _, exists := nodeSpecs[NodeKey(reduced.Key)]; exists {
			return fmt.Errorf("reduction key %q conflicts with a node key", reduced.Key)
		}
		if _, exists := gateSchemas[NodeKey(reduced.Key)]; exists {
			return fmt.Errorf("reduction key %q conflicts with a gate key", reduced.Key)
		}
		if _, exists := mapOutputs[reduced.Key]; exists {
			return fmt.Errorf("reduction key %q conflicts with a map key", reduced.Key)
		}
		if _, exists := reductionOutputs[reduced.Key]; exists {
			return fmt.Errorf("duplicate reduction key %q", reduced.Key)
		}
		source, err := setRefSchema(reduced.Source, setInputs, mapOutputs)
		if err != nil {
			return fmt.Errorf("reduction %q source: %w", reduced.Key, err)
		}
		if reduced.Policy.FanIn < 2 || reduced.Policy.MaxLevels < 1 {
			return fmt.Errorf("reduction %q has invalid reduction policy", reduced.Key)
		}
		spec, ok := catalog.Lookup(reduced.PartitionTask)
		if !ok {
			return fmt.Errorf("reduction %q references unknown task %s@%s", reduced.Key, reduced.PartitionTask.Kind, reduced.PartitionTask.Version)
		}
		if len(spec.Outputs) != 1 {
			return fmt.Errorf("reduction %q partition task must declare exactly one output", reduced.Key)
		}
		if _, err := compileBudgetClaim(reduced.Budget, spec.BudgetMaximum, budgetAccounts); err != nil {
			return fmt.Errorf("reduction %q budget: %w", reduced.Key, err)
		}
		compiledIsolation, err := CompileIsolation(reduced.Isolation, spec.IsolationMaximum, spec.IsolationExecutorDigest)
		if err != nil {
			return fmt.Errorf("reduction %q isolation: %w", reduced.Key, err)
		}
		if RequiresRestrictedIsolation(spec.Modules) && compiledIsolation.Effective.Class != IsolationSubprocessRestricted {
			return fmt.Errorf("reduction %q modules require restricted isolation", reduced.Key)
		}
		if reduced.Budget != nil && reduced.Budget.ApprovalGate != "" {
			if _, ok := gateSchemas[reduced.Budget.ApprovalGate]; !ok {
				return fmt.Errorf("reduction %q budget references unknown gate %q", reduced.Key, reduced.Budget.ApprovalGate)
			}
		}
		if len(reduced.Bindings) != len(spec.Inputs) {
			return fmt.Errorf("reduction %q has %d bindings, task requires %d", reduced.Key, len(reduced.Bindings), len(spec.Inputs))
		}
		partitionBindings := 0
		for port, expectedSchema := range spec.Inputs {
			binding, ok := reduced.Bindings[port]
			if !ok {
				return fmt.Errorf("reduction %q is missing binding %q", reduced.Key, port)
			}
			var actualSchema string
			if binding.Source == "reduction-partition" {
				if binding.ReduceKey != reduced.Key {
					return fmt.Errorf("reduction %q binding %q has wrong partition owner %q", reduced.Key, port, binding.ReduceKey)
				}
				actualSchema = ReductionPartitionSchemaV1
				partitionBindings++
			} else {
				if binding.Source == "gate-output" {
					if _, budgetOnly := budgetGateKeys[binding.GateKey]; budgetOnly {
						return fmt.Errorf("reduction %q cannot consume budget-activation gate %q", reduced.Key, binding.GateKey)
					}
				}
				actualSchema, err = refSchema(binding, inputSchemas, nodeSpecs, gateSchemas, reductionSchemas)
				if err != nil {
					return fmt.Errorf("reduction %q binding %q: %w", reduced.Key, port, err)
				}
			}
			if actualSchema != expectedSchema || binding.Schema != expectedSchema {
				return fmt.Errorf("reduction %q binding %q schema %q does not match %q", reduced.Key, port, actualSchema, expectedSchema)
			}
		}
		if partitionBindings != 1 {
			return fmt.Errorf("reduction %q task requires exactly one partition binding", reduced.Key)
		}
		outputSchema := ""
		for _, schema := range spec.Outputs {
			outputSchema = schema
		}
		if outputSchema != source.ItemSchema {
			return fmt.Errorf("reduction %q output schema %q must equal source item schema %q", reduced.Key, outputSchema, source.ItemSchema)
		}
		reductionOutputs[reduced.Key] = outputSchema
	}

	outputNames := map[string]struct{}{}
	for _, output := range ir.Outputs {
		if strings.TrimSpace(output.Name) == "" {
			return fmt.Errorf("workflow output name is required")
		}
		if _, exists := outputNames[output.Name]; exists {
			return fmt.Errorf("duplicate workflow output %q", output.Name)
		}
		outputNames[output.Name] = struct{}{}
		var actual string
		var err error
		if output.Value.Source == "gate-output" {
			if _, budgetOnly := budgetGateKeys[output.Value.GateKey]; budgetOnly {
				return fmt.Errorf("workflow output %q cannot expose budget-activation gate %q", output.Name, output.Value.GateKey)
			}
		}
		if output.Value.Source == "reduction-output" {
			schema, exists := reductionOutputs[output.Value.ReduceKey]
			if !exists {
				return fmt.Errorf("workflow output %q: unknown reduction %q", output.Name, output.Value.ReduceKey)
			}
			actual = schema
		} else {
			actual, err = refSchema(output.Value, inputSchemas, nodeSpecs, gateSchemas, reductionSchemas)
			if err != nil {
				return fmt.Errorf("workflow output %q: %w", output.Name, err)
			}
		}
		if actual != output.Value.Schema {
			return fmt.Errorf("workflow output %q schema mismatch", output.Name)
		}
	}
	for _, output := range ir.SetOutputs {
		if strings.TrimSpace(output.Name) == "" {
			return fmt.Errorf("workflow set output name is required")
		}
		if _, exists := outputNames[output.Name]; exists {
			return fmt.Errorf("duplicate workflow output %q", output.Name)
		}
		outputNames[output.Name] = struct{}{}
		actual, err := setRefSchema(output.Value, setInputs, mapOutputs)
		if err != nil {
			return fmt.Errorf("workflow set output %q: %w", output.Name, err)
		}
		if actual.ItemSchema != output.Value.ItemSchema ||
			actual.ManifestSchema != output.Value.ManifestSchema {
			return fmt.Errorf("workflow set output %q schema mismatch", output.Name)
		}
	}
	if len(ir.Outputs) == 0 && len(ir.SetOutputs) == 0 {
		return fmt.Errorf("workflow requires at least one output")
	}
	return nil
}

func Compile(ir WorkflowIR, catalog *Catalog) (WorkflowPlan, error) {
	if err := ValidateIR(ir, catalog); err != nil {
		return WorkflowPlan{}, err
	}
	irDigest, err := Digest(ir)
	if err != nil {
		return WorkflowPlan{}, err
	}
	catalogDigest, err := catalog.Digest()
	if err != nil {
		return WorkflowPlan{}, err
	}
	plan := WorkflowPlan{
		Schema:        PlanSchema,
		Name:          ir.Name,
		IRDigest:      irDigest,
		CatalogDigest: catalogDigest,
		Inputs:        append([]IRInput{}, ir.Inputs...),
		SetInputs:     append([]IRSetInput(nil), ir.SetInputs...),
		Budgets:       cloneBudgetAccounts(ir.Budgets),
		Nodes:         make([]PlanNode, 0, len(ir.Nodes)),
		Gates:         make([]PlanGate, 0, len(ir.Gates)),
		Outputs:       append([]IROutput{}, ir.Outputs...),
		SetOutputs:    append([]IRSetOutput(nil), ir.SetOutputs...),
	}
	budgetAccounts := make(map[string]BudgetAccount, len(ir.Budgets))
	for _, account := range ir.Budgets {
		budgetAccounts[account.Account] = account
	}
	for _, node := range ir.Nodes {
		spec, _ := catalog.Lookup(node.Task)
		budget, _ := compileBudgetClaim(node.Budget, spec.BudgetMaximum, budgetAccounts)
		isolation, _ := CompileIsolation(node.Isolation, spec.IsolationMaximum, spec.IsolationExecutorDigest)
		plan.Nodes = append(plan.Nodes, PlanNode{
			Key:            node.Key,
			Implementation: spec.Identity,
			Bindings:       cloneBindings(node.Bindings),
			DependsOn:      append([]NodeKey(nil), node.DependsOn...),
			InputSchemas:   cloneStringMap(spec.Inputs),
			OutputSchemas:  cloneStringMap(spec.Outputs),
			Modules:        append([]string(nil), spec.Modules...),
			ResourceClass:  spec.ResourceClass,
			Retry:          spec.Retry,
			Budget:         budget,
			Isolation:      &isolation,
		})
	}
	for _, mapped := range ir.Maps {
		spec, _ := catalog.Lookup(mapped.ItemTask)
		budget, _ := compileBudgetClaim(mapped.Budget, spec.BudgetMaximum, budgetAccounts)
		isolation, _ := CompileIsolation(mapped.Isolation, spec.IsolationMaximum, spec.IsolationExecutorDigest)
		plan.Maps = append(plan.Maps, PlanMap{
			Key: mapped.Key, Source: mapped.Source, Implementation: spec.Identity,
			Bindings:     cloneBindings(mapped.Bindings),
			InputSchemas: cloneStringMap(spec.Inputs), OutputSchemas: cloneStringMap(spec.Outputs),
			Modules: append([]string(nil), spec.Modules...), ResourceClass: spec.ResourceClass,
			Retry: spec.Retry, Policy: mapped.Policy, Budget: budget, Isolation: &isolation,
		})
	}
	for _, reduced := range ir.Reductions {
		spec, _ := catalog.Lookup(reduced.PartitionTask)
		budget, _ := compileBudgetClaim(reduced.Budget, spec.BudgetMaximum, budgetAccounts)
		isolation, _ := CompileIsolation(reduced.Isolation, spec.IsolationMaximum, spec.IsolationExecutorDigest)
		plan.Reductions = append(plan.Reductions, PlanReduce{
			Key: reduced.Key, Source: reduced.Source, Implementation: spec.Identity,
			Bindings:     cloneBindings(reduced.Bindings),
			InputSchemas: cloneStringMap(spec.Inputs), OutputSchemas: cloneStringMap(spec.Outputs),
			Modules: append([]string(nil), spec.Modules...), ResourceClass: spec.ResourceClass,
			Retry: spec.Retry, Policy: reduced.Policy, Budget: budget, Isolation: &isolation,
		})
	}
	budgetGates := map[NodeKey]struct{}{}
	for _, node := range ir.Nodes {
		if node.Budget != nil && node.Budget.ApprovalGate != "" {
			budgetGates[node.Budget.ApprovalGate] = struct{}{}
		}
	}
	for _, mapped := range ir.Maps {
		if mapped.Budget != nil && mapped.Budget.ApprovalGate != "" {
			budgetGates[mapped.Budget.ApprovalGate] = struct{}{}
		}
	}
	for _, reduced := range ir.Reductions {
		if reduced.Budget != nil && reduced.Budget.ApprovalGate != "" {
			budgetGates[reduced.Budget.ApprovalGate] = struct{}{}
		}
	}
	for _, gate := range ir.Gates {
		policyDigest, digestErr := Digest(gate.Policy)
		if digestErr != nil {
			return WorkflowPlan{}, digestErr
		}
		_, budgetActivation := budgetGates[gate.Key]
		plan.Gates = append(plan.Gates, PlanGate{
			Key: gate.Key, DependsOn: append([]NodeKey(nil), gate.DependsOn...),
			Policy: gate.Policy, PolicyDigest: policyDigest,
			BudgetActivation: budgetActivation,
		})
	}
	withoutDigest := plan
	withoutDigest.Digest = ""
	plan.Digest, err = Digest(withoutDigest)
	if err != nil {
		return WorkflowPlan{}, err
	}
	return plan, nil
}

func refSchema(ref ValueRef, inputs map[string]string, nodes map[NodeKey]TaskSpec, gates map[NodeKey]string, reductions map[string]string) (string, error) {
	switch ref.Source {
	case "input":
		schema, ok := inputs[ref.Name]
		if !ok {
			return "", fmt.Errorf("unknown input %q", ref.Name)
		}
		return schema, nil
	case "gate-output":
		schema, ok := gates[ref.GateKey]
		if !ok {
			return "", fmt.Errorf("unknown gate %q", ref.GateKey)
		}
		return schema, nil
	case "reduction-output":
		schema, ok := reductions[ref.ReduceKey]
		if !ok {
			return "", fmt.Errorf("unknown reduction %q", ref.ReduceKey)
		}
		return schema, nil
	case "node-output":
		spec, ok := nodes[ref.NodeKey]
		if !ok {
			return "", fmt.Errorf("unknown node %q", ref.NodeKey)
		}
		schema, ok := spec.Outputs[ref.Port]
		if !ok {
			return "", fmt.Errorf("unknown output %q on node %q", ref.Port, ref.NodeKey)
		}
		return schema, nil
	default:
		return "", fmt.Errorf("unsupported ref source %q", ref.Source)
	}
}

func setRefSchema(ref SetRef, inputs, maps map[string]SetRef) (SetRef, error) {
	switch ref.Source {
	case "set-input":
		actual, ok := inputs[ref.Name]
		if !ok {
			return SetRef{}, fmt.Errorf("unknown set input %q", ref.Name)
		}
		if ref.MapKey != "" {
			return SetRef{}, fmt.Errorf("set input ref cannot have a map key")
		}
		return actual, nil
	case "map-output":
		actual, ok := maps[ref.MapKey]
		if !ok {
			return SetRef{}, fmt.Errorf("unknown or forward map output %q", ref.MapKey)
		}
		if ref.Name != "" {
			return SetRef{}, fmt.Errorf("map output ref cannot have an input name")
		}
		return actual, nil
	default:
		return SetRef{}, fmt.Errorf("unsupported set ref source %q", ref.Source)
	}
}

func validateGateAcyclic(nodes []IRNode, gates []IRGate) error {
	dependencies := map[string][]string{}
	for _, node := range nodes {
		key := "node:" + string(node.Key)
		for _, dependency := range node.DependsOn {
			dependencies[key] = append(dependencies[key], "node:"+string(dependency))
		}
		for _, binding := range node.Bindings {
			if binding.Source == "gate-output" {
				dependencies[key] = append(dependencies[key], "gate:"+string(binding.GateKey))
			}
		}
		if node.Budget != nil && node.Budget.ApprovalGate != "" {
			dependencies[key] = append(dependencies[key], "gate:"+string(node.Budget.ApprovalGate))
		}
	}
	for _, gate := range gates {
		key := "gate:" + string(gate.Key)
		for _, dependency := range gate.DependsOn {
			dependencies[key] = append(dependencies[key], "node:"+string(dependency))
		}
	}
	state := map[string]int{}
	var visit func(string) error
	visit = func(key string) error {
		switch state[key] {
		case 1:
			return fmt.Errorf("workflow gate dependency cycle includes %q", key)
		case 2:
			return nil
		}
		state[key] = 1
		for _, dependency := range dependencies[key] {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[key] = 2
		return nil
	}
	keys := make([]string, 0, len(dependencies))
	for key := range dependencies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := visit(key); err != nil {
			return err
		}
	}
	return nil
}

func validateAcyclic(nodes []IRNode) error {
	dependencies := make(map[NodeKey][]NodeKey, len(nodes))
	for _, node := range nodes {
		dependencies[node.Key] = append([]NodeKey(nil), node.DependsOn...)
	}
	state := map[NodeKey]int{}
	var visit func(NodeKey) error
	visit = func(key NodeKey) error {
		switch state[key] {
		case 1:
			return fmt.Errorf("workflow dependency cycle includes %q", key)
		case 2:
			return nil
		}
		state[key] = 1
		for _, dependency := range dependencies[key] {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[key] = 2
		return nil
	}
	keys := make([]string, 0, len(nodes))
	for _, node := range nodes {
		keys = append(keys, string(node.Key))
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := visit(NodeKey(key)); err != nil {
			return err
		}
	}
	return nil
}

func cloneBindings(input map[string]ValueRef) map[string]ValueRef {
	ret := make(map[string]ValueRef, len(input))
	for key, value := range input {
		ret[key] = value
	}
	return ret
}

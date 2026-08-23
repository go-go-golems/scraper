package workflowv3

import (
	"fmt"
	"slices"
	"sort"
)

const (
	workNode      = "node:"
	workMap       = "map:"
	workReduction = "reduction:"
	workGate      = "gate:"
)

// EffectiveNodeDependencies returns the canonical union of explicit control
// dependencies and node producers referenced by data bindings. Bindings are
// authoritative for dataflow; explicit dependencies remain useful for
// control-only ordering when no value is consumed.
func EffectiveNodeDependencies(bindings map[string]ValueRef, explicit []NodeKey) []NodeKey {
	dependencies := make(map[NodeKey]struct{}, len(explicit)+len(bindings))
	for _, dependency := range explicit {
		dependencies[dependency] = struct{}{}
	}
	for _, binding := range bindings {
		if binding.Source == "node-output" {
			dependencies[binding.NodeKey] = struct{}{}
		}
	}
	ret := make([]NodeKey, 0, len(dependencies))
	for dependency := range dependencies {
		ret = append(ret, dependency)
	}
	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })
	return ret
}

// ValidatePlanDependencies checks that a decoded plan retains the canonical
// dependency contract emitted by Compile. This protects cross-process callers
// that can supply a digest-valid but structurally inconsistent plan.
func ValidatePlanDependencies(plan WorkflowPlan) error {
	nodes := make(map[NodeKey]struct{}, len(plan.Nodes))
	maps := make(map[string]struct{}, len(plan.Maps))
	reductions := make(map[string]struct{}, len(plan.Reductions))
	gates := make(map[NodeKey]struct{}, len(plan.Gates))
	for _, node := range plan.Nodes {
		nodes[node.Key] = struct{}{}
	}
	for _, mapped := range plan.Maps {
		maps[mapped.Key] = struct{}{}
	}
	for _, reduced := range plan.Reductions {
		reductions[reduced.Key] = struct{}{}
	}
	for _, gate := range plan.Gates {
		gates[gate.Key] = struct{}{}
	}
	validateBinding := func(owner string, binding ValueRef) error {
		switch binding.Source {
		case "node-output":
			if _, ok := nodes[binding.NodeKey]; !ok {
				return fmt.Errorf("%s references unknown node %q", owner, binding.NodeKey)
			}
		case "gate-output":
			if _, ok := gates[binding.GateKey]; !ok {
				return fmt.Errorf("%s references unknown gate %q", owner, binding.GateKey)
			}
		case "reduction-output":
			if _, ok := reductions[binding.ReduceKey]; !ok {
				return fmt.Errorf("%s references unknown reduction %q", owner, binding.ReduceKey)
			}
		}
		return nil
	}
	validateBudgetGate := func(owner string, gate NodeKey) error {
		if gate == "" {
			return nil
		}
		if _, ok := gates[gate]; !ok {
			return fmt.Errorf("%s references unknown budget gate %q", owner, gate)
		}
		return nil
	}

	ir := WorkflowIR{}
	for _, node := range plan.Nodes {
		canonical := EffectiveNodeDependencies(node.Bindings, node.DependsOn)
		if !slices.Equal(node.DependsOn, canonical) {
			return fmt.Errorf("plan node %q dependencies are not canonical", node.Key)
		}
		for _, dependency := range node.DependsOn {
			if _, ok := nodes[dependency]; !ok {
				return fmt.Errorf("plan node %q references unknown dependency %q", node.Key, dependency)
			}
		}
		for _, binding := range node.Bindings {
			if err := validateBinding("plan node "+string(node.Key), binding); err != nil {
				return err
			}
		}
		if node.Budget != nil {
			if err := validateBudgetGate("plan node "+string(node.Key), node.Budget.ApprovalGate); err != nil {
				return err
			}
		}
		ir.Nodes = append(ir.Nodes, IRNode{Key: node.Key, Bindings: node.Bindings, DependsOn: node.DependsOn, Budget: planBudgetDependency(node.Budget)})
	}
	for _, mapped := range plan.Maps {
		if mapped.Source.Source == "map-output" {
			if _, ok := maps[mapped.Source.MapKey]; !ok {
				return fmt.Errorf("plan map %q references unknown source map %q", mapped.Key, mapped.Source.MapKey)
			}
		}
		for _, binding := range mapped.Bindings {
			if err := validateBinding("plan map "+mapped.Key, binding); err != nil {
				return err
			}
		}
		if mapped.Budget != nil {
			if err := validateBudgetGate("plan map "+mapped.Key, mapped.Budget.ApprovalGate); err != nil {
				return err
			}
		}
		ir.Maps = append(ir.Maps, IRMap{Key: mapped.Key, Source: mapped.Source, Bindings: mapped.Bindings, Budget: planBudgetDependency(mapped.Budget)})
	}
	for _, reduced := range plan.Reductions {
		if reduced.Source.Source == "map-output" {
			if _, ok := maps[reduced.Source.MapKey]; !ok {
				return fmt.Errorf("plan reduction %q references unknown source map %q", reduced.Key, reduced.Source.MapKey)
			}
		}
		for _, binding := range reduced.Bindings {
			if err := validateBinding("plan reduction "+reduced.Key, binding); err != nil {
				return err
			}
		}
		if reduced.Budget != nil {
			if err := validateBudgetGate("plan reduction "+reduced.Key, reduced.Budget.ApprovalGate); err != nil {
				return err
			}
		}
		ir.Reductions = append(ir.Reductions, IRReduce{Key: reduced.Key, Source: reduced.Source, Bindings: reduced.Bindings, Budget: planBudgetDependency(reduced.Budget)})
	}
	for _, gate := range plan.Gates {
		for _, dependency := range gate.DependsOn {
			if _, ok := nodes[dependency]; !ok {
				return fmt.Errorf("plan gate %q references unknown dependency %q", gate.Key, dependency)
			}
		}
		ir.Gates = append(ir.Gates, IRGate{Key: gate.Key, DependsOn: gate.DependsOn})
	}
	return validateWorkflowAcyclic(ir)
}

func planBudgetDependency(claim *PlanBudgetClaim) *BudgetClaim {
	if claim == nil {
		return nil
	}
	return &BudgetClaim{ApprovalGate: claim.ApprovalGate}
}

func validateWorkflowAcyclic(ir WorkflowIR) error {
	edges := map[string]map[string]struct{}{}
	addWork := func(key string) {
		if _, exists := edges[key]; !exists {
			edges[key] = map[string]struct{}{}
		}
	}
	addEdge := func(consumer, producer string) {
		addWork(consumer)
		addWork(producer)
		edges[consumer][producer] = struct{}{}
	}
	addBinding := func(consumer string, binding ValueRef) {
		switch binding.Source {
		case "node-output":
			addEdge(consumer, workNode+string(binding.NodeKey))
		case "gate-output":
			addEdge(consumer, workGate+string(binding.GateKey))
		case "reduction-output":
			addEdge(consumer, workReduction+binding.ReduceKey)
		}
	}
	addBudgetGate := func(consumer string, claim *BudgetClaim) {
		if claim != nil && claim.ApprovalGate != "" {
			addEdge(consumer, workGate+string(claim.ApprovalGate))
		}
	}

	for _, node := range ir.Nodes {
		consumer := workNode + string(node.Key)
		addWork(consumer)
		for _, dependency := range node.DependsOn {
			addEdge(consumer, workNode+string(dependency))
		}
		for _, binding := range node.Bindings {
			addBinding(consumer, binding)
		}
		addBudgetGate(consumer, node.Budget)
	}
	for _, mapped := range ir.Maps {
		consumer := workMap + mapped.Key
		addWork(consumer)
		if mapped.Source.Source == "map-output" {
			addEdge(consumer, workMap+mapped.Source.MapKey)
		}
		for _, binding := range mapped.Bindings {
			addBinding(consumer, binding)
		}
		addBudgetGate(consumer, mapped.Budget)
	}
	for _, reduced := range ir.Reductions {
		consumer := workReduction + reduced.Key
		addWork(consumer)
		if reduced.Source.Source == "map-output" {
			addEdge(consumer, workMap+reduced.Source.MapKey)
		}
		for _, binding := range reduced.Bindings {
			addBinding(consumer, binding)
		}
		addBudgetGate(consumer, reduced.Budget)
	}
	for _, gate := range ir.Gates {
		consumer := workGate + string(gate.Key)
		addWork(consumer)
		for _, dependency := range gate.DependsOn {
			addEdge(consumer, workNode+string(dependency))
		}
	}

	adjacency := make(map[string][]string, len(edges))
	keys := make([]string, 0, len(edges))
	for key, producers := range edges {
		keys = append(keys, key)
		for producer := range producers {
			adjacency[key] = append(adjacency[key], producer)
		}
		sort.Strings(adjacency[key])
	}
	sort.Strings(keys)

	state := map[string]int{}
	stack := []string{}
	var visit func(string) error
	visit = func(key string) error {
		switch state[key] {
		case 1:
			cycle := append(append([]string(nil), stack...), key)
			return fmt.Errorf("workflow dependency cycle includes %q", cycle)
		case 2:
			return nil
		}
		state[key] = 1
		stack = append(stack, key)
		for _, producer := range adjacency[key] {
			if err := visit(producer); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		state[key] = 2
		return nil
	}
	for _, key := range keys {
		if err := visit(key); err != nil {
			return err
		}
	}
	return nil
}

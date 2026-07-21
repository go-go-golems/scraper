package workflowv3

import (
	"fmt"
	"sort"
	"strings"
)

type Catalog struct {
	tasks map[TaskKey]TaskSpec
}

func NewCatalog(specs ...TaskSpec) (*Catalog, error) {
	catalog := &Catalog{tasks: make(map[TaskKey]TaskSpec, len(specs))}
	for _, spec := range specs {
		if err := validateTaskSpec(spec); err != nil {
			return nil, err
		}
		key := spec.Identity.TaskKey
		if _, exists := catalog.tasks[key]; exists {
			return nil, fmt.Errorf("task %s@%s is already registered", key.Kind, key.Version)
		}
		catalog.tasks[key] = cloneTaskSpec(spec)
	}
	return catalog, nil
}

func (c *Catalog) Lookup(key TaskKey) (TaskSpec, bool) {
	if c == nil {
		return TaskSpec{}, false
	}
	spec, ok := c.tasks[key]
	return cloneTaskSpec(spec), ok
}

func (c *Catalog) Specs() []TaskSpec {
	if c == nil {
		return nil
	}
	ret := make([]TaskSpec, 0, len(c.tasks))
	for _, spec := range c.tasks {
		ret = append(ret, cloneTaskSpec(spec))
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Identity.Kind == ret[j].Identity.Kind {
			return ret[i].Identity.Version < ret[j].Identity.Version
		}
		return ret[i].Identity.Kind < ret[j].Identity.Kind
	})
	return ret
}

func (c *Catalog) Digest() (string, error) {
	return Digest(c.Specs())
}

func validateTaskSpec(spec TaskSpec) error {
	identity := spec.Identity
	if strings.TrimSpace(identity.Kind) == "" || strings.TrimSpace(identity.Version) == "" {
		return fmt.Errorf("task kind and version are required")
	}
	if strings.TrimSpace(identity.BundleDigest) == "" {
		return fmt.Errorf("task %s@%s bundle digest is required", identity.Kind, identity.Version)
	}
	if strings.TrimSpace(identity.Entrypoint) == "" || !strings.Contains(identity.Entrypoint, "#") {
		return fmt.Errorf("task %s@%s entrypoint must contain #export", identity.Kind, identity.Version)
	}
	if identity.ABI != TaskABI {
		return fmt.Errorf("task %s@%s ABI %q is not supported", identity.Kind, identity.Version, identity.ABI)
	}
	if len(spec.Inputs) == 0 || len(spec.Outputs) == 0 {
		return fmt.Errorf("task %s@%s requires input and output ports", identity.Kind, identity.Version)
	}
	for name, schema := range spec.Inputs {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(schema) == "" {
			return fmt.Errorf("task %s@%s has invalid input port", identity.Kind, identity.Version)
		}
	}
	for name, schema := range spec.Outputs {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(schema) == "" {
			return fmt.Errorf("task %s@%s has invalid output port", identity.Kind, identity.Version)
		}
	}
	seenModules := map[string]struct{}{}
	for _, module := range spec.Modules {
		if strings.TrimSpace(module) == "" {
			return fmt.Errorf("task %s@%s has empty module alias", identity.Kind, identity.Version)
		}
		if _, exists := seenModules[module]; exists {
			return fmt.Errorf("task %s@%s repeats module alias %q", identity.Kind, identity.Version, module)
		}
		seenModules[module] = struct{}{}
	}
	return nil
}

func cloneTaskSpec(spec TaskSpec) TaskSpec {
	ret := spec
	ret.Inputs = cloneStringMap(spec.Inputs)
	ret.Outputs = cloneStringMap(spec.Outputs)
	ret.Modules = append([]string(nil), spec.Modules...)
	sort.Strings(ret.Modules)
	return ret
}

func cloneStringMap(input map[string]string) map[string]string {
	ret := make(map[string]string, len(input))
	for key, value := range input {
		ret[key] = value
	}
	return ret
}

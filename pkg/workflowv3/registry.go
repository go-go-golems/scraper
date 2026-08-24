package workflowv3

import (
	"fmt"
	"sort"
	"strings"
)

type RegisteredTask struct {
	Spec   TaskSpec
	Bundle *Bundle
}

type RegistryBuilder struct {
	tasks              map[ImplementationIdentity]RegisteredTask
	modules            map[string]struct{}
	isolationExecutors map[string]string
}

type RegistryResolver interface {
	ResolveNode(PlanNode) (RegisteredTask, error)
	AcquireNode(PlanNode) (RegisteredTask, string, func(), error)
	ModuleAliases() []string
	Catalog() (*Catalog, error)
}

type SealedRegistry struct {
	generation         string
	tasks              map[ImplementationIdentity]RegisteredTask
	modules            map[string]struct{}
	isolationExecutors map[string]string
}

func NewRegistryBuilder() *RegistryBuilder {
	return &RegistryBuilder{
		tasks:              map[ImplementationIdentity]RegisteredTask{},
		modules:            map[string]struct{}{},
		isolationExecutors: map[string]string{},
	}
}

// AdvertiseModules adds exact policy-selected module aliases to this worker
// registry transaction. It grants no runtime implementation by itself.
func (b *RegistryBuilder) AdvertiseModules(aliases ...string) error {
	if b == nil {
		return fmt.Errorf("registry builder is nil")
	}
	for _, alias := range aliases {
		alias = strings.TrimSpace(alias)
		if alias == "" {
			return fmt.Errorf("module alias is required")
		}
		b.modules[alias] = struct{}{}
	}
	return nil
}

func (b *RegistryBuilder) AdvertiseIsolationExecutor(class, digest string) error {
	if b == nil {
		return fmt.Errorf("registry builder is nil")
	}
	if class != IsolationSubprocessRestricted {
		return fmt.Errorf("isolation executor class %q is not advertisable", class)
	}
	if err := validateSHA256Digest(digest); err != nil {
		return fmt.Errorf("isolation executor digest: %w", err)
	}
	if existing, ok := b.isolationExecutors[class]; ok && existing != digest {
		return fmt.Errorf("isolation executor class %q is already advertised with another digest", class)
	}
	b.isolationExecutors[class] = digest
	return nil
}

func (b *RegistryBuilder) AddBundle(bundle *Bundle) error {
	if b == nil {
		return fmt.Errorf("registry builder is nil")
	}
	if bundle == nil {
		return fmt.Errorf("bundle is required")
	}
	for _, spec := range bundle.TaskSpecs() {
		if _, exists := b.tasks[spec.Identity]; exists {
			return fmt.Errorf("implementation is already registered: %s", formatIdentity(spec.Identity))
		}
		b.tasks[spec.Identity] = RegisteredTask{Spec: spec, Bundle: bundle}
	}
	return nil
}

func (b *RegistryBuilder) Seal() (*SealedRegistry, error) {
	if b == nil || len(b.tasks) == 0 {
		return nil, fmt.Errorf("cannot seal an empty registry")
	}
	identities := make([]ImplementationIdentity, 0, len(b.tasks))
	sealedTasks := make(map[ImplementationIdentity]RegisteredTask, len(b.tasks))
	for identity, task := range b.tasks {
		identities = append(identities, identity)
		if task.Spec.IsolationMaximum.Class == IsolationSubprocessRestricted {
			digest, ok := b.isolationExecutors[IsolationSubprocessRestricted]
			if !ok {
				return nil, fmt.Errorf("implementation %s requires an unadvertised restricted isolation executor", formatIdentity(task.Spec.Identity))
			}
			task.Spec.IsolationExecutorDigest = digest
		}
		sealedTasks[identity] = task
	}
	sort.Slice(identities, func(i, j int) bool {
		return formatIdentity(identities[i]) < formatIdentity(identities[j])
	})
	modules := make([]string, 0, len(b.modules))
	sealedModules := make(map[string]struct{}, len(b.modules))
	for alias := range b.modules {
		modules = append(modules, alias)
		sealedModules[alias] = struct{}{}
	}
	sort.Strings(modules)
	for _, task := range sealedTasks {
		for _, alias := range task.Spec.Modules {
			if _, ok := sealedModules[alias]; !ok {
				return nil, fmt.Errorf("implementation %s requires unadvertised module %q", formatIdentity(task.Spec.Identity), alias)
			}
		}
	}
	type isolationExecutor struct {
		Class  string `json:"class"`
		Digest string `json:"digest"`
	}
	executors := make([]isolationExecutor, 0, len(b.isolationExecutors))
	sealedExecutors := make(map[string]string, len(b.isolationExecutors))
	for class, digest := range b.isolationExecutors {
		executors = append(executors, isolationExecutor{Class: class, Digest: digest})
		sealedExecutors[class] = digest
	}
	sort.Slice(executors, func(i, j int) bool { return executors[i].Class < executors[j].Class })
	generation, err := Digest(struct {
		Identities         []ImplementationIdentity `json:"identities"`
		Modules            []string                 `json:"modules"`
		IsolationExecutors []isolationExecutor      `json:"isolationExecutors,omitempty"`
	}{Identities: identities, Modules: modules, IsolationExecutors: executors})
	if err != nil {
		return nil, err
	}
	return &SealedRegistry{
		generation: generation, tasks: sealedTasks, modules: sealedModules,
		isolationExecutors: sealedExecutors,
	}, nil
}

func (r *SealedRegistry) Generation() string {
	if r == nil {
		return ""
	}
	return r.generation
}

func (r *SealedRegistry) IsolationExecutorDigests() []string {
	if r == nil {
		return nil
	}
	ret := make([]string, 0, len(r.isolationExecutors))
	for _, digest := range r.isolationExecutors {
		ret = append(ret, digest)
	}
	sort.Strings(ret)
	return ret
}

func (r *SealedRegistry) ModuleAliases() []string {
	if r == nil {
		return nil
	}
	aliases := make([]string, 0, len(r.modules))
	for alias := range r.modules {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func (r *SealedRegistry) Resolve(identity ImplementationIdentity) (RegisteredTask, error) {
	if r == nil {
		return RegisteredTask{}, fmt.Errorf("sealed registry is required")
	}
	task, ok := r.tasks[identity]
	if !ok {
		return RegisteredTask{}, fmt.Errorf("registry generation %s does not advertise exact implementation %s", r.generation, formatIdentity(identity))
	}
	return task, nil
}

func (r *SealedRegistry) AcquireNode(node PlanNode) (RegisteredTask, string, func(), error) {
	task, err := r.ResolveNode(node)
	if err != nil {
		return RegisteredTask{}, "", nil, err
	}
	return task, r.Generation(), func() {}, nil
}

func (r *SealedRegistry) ResolveNode(node PlanNode) (RegisteredTask, error) {
	task, err := r.Resolve(node.Implementation)
	if err != nil {
		return RegisteredTask{}, err
	}
	if task.Spec.ResourceClass != node.ResourceClass || task.Spec.Retry != node.Retry {
		return RegisteredTask{}, fmt.Errorf("plan node policy does not match registered implementation")
	}
	if err := ValidatePlanIsolation(node.Isolation, task.Spec.IsolationMaximum); err != nil {
		return RegisteredTask{}, fmt.Errorf("plan node isolation does not match registered implementation: %w", err)
	}
	if strings.Join(task.Spec.Modules, "\x00") != strings.Join(node.Modules, "\x00") {
		return RegisteredTask{}, fmt.Errorf("plan node modules do not match registered implementation")
	}
	for _, alias := range node.Modules {
		if _, ok := r.modules[alias]; !ok {
			return RegisteredTask{}, fmt.Errorf("registry generation %s does not advertise module %q", r.generation, alias)
		}
	}
	return task, nil
}

func (r *SealedRegistry) Catalog() (*Catalog, error) {
	if r == nil {
		return nil, fmt.Errorf("sealed registry is required")
	}
	specs := make([]TaskSpec, 0, len(r.tasks))
	for _, task := range r.tasks {
		specs = append(specs, task.Spec)
	}
	return NewCatalog(specs...)
}

func formatIdentity(identity ImplementationIdentity) string {
	return fmt.Sprintf(
		"%s@%s bundle=%s entrypoint=%s abi=%s",
		identity.Kind,
		identity.Version,
		identity.BundleDigest,
		identity.Entrypoint,
		identity.ABI,
	)
}

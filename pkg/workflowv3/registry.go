package workflowv3

import (
	"fmt"
	"sort"
)

type RegisteredTask struct {
	Spec   TaskSpec
	Bundle *Bundle
}

type RegistryBuilder struct {
	tasks map[ImplementationIdentity]RegisteredTask
}

type SealedRegistry struct {
	generation string
	tasks      map[ImplementationIdentity]RegisteredTask
}

func NewRegistryBuilder() *RegistryBuilder {
	return &RegistryBuilder{tasks: map[ImplementationIdentity]RegisteredTask{}}
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
		sealedTasks[identity] = task
	}
	sort.Slice(identities, func(i, j int) bool {
		return formatIdentity(identities[i]) < formatIdentity(identities[j])
	})
	generation, err := Digest(identities)
	if err != nil {
		return nil, err
	}
	return &SealedRegistry{generation: generation, tasks: sealedTasks}, nil
}

func (r *SealedRegistry) Generation() string {
	if r == nil {
		return ""
	}
	return r.generation
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

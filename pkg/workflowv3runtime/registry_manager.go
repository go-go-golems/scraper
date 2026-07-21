package workflowv3runtime

import (
	"fmt"
	"sort"
	"sync"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type RegistryGenerationSnapshot struct {
	Generation     string
	State          string
	References     int
	Failures       int
	QuarantineCode string
}

type RegistryManagerSnapshot struct {
	Active      string
	Generations []RegistryGenerationSnapshot
}

type managedGeneration struct {
	registry       *workflowv3.SealedRegistry
	state          string
	activation     uint64
	references     int
	failures       int
	quarantineCode string
}

// RegistryManager atomically activates immutable sealed registries and retains
// acquired old generations until attempts release them.
type RegistryManager struct {
	mu          sync.Mutex
	generations map[string]*managedGeneration
	active      string
	sequence    uint64
}

func NewRegistryManager(initial *workflowv3.SealedRegistry) (*RegistryManager, error) {
	if initial == nil || initial.Generation() == "" {
		return nil, fmt.Errorf("initial sealed registry is required")
	}
	manager := &RegistryManager{generations: map[string]*managedGeneration{}}
	manager.sequence = 1
	manager.active = initial.Generation()
	manager.generations[manager.active] = &managedGeneration{
		registry: initial, state: "active", activation: manager.sequence,
	}
	return manager, nil
}

// Activate validates a complete candidate before one atomic active-generation
// swap. A failed self-test leaves all manager state unchanged.
func (m *RegistryManager) Activate(
	candidate *workflowv3.SealedRegistry,
	selfTest func(*workflowv3.SealedRegistry) error,
) error {
	if m == nil || candidate == nil || candidate.Generation() == "" {
		return fmt.Errorf("candidate sealed registry is required")
	}
	if selfTest != nil {
		if err := selfTest(candidate); err != nil {
			return fmt.Errorf("candidate registry self-test: %w", err)
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	digest := candidate.Generation()
	if digest == m.active {
		return nil
	}
	entry := m.generations[digest]
	if entry != nil && entry.references > 0 && entry.registry != candidate {
		return fmt.Errorf("generation %s is already retained with different registry object", digest)
	}
	if current := m.generations[m.active]; current != nil && current.state == "active" {
		current.state = "draining"
	}
	m.sequence++
	if entry == nil {
		entry = &managedGeneration{registry: candidate}
		m.generations[digest] = entry
	}
	entry.registry = candidate
	entry.state = "active"
	entry.activation = m.sequence
	entry.failures = 0
	entry.quarantineCode = ""
	m.active = digest
	return nil
}

func (m *RegistryManager) ResolveNode(node workflowv3.PlanNode) (workflowv3.RegisteredTask, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, task, err := m.resolveLocked(node)
	_ = entry
	return task, err
}

func (m *RegistryManager) AcquireNode(node workflowv3.PlanNode) (workflowv3.RegisteredTask, string, func(), error) {
	m.mu.Lock()
	entry, task, err := m.resolveLocked(node)
	if err != nil {
		m.mu.Unlock()
		return workflowv3.RegisteredTask{}, "", nil, err
	}
	entry.references++
	generation := entry.registry.Generation()
	m.mu.Unlock()
	var once sync.Once
	release := func() {
		once.Do(func() {
			m.mu.Lock()
			defer m.mu.Unlock()
			current := m.generations[generation]
			if current != nil && current.references > 0 {
				current.references--
			}
		})
	}
	return task, generation, release, nil
}

func (m *RegistryManager) Catalog() (*workflowv3.Catalog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.generations[m.active]
	if entry == nil || entry.state == "quarantined" {
		return nil, fmt.Errorf("active registry generation is unavailable")
	}
	return entry.registry.Catalog()
}

func (m *RegistryManager) ModuleAliases() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	aliases := map[string]struct{}{}
	for _, entry := range m.generations {
		for _, alias := range entry.registry.ModuleAliases() {
			aliases[alias] = struct{}{}
		}
	}
	ret := make([]string, 0, len(aliases))
	for alias := range aliases {
		ret = append(ret, alias)
	}
	sort.Strings(ret)
	return ret
}

func (m *RegistryManager) Quarantine(generation, code string) error {
	if m == nil || generation == "" || code == "" {
		return fmt.Errorf("generation and quarantine code are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.generations[generation]
	if entry == nil {
		return fmt.Errorf("unknown registry generation %s", generation)
	}
	entry.state = "quarantined"
	entry.quarantineCode = code
	return nil
}

func (m *RegistryManager) RecordConstructionFailure(generation, code string, threshold int) (bool, error) {
	if threshold < 1 {
		return false, fmt.Errorf("quarantine threshold must be positive")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.generations[generation]
	if entry == nil {
		return false, fmt.Errorf("unknown registry generation %s", generation)
	}
	entry.failures++
	if entry.failures < threshold {
		return false, nil
	}
	entry.state = "quarantined"
	entry.quarantineCode = code
	return true, nil
}

func (m *RegistryManager) RemoveDrained(generation string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry := m.generations[generation]
	if entry == nil {
		return fmt.Errorf("unknown registry generation %s", generation)
	}
	if entry.state != "draining" || entry.references != 0 || generation == m.active {
		return fmt.Errorf("generation %s is not removable", generation)
	}
	delete(m.generations, generation)
	return nil
}

func (m *RegistryManager) Progress() []workflowv3.RegistryGenerationProgress {
	snapshot := m.Snapshot()
	progress := make([]workflowv3.RegistryGenerationProgress, 0, len(snapshot.Generations))
	for _, generation := range snapshot.Generations {
		progress = append(progress, workflowv3.RegistryGenerationProgress{
			Generation: generation.Generation, State: generation.State,
			References: generation.References, Failures: generation.Failures,
			QuarantineCode: generation.QuarantineCode,
		})
	}
	return progress
}

func (m *RegistryManager) Snapshot() RegistryManagerSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot := RegistryManagerSnapshot{Active: m.active}
	for generation, entry := range m.generations {
		snapshot.Generations = append(snapshot.Generations, RegistryGenerationSnapshot{
			Generation: generation, State: entry.state, References: entry.references,
			Failures: entry.failures, QuarantineCode: entry.quarantineCode,
		})
	}
	sort.Slice(snapshot.Generations, func(i, j int) bool {
		return snapshot.Generations[i].Generation < snapshot.Generations[j].Generation
	})
	return snapshot
}

func (m *RegistryManager) resolveLocked(node workflowv3.PlanNode) (*managedGeneration, workflowv3.RegisteredTask, error) {
	ordered := make([]*managedGeneration, 0, len(m.generations))
	if active := m.generations[m.active]; active != nil && active.state != "quarantined" {
		ordered = append(ordered, active)
	}
	var retained []*managedGeneration
	for generation, entry := range m.generations {
		if generation == m.active || entry.state == "quarantined" {
			continue
		}
		retained = append(retained, entry)
	}
	sort.Slice(retained, func(i, j int) bool {
		return retained[i].activation > retained[j].activation
	})
	ordered = append(ordered, retained...)
	for _, entry := range ordered {
		task, err := entry.registry.ResolveNode(node)
		if err == nil {
			return entry, task, nil
		}
	}
	return nil, workflowv3.RegisteredTask{}, fmt.Errorf("no retained registry generation advertises exact node implementation")
}

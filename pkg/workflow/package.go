package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

// Package describes a workflow package/domain that can start durable runs.
type Package struct {
	Name        string
	DisplayName string
	Entrypoint  Entrypoint
}

type PackageBuilder struct {
	pkg *Package
}

func NewPackage(name string) *PackageBuilder {
	return &PackageBuilder{pkg: &Package{Name: name}}
}

func (b *PackageBuilder) DisplayName(name string) *PackageBuilder {
	b.pkg.DisplayName = name
	return b
}

func (b *PackageBuilder) Entrypoint(entrypoint Entrypoint) *PackageBuilder {
	b.pkg.Entrypoint = entrypoint
	return b
}

func (b *PackageBuilder) Build() *Package {
	cp := *b.pkg
	return &cp
}

// Entrypoint creates the initial durable step graph for a run.
type Entrypoint interface {
	Start(ctx context.Context, run *RunBuilder, rawInput json.RawMessage) error
}

// EntrypointFunc adapts a typed function to an Entrypoint.
type EntrypointFunc[I any] func(context.Context, *RunBuilder, I) error

func (f EntrypointFunc[I]) Start(ctx context.Context, run *RunBuilder, rawInput json.RawMessage) error {
	if f == nil {
		return fmt.Errorf("workflow entrypoint function is nil")
	}
	var input I
	if len(rawInput) > 0 {
		if err := json.Unmarshal(rawInput, &input); err != nil {
			return fmt.Errorf("decode workflow run input: %w", err)
		}
	}
	return f(ctx, run, input)
}

// RunBuilder constructs the initial step graph for a workflow run.
type RunBuilder struct {
	workflow model.WorkflowRun
	steps    []model.OpSpec
}

type StepHandle struct {
	ID model.OpID
}

func (b *RunBuilder) Name(name string) {
	b.workflow.Name = name
}

func (b *RunBuilder) Metadata(key, value string) {
	if b.workflow.Metadata == nil {
		b.workflow.Metadata = map[string]string{}
	}
	b.workflow.Metadata[key] = value
}

// Step appends an initial step to the run graph.
func (b *RunBuilder) Step(id string, input any, opts StepOpts) (StepHandle, error) {
	if b == nil {
		return StepHandle{}, fmt.Errorf("workflow run builder is nil")
	}
	if id == "" {
		id = fmt.Sprintf("%s:step:%03d", b.workflow.ID, len(b.steps)+1)
	}
	if opts.Kind == "" {
		return StepHandle{}, fmt.Errorf("workflow step kind is required")
	}
	raw, err := marshalJSON(input)
	if err != nil {
		return StepHandle{}, fmt.Errorf("marshal workflow step %s input: %w", id, err)
	}
	site := opts.Site
	if site == "" {
		site = b.workflow.Site
	}
	step := model.OpSpec{
		ID:         model.OpID(id),
		WorkflowID: b.workflow.ID,
		ParentID:   opts.ParentID,
		Site:       site,
		Kind:       opts.Kind,
		Queue:      opts.Queue,
		DedupKey:   opts.DedupKey,
		Input:      raw,
		DependsOn:  append([]model.Dependency(nil), opts.DependsOn...),
		Retry:      opts.Retry,
		Metadata:   cloneStringMap(opts.Metadata),
	}
	b.steps = append(b.steps, step)
	return StepHandle{ID: step.ID}, nil
}

func Require(handles ...StepHandle) []model.Dependency {
	deps := make([]model.Dependency, 0, len(handles))
	for _, handle := range handles {
		if handle.ID == "" {
			continue
		}
		deps = append(deps, model.Dependency{OpID: handle.ID, Required: true})
	}
	return deps
}

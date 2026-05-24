package workflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/runner"
)

// StepContext is the public executor-facing view of a durable workflow step.
// It wraps runner.RunContext and accumulates the result data, records,
// artifacts, and dynamically emitted child steps that will be persisted by the
// existing scheduler/store completion path.
type StepContext struct {
	run runner.RunContext

	data      json.RawMessage
	records   []model.RecordWrite
	artifacts []model.ArtifactWrite
	emitted   []model.OpSpec
}

func newStepContext(run runner.RunContext) *StepContext {
	return &StepContext{run: run}
}

// Workflow returns the current durable workflow run.
func (s *StepContext) Workflow() model.WorkflowRun {
	return s.run.Workflow
}

// Step returns the current durable step/op spec.
func (s *StepContext) Step() model.OpSpec {
	return s.run.Op
}

// Lease returns the current lease metadata.
func (s *StepContext) Lease() model.Lease {
	return s.run.Lease
}

// Now returns the scheduler-provided execution timestamp.
func (s *StepContext) Now() time.Time {
	return s.run.Now
}

// Input decodes this step's JSON input into out.
func (s *StepContext) Input(out any) error {
	if out == nil {
		return fmt.Errorf("workflow step input target is nil")
	}
	if len(s.run.Op.Input) == 0 {
		return nil
	}
	if err := json.Unmarshal(s.run.Op.Input, out); err != nil {
		return fmt.Errorf("decode workflow step input for %s: %w", s.run.Op.ID, err)
	}
	return nil
}

// RawInput returns a copy of the raw JSON input for advanced executors.
func (s *StepContext) RawInput() json.RawMessage {
	return append(json.RawMessage(nil), s.run.Op.Input...)
}

// Result stores structured step result data. It is serialized into the engine
// result row when the step completes.
func (s *StepContext) Result(data any) error {
	raw, err := marshalJSON(data)
	if err != nil {
		return fmt.Errorf("marshal workflow step result for %s: %w", s.run.Op.ID, err)
	}
	s.data = raw
	return nil
}

// Record appends a projection-style record write to the step result.
func (s *StepContext) Record(collection, key string, data any) error {
	collection = strings.TrimSpace(collection)
	key = strings.TrimSpace(key)
	if collection == "" {
		return fmt.Errorf("record collection is required")
	}
	if key == "" {
		return fmt.Errorf("record key is required")
	}
	raw, err := marshalJSON(data)
	if err != nil {
		return fmt.Errorf("marshal record %s/%s: %w", collection, key, err)
	}
	s.records = append(s.records, model.RecordWrite{Collection: collection, Key: key, Data: raw})
	return nil
}

// ArtifactOption customizes an artifact emitted by a step.
type ArtifactOption func(*model.ArtifactWrite)

func ArtifactID(id string) ArtifactOption {
	return func(a *model.ArtifactWrite) { a.ID = model.ArtifactID(strings.TrimSpace(id)) }
}

func ArtifactKind(kind string) ArtifactOption {
	return func(a *model.ArtifactWrite) { a.Kind = strings.TrimSpace(kind) }
}

func ArtifactMetadata(metadata map[string]string) ArtifactOption {
	return func(a *model.ArtifactWrite) { a.Metadata = cloneStringMap(metadata) }
}

// Artifact appends an artifact to the step result and returns its stable ID.
func (s *StepContext) Artifact(name, contentType string, body []byte, opts ...ArtifactOption) (model.ArtifactID, error) {
	name = strings.TrimSpace(name)
	contentType = strings.TrimSpace(contentType)
	if name == "" {
		return "", fmt.Errorf("artifact name is required")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	artifact := model.ArtifactWrite{
		ID:          model.ArtifactID(fmt.Sprintf("%s:artifact:%03d", s.run.Op.ID, len(s.artifacts)+1)),
		Name:        name,
		Kind:        inferArtifactKind(name, contentType),
		ContentType: contentType,
		Metadata:    map[string]string{},
		Body:        append([]byte(nil), body...),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&artifact)
		}
	}
	if artifact.ID == "" {
		return "", fmt.Errorf("artifact id cannot be empty")
	}
	s.artifacts = append(s.artifacts, artifact)
	return artifact.ID, nil
}

// StepOpts customizes a dynamically emitted child step.
type StepOpts struct {
	Kind      string
	Queue     model.QueueKey
	DedupKey  string
	DependsOn []model.Dependency
	Retry     model.RetryPolicy
	Metadata  map[string]string
	Site      model.SiteName
	ParentID  *model.OpID
}

// Emit appends a child step to this step's result. The child is persisted by the
// store when the current step completes successfully.
func (s *StepContext) Emit(id string, input any, opts StepOpts) (model.OpID, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		id = fmt.Sprintf("%s:emit:%03d", s.run.Op.ID, len(s.emitted)+1)
	}
	kind := strings.TrimSpace(opts.Kind)
	if kind == "" {
		return "", fmt.Errorf("emitted step kind is required")
	}
	raw, err := marshalJSON(input)
	if err != nil {
		return "", fmt.Errorf("marshal emitted step %s input: %w", id, err)
	}
	site := opts.Site
	if site == "" {
		site = s.run.Op.Site
	}
	parentID := opts.ParentID
	if parentID == nil {
		current := s.run.Op.ID
		parentID = &current
	}
	step := model.OpSpec{
		ID:         model.OpID(id),
		WorkflowID: s.run.Op.WorkflowID,
		ParentID:   parentID,
		Site:       site,
		Kind:       kind,
		Queue:      opts.Queue,
		DedupKey:   opts.DedupKey,
		Input:      raw,
		DependsOn:  append([]model.Dependency(nil), opts.DependsOn...),
		Retry:      opts.Retry,
		Metadata:   cloneStringMap(opts.Metadata),
	}
	s.emitted = append(s.emitted, step)
	return step.ID, nil
}

func (s *StepContext) opResult() *model.OpResult {
	result := &model.OpResult{
		OpID:        s.run.Op.ID,
		Data:        append(json.RawMessage(nil), s.data...),
		Records:     append([]model.RecordWrite(nil), s.records...),
		Artifacts:   append([]model.ArtifactWrite(nil), s.artifacts...),
		Emitted:     append([]model.OpSpec(nil), s.emitted...),
		CompletedAt: s.run.Now,
	}
	for _, emitted := range result.Emitted {
		result.EmittedIDs = append(result.EmittedIDs, emitted.ID)
	}
	return result
}

func marshalJSON(value any) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage(`null`), nil
	}
	if raw, ok := value.(json.RawMessage); ok {
		return append(json.RawMessage(nil), raw...), nil
	}
	body, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func inferArtifactKind(name, contentType string) string {
	if slash := strings.Index(contentType, "/"); slash > 0 {
		return contentType[:slash]
	}
	if dot := strings.LastIndex(name, "."); dot >= 0 && dot < len(name)-1 {
		return name[dot+1:]
	}
	return "artifact"
}

package submission

import (
	"context"
	"fmt"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	"github.com/go-go-golems/scraper/pkg/services/catalog"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	submitverbs "github.com/go-go-golems/scraper/pkg/sites/submitverbs"
)

type Request struct {
	Site       model.SiteName                    `json:"site"`
	Verb       string                            `json:"verb"`
	WorkflowID string                            `json:"workflowID,omitempty"`
	EngineDB   string                            `json:"engineDB,omitempty"`
	SitesDir   string                            `json:"sitesDir,omitempty"`
	Values     map[string]interface{}            `json:"values,omitempty"`
	Sections   map[string]map[string]interface{} `json:"sections,omitempty"`
}

type Response struct {
	Site        model.SiteName            `json:"site"`
	Verb        string                    `json:"verb"`
	CommandPath string                    `json:"commandPath"`
	Result      *submitverbs.SubmitResult `json:"result"`
}

type Service struct {
	siteRegistry *siteregistry.Registry
	catalog      *catalog.Service
	events       *runtimeevents.Publisher
}

func NewService(siteRegistry *siteregistry.Registry, events *runtimeevents.Publisher) *Service {
	return &Service{
		siteRegistry: siteRegistry,
		catalog:      catalog.NewService(siteRegistry),
		events:       events,
	}
}

func (s *Service) Submit(ctx context.Context, request Request) (*Response, error) {
	if s == nil || s.siteRegistry == nil {
		return nil, fmt.Errorf("submission service is not configured")
	}
	verbSummary, verb, command, def, err := s.catalog.GetVerb(request.Site, request.Verb)
	if err != nil {
		return nil, err
	}
	parsedValues, err := ValuesFromRequest(command.Description(), request.Values, request.Sections)
	if err != nil {
		return nil, err
	}

	siteVerbs, err := submitverbs.LoadSiteVerbs(def)
	if err != nil {
		return nil, err
	}
	host := submitverbs.NewHost(s.siteRegistry, def, siteVerbs.Registry, s.events)
	result, err := host.Submit(ctx, verb, parsedValues, submitverbs.SubmitOptions{
		EngineDB:   request.EngineDB,
		SitesDir:   request.SitesDir,
		WorkflowID: request.WorkflowID,
	})
	if err != nil {
		return nil, err
	}
	result.CommandPath = verbSummary.CommandPath
	_ = runtimeevents.EmitSimpleEvent(s.events, &runtimev1.RuntimeEventV1{
		Source:     runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_SUBMISSION,
		Component:  "submission-service",
		Kind:       runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_SUBMISSION_ACCEPTED,
		Severity:   runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_INFO,
		Message:    "submission accepted",
		WorkflowId: string(result.Workflow.ID),
		OpId:       string(result.TargetOpID),
		Site:       string(request.Site),
		RequestId:  runtimeevents.RequestIDFromContext(ctx),
		Payload: runtimeevents.BuildPayload(map[string]any{
			"verb":           request.Verb,
			"submittedCount": len(result.Submitted),
			"commandPath":    verbSummary.CommandPath,
			"siteDbPath":     result.SiteDBPath,
		}),
	})

	return &Response{
		Site:        request.Site,
		Verb:        request.Verb,
		CommandPath: verbSummary.CommandPath,
		Result:      result,
	}, nil
}

package submission

import (
	"context"
	"fmt"

	"github.com/go-go-golems/scraper/pkg/engine/model"
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
}

func NewService(siteRegistry *siteregistry.Registry) *Service {
	return &Service{
		siteRegistry: siteRegistry,
		catalog:      catalog.NewService(siteRegistry),
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
	host := submitverbs.NewHost(s.siteRegistry, def, siteVerbs.Registry)
	result, err := host.Submit(ctx, verb, parsedValues, submitverbs.SubmitOptions{
		EngineDB:   request.EngineDB,
		SitesDir:   request.SitesDir,
		WorkflowID: request.WorkflowID,
	})
	if err != nil {
		return nil, err
	}
	result.CommandPath = verbSummary.CommandPath

	return &Response{
		Site:        request.Site,
		Verb:        request.Verb,
		CommandPath: verbSummary.CommandPath,
		Result:      result,
	}, nil
}

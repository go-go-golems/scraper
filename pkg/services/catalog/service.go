package catalog

import (
	"fmt"
	"sort"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	submitverbs "github.com/go-go-golems/scraper/pkg/sites/submitverbs"
)

type NotFoundError struct {
	Kind string
	Name string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s %q not found", e.Kind, e.Name)
}

type SiteSummary struct {
	Name             model.SiteName `json:"name"`
	DatabaseFileName string         `json:"databaseFileName"`
	HasScripts       bool           `json:"hasScripts"`
	HasSubmitVerbs   bool           `json:"hasSubmitVerbs"`
}

type VerbSummary struct {
	Name         string           `json:"name"`
	FullPath     string           `json:"fullPath"`
	CommandPath  string           `json:"commandPath"`
	FunctionName string           `json:"functionName"`
	Short        string           `json:"short"`
	Long         string           `json:"long"`
	OutputMode   string           `json:"outputMode"`
	Parents      []string         `json:"parents"`
	SourceFile   string           `json:"sourceFile"`
	Module       string           `json:"module"`
	Sections     []SectionSummary `json:"sections"`
}

type SectionSummary struct {
	Slug        string         `json:"slug"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Fields      []FieldSummary `json:"fields"`
}

type FieldSummary struct {
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Help       string      `json:"help,omitempty"`
	ShortFlag  string      `json:"shortFlag,omitempty"`
	Default    interface{} `json:"default,omitempty"`
	Choices    []string    `json:"choices,omitempty"`
	Required   bool        `json:"required"`
	IsArgument bool        `json:"isArgument"`
}

type Service struct {
	siteRegistry *siteregistry.Registry
}

func NewService(siteRegistry *siteregistry.Registry) *Service {
	return &Service{siteRegistry: siteRegistry}
}

func (s *Service) ListSites() []SiteSummary {
	if s == nil || s.siteRegistry == nil {
		return nil
	}
	ret := make([]SiteSummary, 0, len(s.siteRegistry.List()))
	for _, def := range s.siteRegistry.List() {
		ret = append(ret, SiteSummary{
			Name:             def.Name,
			DatabaseFileName: def.DatabaseFileName,
			HasScripts:       def.ScriptsFS != nil && def.ScriptsRoot != "",
			HasSubmitVerbs:   def.VerbsFS != nil && def.VerbsRoot != "",
		})
	}
	return ret
}

func (s *Service) GetSite(site model.SiteName) (*SiteSummary, siteregistry.Definition, error) {
	if s == nil || s.siteRegistry == nil {
		return nil, siteregistry.Definition{}, fmt.Errorf("catalog service is not configured")
	}
	def, ok := s.siteRegistry.Get(site)
	if !ok {
		return nil, siteregistry.Definition{}, &NotFoundError{Kind: "site", Name: string(site)}
	}
	summary := &SiteSummary{
		Name:             def.Name,
		DatabaseFileName: def.DatabaseFileName,
		HasScripts:       def.ScriptsFS != nil && def.ScriptsRoot != "",
		HasSubmitVerbs:   def.VerbsFS != nil && def.VerbsRoot != "",
	}
	return summary, def, nil
}

func (s *Service) ListVerbs(site model.SiteName) ([]VerbSummary, error) {
	_, def, err := s.GetSite(site)
	if err != nil {
		return nil, err
	}
	siteVerbs, err := submitverbs.LoadSiteVerbs(def)
	if err != nil {
		return nil, err
	}

	ret := make([]VerbSummary, 0, len(siteVerbs.Registry.Verbs()))
	for _, verb := range siteVerbs.Registry.Verbs() {
		command := siteVerbs.CommandsByName[verb.Name]
		ret = append(ret, buildVerbSummary(site, verb, command))
	}

	sort.Slice(ret, func(i, j int) bool { return ret[i].Name < ret[j].Name })
	return ret, nil
}

func (s *Service) GetVerb(site model.SiteName, verbName string) (*VerbSummary, *jsverbs.VerbSpec, cmds.Command, siteregistry.Definition, error) {
	_, def, err := s.GetSite(site)
	if err != nil {
		return nil, nil, nil, siteregistry.Definition{}, err
	}
	siteVerbs, err := submitverbs.LoadSiteVerbs(def)
	if err != nil {
		return nil, nil, nil, siteregistry.Definition{}, err
	}
	verb, command, ok := siteVerbs.Resolve(verbName)
	if !ok {
		return nil, nil, nil, siteregistry.Definition{}, &NotFoundError{Kind: "verb", Name: string(site) + "/" + verbName}
	}
	summary := buildVerbSummary(site, verb, command)
	return &summary, verb, command, def, nil
}

func buildVerbSummary(site model.SiteName, verb *jsverbs.VerbSpec, command cmds.Command) VerbSummary {
	description := command.Description()
	return VerbSummary{
		Name:         verb.Name,
		FullPath:     verb.FullPath(),
		CommandPath:  "site " + string(site) + " run " + verb.Name,
		FunctionName: verb.FunctionName,
		Short:        verb.Short,
		Long:         verb.Long,
		OutputMode:   verb.OutputMode,
		Parents:      append([]string(nil), verb.Parents...),
		SourceFile:   verb.File.RelPath,
		Module:       verb.File.ModulePath,
		Sections:     summarizeSections(description.Schema),
	}
}

func summarizeSections(s *schema.Schema) []SectionSummary {
	if s == nil {
		return nil
	}
	ret := make([]SectionSummary, 0, s.Len())
	s.ForEach(func(_ string, section schema.Section) {
		ret = append(ret, SectionSummary{
			Slug:        section.GetSlug(),
			Title:       section.GetName(),
			Description: section.GetDescription(),
			Fields:      summarizeFields(section.GetDefinitions()),
		})
	})
	return ret
}

func summarizeFields(defs *fields.Definitions) []FieldSummary {
	if defs == nil {
		return nil
	}
	ret := make([]FieldSummary, 0, defs.Len())
	defs.ForEach(func(def *fields.Definition) {
		var defaultValue interface{}
		if def.Default != nil {
			defaultValue = *def.Default
		}
		ret = append(ret, FieldSummary{
			Name:       def.Name,
			Type:       string(def.Type),
			Help:       def.Help,
			ShortFlag:  def.ShortFlag,
			Default:    defaultValue,
			Choices:    append([]string(nil), def.Choices...),
			Required:   def.Required,
			IsArgument: def.IsArgument,
		})
	})
	sort.Slice(ret, func(i, j int) bool { return ret[i].Name < ret[j].Name })
	return ret
}

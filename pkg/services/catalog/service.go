package catalog

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

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
	OriginKind       string         `json:"originKind"`
	ManifestPath     string         `json:"manifestPath,omitempty"`
	HasScripts       bool           `json:"hasScripts"`
	HasSubmitVerbs   bool           `json:"hasSubmitVerbs"`
}

type QueuePolicySummary struct {
	Queue       model.QueueKey `json:"queue"`
	MaxInFlight int            `json:"maxInFlight"`
	RateLimit   *RateLimitInfo `json:"rateLimit,omitempty"`
}

type RateLimitInfo struct {
	Kind          string  `json:"kind"`
	RatePerSecond float64 `json:"ratePerSecond"`
	Burst         int     `json:"burst"`
}

type SiteDetail struct {
	SiteSummary
	VerbCount     int                  `json:"verbCount"`
	ScriptCount   int                  `json:"scriptCount"`
	Scripts       []string             `json:"scripts"`
	QueuePolicies []QueuePolicySummary `json:"queuePolicies"`
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
			OriginKind:       string(def.Origin),
			ManifestPath:     def.ManifestPath,
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
		OriginKind:       string(def.Origin),
		ManifestPath:     def.ManifestPath,
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

func (s *Service) GetSiteDetail(site model.SiteName) (*SiteDetail, error) {
	summary, def, err := s.GetSite(site)
	if err != nil {
		return nil, err
	}
	scripts, _ := s.ListScripts(site)
	verbs, _ := s.ListVerbs(site)

	detail := &SiteDetail{
		SiteSummary:   *summary,
		VerbCount:     len(verbs),
		ScriptCount:   len(scripts),
		Scripts:       scripts,
		QueuePolicies: buildQueuePolicies(def),
	}
	return detail, nil
}

func (s *Service) GetAllQueuePolicies() map[string]QueuePolicySummary {
	if s == nil || s.siteRegistry == nil {
		return nil
	}
	result := map[string]QueuePolicySummary{}
	for _, def := range s.siteRegistry.List() {
		for queue, policy := range def.QueuePolicies {
			normalized := policy.Normalize()
			summary := QueuePolicySummary{
				Queue:       queue,
				MaxInFlight: normalized.MaxInFlight,
			}
			if normalized.RateLimit != nil {
				summary.RateLimit = &RateLimitInfo{
					Kind:          string(normalized.RateLimit.Kind),
					RatePerSecond: normalized.RateLimit.RatePerSecond,
					Burst:         normalized.RateLimit.Burst,
				}
			}
			result[string(def.Name)+":"+string(queue)] = summary
		}
	}
	return result
}

func buildQueuePolicies(def siteregistry.Definition) []QueuePolicySummary {
	if len(def.QueuePolicies) == 0 {
		return nil
	}
	ret := make([]QueuePolicySummary, 0, len(def.QueuePolicies))
	for queue, policy := range def.QueuePolicies {
		normalized := policy.Normalize()
		summary := QueuePolicySummary{
			Queue:       queue,
			MaxInFlight: normalized.MaxInFlight,
		}
		if normalized.RateLimit != nil {
			summary.RateLimit = &RateLimitInfo{
				Kind:          string(normalized.RateLimit.Kind),
				RatePerSecond: normalized.RateLimit.RatePerSecond,
				Burst:         normalized.RateLimit.Burst,
			}
		}
		ret = append(ret, summary)
	}
	return ret
}

func (s *Service) ListScripts(site model.SiteName) ([]string, error) {
	_, def, err := s.GetSite(site)
	if err != nil {
		return nil, err
	}
	if def.ScriptsFS == nil || def.ScriptsRoot == "" {
		return []string{}, nil
	}

	var scripts []string
	scriptsFS, fErr := fs.Sub(def.ScriptsFS, def.ScriptsRoot)
	if fErr != nil {
		return nil, fmt.Errorf("open scripts fs: %w", fErr)
	}
	_ = fs.WalkDir(scriptsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(path, ".js") {
			scripts = append(scripts, path)
		}
		return nil
	})
	sort.Strings(scripts)
	return scripts, nil
}

func (s *Service) ReadScript(site model.SiteName, scriptPath string) (string, error) {
	_, def, err := s.GetSite(site)
	if err != nil {
		return "", err
	}
	if def.ScriptsFS == nil || def.ScriptsRoot == "" {
		return "", &NotFoundError{Kind: "script", Name: string(site) + "/" + scriptPath}
	}

	// Prevent path traversal
	cleaned := filepath.ToSlash(filepath.Clean(scriptPath))
	if strings.Contains(cleaned, "..") {
		return "", &NotFoundError{Kind: "script", Name: string(site) + "/" + scriptPath}
	}

	fullPath := filepath.ToSlash(filepath.Join(def.ScriptsRoot, cleaned))
	body, fErr := fs.ReadFile(def.ScriptsFS, fullPath)
	if fErr != nil {
		return "", &NotFoundError{Kind: "script", Name: string(site) + "/" + scriptPath}
	}
	return string(body), nil
}

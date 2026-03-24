package submitverbs

import (
	"fmt"
	"strings"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/go-go-goja/pkg/jsverbs"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
)

type SiteVerbs struct {
	Definition     siteregistry.Definition
	Registry       *jsverbs.Registry
	VerbsByName    map[string]*jsverbs.VerbSpec
	CommandsByName map[string]cmds.Command
}

func LoadSiteVerbs(def siteregistry.Definition) (*SiteVerbs, error) {
	if def.VerbsFS == nil || strings.TrimSpace(def.VerbsRoot) == "" {
		return nil, fmt.Errorf("site %s does not define JS submit verbs", def.Name)
	}

	registry, err := jsverbs.ScanFS(def.VerbsFS, def.VerbsRoot, jsverbs.ScanOptions{
		IncludePublicFunctions: false,
		Extensions:             []string{".js"},
		FailOnErrorDiagnostics: true,
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s submit verbs: %w", def.Name, err)
	}

	commands, err := registry.Commands()
	if err != nil {
		return nil, fmt.Errorf("build %s submit verb commands: %w", def.Name, err)
	}

	ret := &SiteVerbs{
		Definition:     def,
		Registry:       registry,
		VerbsByName:    map[string]*jsverbs.VerbSpec{},
		CommandsByName: map[string]cmds.Command{},
	}
	for _, verb := range registry.Verbs() {
		ret.VerbsByName[verb.Name] = verb
	}
	for _, command := range commands {
		ret.CommandsByName[command.Description().Name] = command
	}

	return ret, nil
}

func (s *SiteVerbs) Resolve(verbName string) (*jsverbs.VerbSpec, cmds.Command, bool) {
	if s == nil {
		return nil, nil, false
	}
	verb, ok := s.VerbsByName[verbName]
	if !ok {
		return nil, nil, false
	}
	command, ok := s.CommandsByName[verbName]
	if !ok {
		return nil, nil, false
	}
	return verb, command, true
}

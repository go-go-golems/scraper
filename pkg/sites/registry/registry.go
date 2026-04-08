package registry

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/spf13/cobra"
)

type Definition struct {
	Name                    model.SiteName
	DatabaseFileName        string
	Origin                  DefinitionOrigin
	ManifestPath            string
	ScriptsFS               fs.FS
	ScriptsRoot             string
	VerbsFS                 fs.FS
	VerbsRoot               string
	Modules                 []gggengine.ModuleSpec
	SQLMigrationsFS         fs.FS
	SQLMigrationsRoot       string
	JSMigrationsFS          fs.FS
	JSMigrationsRoot        string
	HelpFS                  fs.FS
	HelpRoot                string
	QueuePolicies           map[model.QueueKey]model.QueuePolicy
	RuntimeModuleRegistrars []gggengine.RuntimeModuleRegistrar
	RegisterCLI             func(root *cobra.Command) error
}

type DefinitionOrigin string

const (
	DefinitionOriginGo       DefinitionOrigin = "go"
	DefinitionOriginManifest DefinitionOrigin = "manifest"
)

type Registry struct {
	sites map[model.SiteName]Definition
}

func New() *Registry {
	return &Registry{
		sites: map[model.SiteName]Definition{},
	}
}

func (r *Registry) Register(def Definition) error {
	if def.Name == "" {
		return fmt.Errorf("site name is required")
	}
	if def.Origin == "" {
		def.Origin = DefinitionOriginGo
	}
	if _, ok := r.sites[def.Name]; ok {
		return fmt.Errorf("site already registered: %s", def.Name)
	}

	r.sites[def.Name] = def
	return nil
}

func (r *Registry) Get(name model.SiteName) (Definition, bool) {
	def, ok := r.sites[name]
	return def, ok
}

func (r *Registry) List() []Definition {
	names := make([]string, 0, len(r.sites))
	for name := range r.sites {
		names = append(names, string(name))
	}
	sort.Strings(names)

	ret := make([]Definition, 0, len(names))
	for _, name := range names {
		ret = append(ret, r.sites[model.SiteName(name)])
	}

	return ret
}

func (r *Registry) QueuePolicyProvider() func(ctx context.Context, site model.SiteName, queue model.QueueKey) model.QueuePolicy {
	return func(ctx context.Context, site model.SiteName, queue model.QueueKey) model.QueuePolicy {
		if r == nil {
			return model.DefaultQueuePolicy()
		}
		def, ok := r.Get(site)
		if !ok {
			return model.DefaultQueuePolicy()
		}
		if def.QueuePolicies == nil {
			return model.DefaultQueuePolicy()
		}
		policy, ok := def.QueuePolicies[queue]
		if !ok {
			return model.DefaultQueuePolicy()
		}
		return policy.Normalize()
	}
}

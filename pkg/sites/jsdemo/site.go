package jsdemo

import (
	"embed"

	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
)

//go:embed scripts/*.js scripts/lib/*.js verbs/*.js migrations/*.sql
var siteFS embed.FS

func Definition() siteregistry.Definition {
	return siteregistry.Definition{
		Name:              model.SiteName("js-demo"),
		DatabaseFileName:  "js-demo.db",
		ScriptsFS:         siteFS,
		ScriptsRoot:       "scripts",
		VerbsFS:           siteFS,
		VerbsRoot:         "verbs",
		Modules:           []gggengine.ModuleSpec{gggengine.DefaultRegistryModules()},
		SQLMigrationsFS:   siteFS,
		SQLMigrationsRoot: "migrations",
	}
}

func Register(registry *siteregistry.Registry) error {
	return registry.Register(Definition())
}

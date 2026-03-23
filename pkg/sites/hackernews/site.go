package hackernews

import (
	"embed"
	"io/fs"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
)

//go:embed scripts/*.js scripts/lib/*.js migrations/*.sql fixtures/*.html
var siteFS embed.FS

func Definition() siteregistry.Definition {
	return siteregistry.Definition{
		Name:              model.SiteName("hackernews"),
		DatabaseFileName:  "hackernews.db",
		ScriptsFS:         siteFS,
		ScriptsRoot:       "scripts",
		SQLMigrationsFS:   siteFS,
		SQLMigrationsRoot: "migrations",
		RegisterCLI:       registerCLI,
	}
}

func Register(registry *siteregistry.Registry) error {
	return registry.Register(Definition())
}

func ReadFixture(name string) ([]byte, error) {
	return fs.ReadFile(siteFS, "fixtures/"+name)
}

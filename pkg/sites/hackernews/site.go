package hackernews

import (
	"embed"
	"io/fs"
	"sync"

	sitemanifest "github.com/go-go-golems/scraper/pkg/sites/manifest"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
)

//go:embed site.yaml scripts/*.js scripts/lib/*.js verbs/*.js migrations/*.sql fixtures/*.html
var siteFS embed.FS

var (
	definitionOnce sync.Once
	definition     siteregistry.Definition
	definitionErr  error
)

func Definition() siteregistry.Definition {
	definitionOnce.Do(func() {
		definition, definitionErr = sitemanifest.LoadDefinition(siteFS, "")
	})
	if definitionErr != nil {
		panic(definitionErr)
	}
	return definition
}

func Register(registry *siteregistry.Registry) error {
	return registry.Register(Definition())
}

func ReadFixture(name string) ([]byte, error) {
	return fs.ReadFile(siteFS, "fixtures/"+name)
}

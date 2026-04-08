package nereval

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
	siteOnce   sync.Once
	siteDef    siteregistry.Definition
	siteDefErr error
)

func Definition() siteregistry.Definition {
	siteOnce.Do(func() {
		siteDef, siteDefErr = sitemanifest.LoadDefinition(siteFS, "")
	})
	if siteDefErr != nil {
		panic(siteDefErr)
	}
	return siteDef
}

func Register(reg *siteregistry.Registry) error {
	return reg.Register(Definition())
}

func ReadFixture(name string) ([]byte, error) {
	return fs.ReadFile(siteFS, "fixtures/"+name)
}

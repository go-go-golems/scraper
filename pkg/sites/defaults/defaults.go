package defaults

import (
	sitemanifest "github.com/go-go-golems/scraper/pkg/sites/manifest"
	"github.com/go-go-golems/scraper/pkg/sites/registry"

	"github.com/go-go-golems/scraper/pkg/sites/hackernews"
	"github.com/go-go-golems/scraper/pkg/sites/jsdemo"
	"github.com/go-go-golems/scraper/pkg/sites/nereval"
	"github.com/go-go-golems/scraper/pkg/sites/slashdot"
)

// NewRegistry creates a registry with all built-in sites registered.
func NewRegistry() (*registry.Registry, error) {
	ret := registry.New()

	if err := ret.Register(jsdemo.Definition()); err != nil {
		return nil, err
	}
	if err := ret.Register(hackernews.Definition()); err != nil {
		return nil, err
	}
	if err := ret.Register(slashdot.Definition()); err != nil {
		return nil, err
	}
	if err := ret.Register(nereval.Definition()); err != nil {
		return nil, err
	}

	return ret, nil
}

// LoadExternalSites loads additional sites from a directory of site manifests.
// Each subdirectory containing a site.yaml is registered as a site.
// Returns nil if dir is empty.
func LoadExternalSites(r *registry.Registry, dir string) error {
	if dir == "" {
		return nil
	}
	return sitemanifest.RegisterDir(r, dir)
}

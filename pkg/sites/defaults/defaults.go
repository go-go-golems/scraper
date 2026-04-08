package defaults

import (
	sitemanifest "github.com/go-go-golems/scraper/pkg/sites/manifest"
	"github.com/go-go-golems/scraper/pkg/sites/registry"
)

// NewRegistryFromDirs creates a registry loaded from one or more manifest directories.
// Each directory is scanned for subdirectories containing site.yaml files.
// If no directories are provided, returns an empty registry.
func NewRegistryFromDirs(dirs ...string) (*registry.Registry, error) {
	ret := registry.New()
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := sitemanifest.RegisterDir(ret, dir); err != nil {
			return nil, err
		}
	}
	return ret, nil
}

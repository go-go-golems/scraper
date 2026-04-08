package manifest

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"
)

const DefaultManifestPath = "site.yaml"

func Load(siteFS fs.FS, manifestPath string) (Site, error) {
	if siteFS == nil {
		return Site{}, errors.New("site filesystem is required")
	}
	manifestPath = normalizeManifestPath(manifestPath)
	raw, err := fs.ReadFile(siteFS, manifestPath)
	if err != nil {
		return Site{}, errors.Wrapf(err, "read manifest %q", manifestPath)
	}

	var site Site
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&site); err != nil {
		return Site{}, errors.Wrapf(err, "decode manifest %q", manifestPath)
	}
	if err := ValidateSite(site); err != nil {
		return Site{}, errors.Wrapf(err, "validate manifest %q", manifestPath)
	}
	return site, nil
}

func LoadDefinition(siteFS fs.FS, manifestPath string) (siteregistry.Definition, error) {
	site, err := Load(siteFS, manifestPath)
	if err != nil {
		return siteregistry.Definition{}, err
	}
	modules, err := ResolveModules(site.Modules)
	if err != nil {
		return siteregistry.Definition{}, errors.Wrap(err, "resolve manifest modules")
	}
	def := siteregistry.Definition{
		Name:             site.Name,
		DatabaseFileName: site.DatabaseFileName,
		Origin:           siteregistry.DefinitionOriginManifest,
		ManifestPath:     normalizeManifestPath(manifestPath),
		Modules:          modules,
		QueuePolicies:    buildQueuePolicies(site.QueuePolicies),
	}
	attachRoots(&def, siteFS, site)
	return def, nil
}

func RegisterFS(reg *siteregistry.Registry, siteFS fs.FS, manifestPath string) error {
	if reg == nil {
		return errors.New("registry is required")
	}
	def, err := LoadDefinition(siteFS, manifestPath)
	if err != nil {
		return err
	}
	return reg.Register(def)
}

func normalizeManifestPath(manifestPath string) string {
	if manifestPath == "" {
		return DefaultManifestPath
	}
	return manifestPath
}

// RegisterDir walks rootDir and loads every site.yaml found, registering each
// into reg. Subdirectories of rootDir are treated as individual site packages.
// Each subdirectory must contain a site.yaml at its root.
// Example: rootDir=/etc/scraper/sites → loads /etc/scraper/sites/hackernews/site.yaml
func RegisterDir(reg *siteregistry.Registry, rootDir string) error {
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return errors.Wrapf(err, "read sites root dir %q", rootDir)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		siteDir := filepath.Join(rootDir, entry.Name())
		manifestPath := filepath.Join(siteDir, DefaultManifestPath)
		if _, err := os.Stat(manifestPath); err != nil {
			// no site.yaml in this subdirectory — skip
			continue
		}
		siteFS := os.DirFS(siteDir)
		def, err := LoadDefinition(siteFS, DefaultManifestPath)
		if err != nil {
			return errors.Wrapf(err, "load site manifest from %q", manifestPath)
		}
		if err := reg.Register(def); err != nil {
			return errors.Wrapf(err, "register site %q", def.Name)
		}
	}
	return nil
}

func attachRoots(def *siteregistry.Definition, siteFS fs.FS, site Site) {
	if def == nil {
		return
	}
	if site.ScriptsRoot != "" {
		def.ScriptsFS = siteFS
		def.ScriptsRoot = site.ScriptsRoot
	}
	if site.VerbsRoot != "" {
		def.VerbsFS = siteFS
		def.VerbsRoot = site.VerbsRoot
	}
	if site.SQLMigrationsRoot != "" {
		def.SQLMigrationsFS = siteFS
		def.SQLMigrationsRoot = site.SQLMigrationsRoot
	}
	if site.JSMigrationsRoot != "" {
		def.JSMigrationsFS = siteFS
		def.JSMigrationsRoot = site.JSMigrationsRoot
	}
	if site.HelpRoot != "" {
		def.HelpFS = siteFS
		def.HelpRoot = site.HelpRoot
	}
}

func buildQueuePolicies(items []QueuePolicy) map[model.QueueKey]model.QueuePolicy {
	if len(items) == 0 {
		return nil
	}
	ret := make(map[model.QueueKey]model.QueuePolicy, len(items))
	for _, item := range items {
		policy := model.QueuePolicy{
			MaxInFlight: item.MaxInFlight,
		}
		if item.RateLimit != nil {
			policy.RateLimit = &model.RateLimitPolicy{
				Kind:          item.RateLimit.Kind,
				RatePerSecond: item.RateLimit.RatePerSecond,
				Burst:         item.RateLimit.Burst,
			}
		}
		ret[item.Queue] = policy.Normalize()
	}
	return ret
}

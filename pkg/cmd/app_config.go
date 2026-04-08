package cmd

import (
	"os"
	"path/filepath"
	"strings"

	glazedconfig "github.com/go-go-golems/glazed/pkg/config"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const scraperSitesManifestDirsEnvVar = "SCRAPER_SITES_MANIFEST_DIRS"

type AppConfig struct {
	SitesManifestDirs []string `yaml:"sitesManifestDirs"`
}

func loadAppConfig(appName string) (*AppConfig, error) {
	configPath, err := glazedconfig.ResolveAppConfigPath(appName, "")
	if err != nil {
		return nil, errors.Wrap(err, "could not resolve app config path")
	}

	return loadAppConfigFromPath(configPath)
}

func loadAppConfigFromPath(configPath string) (*AppConfig, error) {
	if configPath == "" {
		return &AppConfig{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.Wrap(err, "could not read app config")
	}

	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "could not parse app config")
	}

	cfg.SitesManifestDirs = normalizeManifestDirs(cfg.SitesManifestDirs)
	return &cfg, nil
}

func collectSitesManifestDirs(appName string, bootstrapDirs []string) ([]string, error) {
	cfg, err := loadAppConfig(appName)
	if err != nil {
		return nil, err
	}

	dirs := append([]string{}, cfg.SitesManifestDirs...)
	dirs = append(dirs, sitesManifestDirsFromEnv()...)
	dirs = append(dirs, bootstrapDirs...)

	return normalizeManifestDirs(dirs), nil
}

func sitesManifestDirsFromEnv() []string {
	value, ok := os.LookupEnv(scraperSitesManifestDirsEnvVar)
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}

	return normalizeManifestDirs(filepath.SplitList(value))
}

func normalizeManifestDirs(dirs []string) []string {
	ret := make([]string, 0, len(dirs))
	seen := map[string]struct{}{}

	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		dir = os.ExpandEnv(dir)
		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		ret = append(ret, dir)
	}

	return ret
}

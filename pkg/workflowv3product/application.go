package workflowv3product

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3runtime"
	"github.com/go-go-golems/scraper/pkg/workflowv3sqlite"
)

type Config struct {
	DatabasePath     string
	ArtifactRoot     string
	TaskPackages     []string
	LeaseDuration    time.Duration
	PollInterval     time.Duration
	Capacities       map[string]int
	MaxArtifactBytes int64
}

func DefaultConfig() Config {
	return Config{
		DatabasePath: "state/workflow-v3.db", ArtifactRoot: "state/workflow-v3-artifacts",
		TaskPackages: []string{"cookbook-linear"}, LeaseDuration: 30 * time.Second,
		PollInterval:     100 * time.Millisecond,
		Capacities:       map[string]int{workflowv3.ResourceCPUDefault: 4},
		MaxArtifactBytes: workflowv3.DefaultMaxArtifactBytes,
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.DatabasePath) == "" || strings.TrimSpace(c.ArtifactRoot) == "" {
		return fmt.Errorf("workflow database path and artifact root are required")
	}
	if c.LeaseDuration <= 0 || c.PollInterval <= 0 {
		return fmt.Errorf("workflow lease duration and poll interval must be positive")
	}
	if len(c.Capacities) == 0 {
		return fmt.Errorf("workflow worker capacities are required")
	}
	for resource, capacity := range c.Capacities {
		if strings.TrimSpace(resource) == "" || capacity < 1 {
			return fmt.Errorf("workflow capacity %q must be positive", resource)
		}
	}
	if c.MaxArtifactBytes <= 0 {
		return fmt.Errorf("workflow maximum artifact bytes must be positive")
	}
	return nil
}

type AuthoringEnvironment struct {
	Packages *PackageSet
}

func NewAuthoringEnvironment(selected []string, available ...TaskPackage) (*AuthoringEnvironment, error) {
	packages, err := BuildPackageSet(selected, available...)
	if err != nil {
		return nil, err
	}
	return &AuthoringEnvironment{Packages: packages}, nil
}

func (e *AuthoringEnvironment) Author(ctx context.Context, source string) (workflowmodule.AuthoringResult, error) {
	if e == nil || e.Packages == nil {
		return workflowmodule.AuthoringResult{}, fmt.Errorf("workflow authoring environment is required")
	}
	return workflowmodule.Author(ctx, source, e.Packages.Catalog(), e.Packages.Descriptors()...)
}

type Application struct {
	Config     Config
	Authoring  *AuthoringEnvironment
	Store      *workflowv3sqlite.Store
	Artifacts  *workflowv3.FileArtifactStore
	Registry   *workflowv3runtime.RegistryManager
	Engine     *workflowv3runtime.Engine
	Dispatcher *workflowv3runtime.Dispatcher
}

func Open(ctx context.Context, config Config, available ...TaskPackage) (*Application, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	authoring, err := NewAuthoringEnvironment(config.TaskPackages, available...)
	if err != nil {
		return nil, err
	}
	if err := ensureParent(config.DatabasePath); err != nil {
		return nil, err
	}
	store, err := workflowv3sqlite.Open(ctx, config.DatabasePath)
	if err != nil {
		return nil, err
	}
	artifacts, err := workflowv3.NewFileArtifactStore(config.ArtifactRoot, config.MaxArtifactBytes)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	registry, err := workflowv3runtime.NewRegistryManager(authoring.Packages.Registry())
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	engine := &workflowv3runtime.Engine{
		Store: store, Registry: registry, Artifacts: artifacts,
		Modules: authoring.Packages.Modules(), LeaseDuration: config.LeaseDuration,
	}
	capacities := make(map[string]int, len(config.Capacities))
	for resource, capacity := range config.Capacities {
		capacities[resource] = capacity
	}
	dispatcher := &workflowv3runtime.Dispatcher{
		Engine: engine, Capacities: capacities, PollInterval: config.PollInterval,
	}
	return &Application{
		Config: config, Authoring: authoring, Store: store, Artifacts: artifacts,
		Registry: registry, Engine: engine, Dispatcher: dispatcher,
	}, nil
}

func (a *Application) Close() error {
	if a == nil || a.Store == nil {
		return nil
	}
	return a.Store.Close()
}

func ensureParent(path string) error {
	parent := filepath.Dir(path)
	if parent == "." {
		return nil
	}
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create workflow database parent directory: %w", err)
	}
	return nil
}

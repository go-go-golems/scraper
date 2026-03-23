package runtime

import (
	"fmt"

	"github.com/dop251/goja_nodejs/require"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	databasemod "github.com/go-go-golems/go-go-goja/modules/database"
)

type DatabaseRegistrarConfig struct {
	ScraperDB databasemod.QueryExecer
	SiteDB    databasemod.QueryExecer
}

type DatabaseRegistrar struct {
	config DatabaseRegistrarConfig
}

func NewDatabaseRegistrar(config DatabaseRegistrarConfig) *DatabaseRegistrar {
	return &DatabaseRegistrar{config: config}
}

func (r *DatabaseRegistrar) ID() string {
	return "scraper-preconfigured-databases"
}

func (r *DatabaseRegistrar) RegisterRuntimeModules(ctx *gggengine.RuntimeModuleContext, reg *require.Registry) error {
	if reg == nil {
		return fmt.Errorf("require registry is nil")
	}

	if r.config.ScraperDB != nil {
		registerDBModule(reg, "scraper-db", r.config.ScraperDB)
		if ctx != nil {
			ctx.SetValue("scraper-db.module", "scraper-db")
		}
	}
	if r.config.SiteDB != nil {
		registerDBModule(reg, "site-db", r.config.SiteDB)
		if ctx != nil {
			ctx.SetValue("site-db.module", "site-db")
		}
	}

	return nil
}

func registerDBModule(reg *require.Registry, name string, db databasemod.QueryExecer) {
	module := databasemod.New(
		databasemod.WithName(name),
		databasemod.WithPreconfiguredDB(db),
	)
	reg.RegisterNativeModule(module.Name(), module.Loader)
}

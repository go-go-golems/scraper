package manifest

import (
	"testing"
	"testing/fstest"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	siteregistry "github.com/go-go-golems/scraper/pkg/sites/registry"
)

func TestLoadDefinitionMapsManifestIntoRegistryDefinition(t *testing.T) {
	t.Parallel()

	siteFS := fstest.MapFS{
		"site.yaml": {
			Data: []byte(`
name: js-demo
databaseFileName: js-demo.db
scriptsRoot: scripts
verbsRoot: verbs
sqlMigrationsRoot: migrations
helpRoot: help
modules:
  - default-registry
queuePolicies:
  - queue: site:js-demo:http
    maxInFlight: 2
    rateLimit:
      ratePerSecond: 1
      burst: 2
`),
		},
	}

	def, err := LoadDefinition(siteFS, "")
	if err != nil {
		t.Fatalf("LoadDefinition() error = %v", err)
	}

	if def.Name != model.SiteName("js-demo") {
		t.Fatalf("def.Name = %q", def.Name)
	}
	if def.DatabaseFileName != "js-demo.db" {
		t.Fatalf("def.DatabaseFileName = %q", def.DatabaseFileName)
	}
	if def.Origin != siteregistry.DefinitionOriginManifest {
		t.Fatalf("def.Origin = %q", def.Origin)
	}
	if def.ManifestPath != DefaultManifestPath {
		t.Fatalf("def.ManifestPath = %q", def.ManifestPath)
	}
	if def.ScriptsFS == nil || def.ScriptsRoot != "scripts" {
		t.Fatalf("scripts root not attached: %#v", def)
	}
	if def.VerbsFS == nil || def.VerbsRoot != "verbs" {
		t.Fatalf("verbs root not attached: %#v", def)
	}
	if def.SQLMigrationsFS == nil || def.SQLMigrationsRoot != "migrations" {
		t.Fatalf("sql migrations root not attached: %#v", def)
	}
	if def.HelpFS == nil || def.HelpRoot != "help" {
		t.Fatalf("help root not attached: %#v", def)
	}
	if len(def.Modules) != 1 {
		t.Fatalf("len(def.Modules) = %d", len(def.Modules))
	}
	policy, ok := def.QueuePolicies[model.QueueKey("site:js-demo:http")]
	if !ok {
		t.Fatalf("queue policy missing")
	}
	if policy.MaxInFlight != 2 {
		t.Fatalf("policy.MaxInFlight = %d", policy.MaxInFlight)
	}
	if policy.RateLimit == nil {
		t.Fatalf("policy.RateLimit = nil")
	}
	if policy.RateLimit.Kind != model.RateLimitKindTokenBucket {
		t.Fatalf("policy.RateLimit.Kind = %q", policy.RateLimit.Kind)
	}
	if policy.RateLimit.RatePerSecond != 1 {
		t.Fatalf("policy.RateLimit.RatePerSecond = %v", policy.RateLimit.RatePerSecond)
	}
	if policy.RateLimit.Burst != 2 {
		t.Fatalf("policy.RateLimit.Burst = %d", policy.RateLimit.Burst)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	t.Parallel()

	siteFS := fstest.MapFS{
		"site.yaml": {
			Data: []byte(`
name: js-demo
databaseFileName: js-demo.db
unexpected: true
`),
		},
	}

	_, err := Load(siteFS, "")
	if err == nil {
		t.Fatalf("Load() error = nil, want decode error")
	}
}

func TestRegisterFSRegistersManifestDefinition(t *testing.T) {
	t.Parallel()

	siteFS := fstest.MapFS{
		"manifest/site.yaml": {
			Data: []byte(`
name: slashdot
databaseFileName: slashdot.db
scriptsRoot: scripts
verbsRoot: verbs
`),
		},
	}

	reg := siteregistry.New()
	if err := RegisterFS(reg, siteFS, "manifest/site.yaml"); err != nil {
		t.Fatalf("RegisterFS() error = %v", err)
	}

	def, ok := reg.Get(model.SiteName("slashdot"))
	if !ok {
		t.Fatalf("slashdot definition not registered")
	}
	if def.ScriptsRoot != "scripts" || def.VerbsRoot != "verbs" {
		t.Fatalf("registered roots = scripts:%q verbs:%q", def.ScriptsRoot, def.VerbsRoot)
	}
}

func TestRegistryCanMixGoAndManifestDefinitions(t *testing.T) {
	t.Parallel()

	reg := siteregistry.New()
	if err := reg.Register(siteregistry.Definition{
		Name:             model.SiteName("go-site"),
		DatabaseFileName: "go-site.db",
	}); err != nil {
		t.Fatalf("Register() go-site error = %v", err)
	}

	siteFS := fstest.MapFS{
		"site.yaml": {
			Data: []byte(`
name: manifest-site
databaseFileName: manifest-site.db
scriptsRoot: scripts
`),
		},
	}

	if err := RegisterFS(reg, siteFS, ""); err != nil {
		t.Fatalf("RegisterFS() manifest-site error = %v", err)
	}

	if _, ok := reg.Get(model.SiteName("go-site")); !ok {
		t.Fatalf("go-site missing after manifest registration")
	}
	manifestSite, ok := reg.Get(model.SiteName("manifest-site"))
	if !ok {
		t.Fatalf("manifest-site missing after manifest registration")
	}
	if manifestSite.Origin != siteregistry.DefinitionOriginManifest {
		t.Fatalf("manifest-site origin = %q", manifestSite.Origin)
	}
}

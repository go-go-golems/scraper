package manifest

import (
	"testing"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

func TestValidateSiteAcceptsManifestWithModulesAndQueuePolicies(t *testing.T) {
	t.Parallel()

	site := Site{
		Name:              model.SiteName("hackernews"),
		DatabaseFileName:  "hackernews.db",
		ScriptsRoot:       "scripts",
		VerbsRoot:         "verbs",
		SQLMigrationsRoot: "migrations",
		Modules:           []string{ModuleDefaultRegistry},
		QueuePolicies: []QueuePolicy{
			{
				Queue:       model.QueueKey("site:hackernews:http"),
				MaxInFlight: 1,
				RateLimit: &RateLimitPolicy{
					RatePerSecond: 1,
					Burst:         1,
				},
			},
		},
	}

	if err := ValidateSite(site); err != nil {
		t.Fatalf("ValidateSite() error = %v", err)
	}
}

func TestValidateSiteRejectsUnknownModule(t *testing.T) {
	t.Parallel()

	site := Site{
		Name:             model.SiteName("js-demo"),
		DatabaseFileName: "js-demo.db",
		Modules:          []string{"unknown"},
	}

	err := ValidateSite(site)
	if err == nil {
		t.Fatalf("ValidateSite() error = nil, want unsupported module error")
	}
	if got := err.Error(); got != `site.modules contains unsupported module "unknown"` {
		t.Fatalf("ValidateSite() error = %q", got)
	}
}

func TestValidateSiteRejectsEscapingRoot(t *testing.T) {
	t.Parallel()

	site := Site{
		Name:             model.SiteName("slashdot"),
		DatabaseFileName: "slashdot.db",
		ScriptsRoot:      "../scripts",
	}

	err := ValidateSite(site)
	if err == nil {
		t.Fatalf("ValidateSite() error = nil, want root validation error")
	}
	if got := err.Error(); got != "site.scriptsRoot must not escape the site filesystem" {
		t.Fatalf("ValidateSite() error = %q", got)
	}
}

func TestValidateSiteRejectsDuplicateQueuePolicies(t *testing.T) {
	t.Parallel()

	site := Site{
		Name:             model.SiteName("slashdot"),
		DatabaseFileName: "slashdot.db",
		QueuePolicies: []QueuePolicy{
			{Queue: model.QueueKey("site:slashdot:http")},
			{Queue: model.QueueKey("site:slashdot:http")},
		},
	}

	err := ValidateSite(site)
	if err == nil {
		t.Fatalf("ValidateSite() error = nil, want duplicate queue error")
	}
	if got := err.Error(); got != `site.queuePolicies contains duplicate queue "site:slashdot:http"` {
		t.Fatalf("ValidateSite() error = %q", got)
	}
}

func TestValidateSiteRejectsInvalidRateLimit(t *testing.T) {
	t.Parallel()

	site := Site{
		Name:             model.SiteName("hackernews"),
		DatabaseFileName: "hackernews.db",
		QueuePolicies: []QueuePolicy{
			{
				Queue: model.QueueKey("site:hackernews:http"),
				RateLimit: &RateLimitPolicy{
					RatePerSecond: 0,
					Burst:         1,
				},
			},
		},
	}

	err := ValidateSite(site)
	if err == nil {
		t.Fatalf("ValidateSite() error = nil, want rate limit validation error")
	}
	if got := err.Error(); got != `site.queuePolicies["site:hackernews:http"].rateLimit.ratePerSecond must be > 0` {
		t.Fatalf("ValidateSite() error = %q", got)
	}
}

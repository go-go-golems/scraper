package manifest

import "github.com/go-go-golems/scraper/pkg/engine/model"

type Site struct {
	Name              model.SiteName `yaml:"name"`
	DatabaseFileName  string         `yaml:"databaseFileName"`
	ScriptsRoot       string         `yaml:"scriptsRoot,omitempty"`
	VerbsRoot         string         `yaml:"verbsRoot,omitempty"`
	SQLMigrationsRoot string         `yaml:"sqlMigrationsRoot,omitempty"`
	JSMigrationsRoot  string         `yaml:"jsMigrationsRoot,omitempty"`
	HelpRoot          string         `yaml:"helpRoot,omitempty"`
	Modules           []string       `yaml:"modules,omitempty"`
	QueuePolicies     []QueuePolicy  `yaml:"queuePolicies,omitempty"`
}

type QueuePolicy struct {
	Queue       model.QueueKey   `yaml:"queue"`
	MaxInFlight int              `yaml:"maxInFlight,omitempty"`
	RateLimit   *RateLimitPolicy `yaml:"rateLimit,omitempty"`
}

type RateLimitPolicy struct {
	Kind          model.RateLimitKind `yaml:"kind,omitempty"`
	RatePerSecond float64             `yaml:"ratePerSecond,omitempty"`
	Burst         int                 `yaml:"burst,omitempty"`
}

func (s Site) Validate() error {
	return ValidateSite(s)
}

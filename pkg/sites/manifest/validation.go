package manifest

import (
	"fmt"
	"path"
	"strings"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/pkg/errors"
)

func ValidateSite(site Site) error {
	if site.Name == "" {
		return errors.New("site.name is required")
	}
	if strings.TrimSpace(string(site.Name)) != string(site.Name) {
		return errors.New("site.name must not contain leading or trailing whitespace")
	}
	if site.DatabaseFileName == "" {
		return errors.New("site.databaseFileName is required")
	}
	if err := validateDatabaseFileName(site.DatabaseFileName); err != nil {
		return err
	}
	if err := validateOptionalRelativeRoot("site.scriptsRoot", site.ScriptsRoot); err != nil {
		return err
	}
	if err := validateOptionalRelativeRoot("site.verbsRoot", site.VerbsRoot); err != nil {
		return err
	}
	if err := validateOptionalRelativeRoot("site.sqlMigrationsRoot", site.SQLMigrationsRoot); err != nil {
		return err
	}
	if err := validateOptionalRelativeRoot("site.jsMigrationsRoot", site.JSMigrationsRoot); err != nil {
		return err
	}
	if err := validateOptionalRelativeRoot("site.helpRoot", site.HelpRoot); err != nil {
		return err
	}
	if err := validateModules(site.Modules); err != nil {
		return err
	}
	if err := validateQueuePolicies(site.QueuePolicies); err != nil {
		return err
	}
	return nil
}

func validateDatabaseFileName(name string) error {
	if strings.TrimSpace(name) != name {
		return errors.New("site.databaseFileName must not contain leading or trailing whitespace")
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == "" {
		return errors.New("site.databaseFileName must not be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return errors.New("site.databaseFileName must be a file name, not a path")
	}
	if strings.Contains(name, "..") {
		return errors.New("site.databaseFileName must not contain '..'")
	}
	return nil
}

func validateOptionalRelativeRoot(fieldName, root string) error {
	if root == "" {
		return nil
	}
	if strings.TrimSpace(root) != root {
		return errors.Errorf("%s must not contain leading or trailing whitespace", fieldName)
	}
	if strings.HasPrefix(root, "/") || strings.HasPrefix(root, "\\") {
		return errors.Errorf("%s must be relative", fieldName)
	}
	cleaned := path.Clean(root)
	if cleaned == "." || cleaned == "" {
		return errors.Errorf("%s must not resolve to '.'", fieldName)
	}
	if cleaned != root {
		return errors.Errorf("%s must already be normalized", fieldName)
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return errors.Errorf("%s must not escape the site filesystem", fieldName)
	}
	return nil
}

func validateModules(modules []string) error {
	seen := map[string]struct{}{}
	for _, moduleID := range modules {
		if moduleID == "" {
			return errors.New("site.modules must not contain empty module IDs")
		}
		if !IsSupportedModule(moduleID) {
			return errors.Errorf("site.modules contains unsupported module %q", moduleID)
		}
		if _, ok := seen[moduleID]; ok {
			return errors.Errorf("site.modules contains duplicate module %q", moduleID)
		}
		seen[moduleID] = struct{}{}
	}
	return nil
}

func validateQueuePolicies(policies []QueuePolicy) error {
	seen := map[model.QueueKey]struct{}{}
	for _, policy := range policies {
		if policy.Queue == "" {
			return errors.New("site.queuePolicies[].queue is required")
		}
		if _, ok := seen[policy.Queue]; ok {
			return errors.Errorf("site.queuePolicies contains duplicate queue %q", policy.Queue)
		}
		seen[policy.Queue] = struct{}{}
		if policy.MaxInFlight < 0 {
			return errors.Errorf("site.queuePolicies[%q].maxInFlight must be >= 0", policy.Queue)
		}
		if policy.RateLimit == nil {
			continue
		}
		if err := validateRateLimit(policy.Queue, *policy.RateLimit); err != nil {
			return err
		}
	}
	return nil
}

func validateRateLimit(queue model.QueueKey, rateLimit RateLimitPolicy) error {
	kind := rateLimit.Kind
	if kind == "" {
		kind = model.RateLimitKindTokenBucket
	}
	if kind != model.RateLimitKindTokenBucket {
		return errors.Errorf("site.queuePolicies[%q].rateLimit.kind %q is not supported", queue, rateLimit.Kind)
	}
	if rateLimit.RatePerSecond <= 0 {
		return errors.Errorf("site.queuePolicies[%q].rateLimit.ratePerSecond must be > 0", queue)
	}
	if rateLimit.Burst <= 0 {
		return errors.Errorf("site.queuePolicies[%q].rateLimit.burst must be > 0", queue)
	}
	return nil
}

func (s Site) String() string {
	return fmt.Sprintf("site(name=%s)", s.Name)
}

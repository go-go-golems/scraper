package manifest

import (
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/pkg/errors"
)

const (
	ModuleDefaultRegistry = "default-registry"
)

var supportedModules = map[string]struct{}{
	ModuleDefaultRegistry: {},
}

func IsSupportedModule(id string) bool {
	_, ok := supportedModules[id]
	return ok
}

func ResolveModules(ids []string) ([]gggengine.RuntimeModuleSpec, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// The new engine implicitly includes all default-registry modules when
	// no explicit modules are given. The "default-registry" ID is now a no-op.
	for _, id := range ids {
		if !IsSupportedModule(id) {
			return nil, errors.Errorf("unsupported module %q", id)
		}
	}
	return nil, nil
}

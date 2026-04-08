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

func ResolveModules(ids []string) ([]gggengine.ModuleSpec, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	ret := make([]gggengine.ModuleSpec, 0, len(ids))
	for _, id := range ids {
		switch id {
		case ModuleDefaultRegistry:
			ret = append(ret, gggengine.DefaultRegistryModules())
		default:
			return nil, errors.Errorf("unsupported module %q", id)
		}
	}
	return ret, nil
}

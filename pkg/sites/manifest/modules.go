package manifest

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

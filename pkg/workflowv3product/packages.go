package workflowv3product

import (
	"fmt"
	"sort"
	"strings"

	workflowmodule "github.com/go-go-golems/scraper/pkg/gojamodules/workflow"
	"github.com/go-go-golems/scraper/pkg/taskpackages/cookbooklinear"
	"github.com/go-go-golems/scraper/pkg/taskpackages/researchfixture"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3runtime"
)

type TaskPackage interface {
	Name() string
	Version() string
	Bundle() (*workflowv3.Bundle, error)
	DescriptorModules() []workflowmodule.DescriptorModule
	RequiredModules() []string
}

type PackageInfo struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	BundleDigest string   `json:"bundleDigest"`
	Tasks        []string `json:"tasks"`
	Modules      []string `json:"modules"`
}

type PackageSet struct {
	registry    *workflowv3.SealedRegistry
	catalog     *workflowv3.Catalog
	descriptors []workflowmodule.DescriptorModule
	modules     *workflowv3runtime.TaskModuleRegistry
	info        []PackageInfo
}

func BuiltinPackages() []TaskPackage {
	return []TaskPackage{cookbooklinear.New(), researchfixture.New()}
}

func BuildPackageSet(selected []string, available ...TaskPackage) (*PackageSet, error) {
	if len(available) == 0 {
		available = BuiltinPackages()
	}
	byName := make(map[string]TaskPackage, len(available))
	for _, candidate := range available {
		if candidate == nil {
			return nil, fmt.Errorf("task package is nil")
		}
		name := strings.TrimSpace(candidate.Name())
		version := strings.TrimSpace(candidate.Version())
		if name == "" || version == "" {
			return nil, fmt.Errorf("task package name and version are required")
		}
		if _, exists := byName[name]; exists {
			return nil, fmt.Errorf("task package %q is registered more than once", name)
		}
		byName[name] = candidate
	}
	if len(selected) == 0 {
		selected = make([]string, 0, len(byName))
		for name := range byName {
			selected = append(selected, name)
		}
		sort.Strings(selected)
	}

	builder := workflowv3.NewRegistryBuilder()
	var descriptors []workflowmodule.DescriptorModule
	seenDescriptors := map[string]struct{}{}
	var factories []workflowv3runtime.TaskModuleFactory
	seenModuleFactories := map[string]struct{}{}
	var info []PackageInfo
	seenSelection := map[string]struct{}{}
	for _, rawName := range selected {
		name := strings.TrimSpace(rawName)
		if _, duplicate := seenSelection[name]; duplicate {
			return nil, fmt.Errorf("task package %q is selected more than once", name)
		}
		seenSelection[name] = struct{}{}
		candidate, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("unknown task package %q", name)
		}
		bundle, err := candidate.Bundle()
		if err != nil {
			return nil, fmt.Errorf("build task package %q: %w", name, err)
		}
		aliases := candidate.RequiredModules()
		for _, alias := range aliases {
			factory, err := builtinModuleFactory(alias)
			if err != nil {
				return nil, fmt.Errorf("task package %q: %w", name, err)
			}
			if _, seen := seenModuleFactories[factory.Alias]; !seen {
				factories = append(factories, factory)
				seenModuleFactories[factory.Alias] = struct{}{}
			}
		}
		if err := builder.AdvertiseModules(aliases...); err != nil {
			return nil, fmt.Errorf("advertise task package %q modules: %w", name, err)
		}
		if err := builder.AddBundle(bundle); err != nil {
			return nil, fmt.Errorf("register task package %q: %w", name, err)
		}
		tasks := bundle.TaskSpecs()
		taskNames := make([]string, 0, len(tasks))
		for _, task := range tasks {
			taskNames = append(taskNames, task.Identity.Kind+"@"+task.Identity.Version)
		}
		sort.Strings(taskNames)
		sort.Strings(aliases)
		info = append(info, PackageInfo{
			Name: name, Version: candidate.Version(), BundleDigest: bundle.Digest(),
			Tasks: taskNames, Modules: aliases,
		})
		for _, descriptor := range candidate.DescriptorModules() {
			if strings.TrimSpace(descriptor.Name) == "" {
				return nil, fmt.Errorf("task package %q has an unnamed descriptor module", name)
			}
			if _, exists := seenDescriptors[descriptor.Name]; exists {
				return nil, fmt.Errorf("descriptor module %q is registered more than once", descriptor.Name)
			}
			seenDescriptors[descriptor.Name] = struct{}{}
			descriptors = append(descriptors, descriptor)
		}
	}
	registry, err := builder.Seal()
	if err != nil {
		return nil, err
	}
	catalog, err := registry.Catalog()
	if err != nil {
		return nil, err
	}
	modules, err := workflowv3runtime.NewTaskModuleRegistry(factories...)
	if err != nil {
		return nil, err
	}
	sort.Slice(info, func(i, j int) bool { return info[i].Name < info[j].Name })
	return &PackageSet{registry: registry, catalog: catalog, descriptors: descriptors, modules: modules, info: info}, nil
}

func (p *PackageSet) Registry() *workflowv3.SealedRegistry           { return p.registry }
func (p *PackageSet) Catalog() *workflowv3.Catalog                   { return p.catalog }
func (p *PackageSet) Modules() *workflowv3runtime.TaskModuleRegistry { return p.modules }
func (p *PackageSet) Descriptors() []workflowmodule.DescriptorModule {
	return append([]workflowmodule.DescriptorModule(nil), p.descriptors...)
}
func (p *PackageSet) Info() []PackageInfo { return append([]PackageInfo(nil), p.info...) }

func builtinModuleFactory(alias string) (workflowv3runtime.TaskModuleFactory, error) {
	switch alias {
	case "fs:input":
		return workflowv3runtime.FSInputModule(), nil
	case researchfixture.OperationModuleAlias:
		return fixtureOperationModule(), nil
	default:
		return workflowv3runtime.TaskModuleFactory{}, fmt.Errorf("required runtime module %q is not available", alias)
	}
}

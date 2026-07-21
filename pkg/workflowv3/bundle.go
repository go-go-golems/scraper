package workflowv3

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"
)

type BundleTask struct {
	TaskKey
	Entrypoint    string            `json:"entrypoint"`
	Inputs        map[string]string `json:"inputs"`
	Outputs       map[string]string `json:"outputs"`
	Modules       []string          `json:"modules,omitempty"`
	ResourceClass string            `json:"resourceClass"`
	Retry         RetryPolicy       `json:"retry"`
	BudgetMaximum *BudgetClaim      `json:"budgetMaximum,omitempty"`
}

type BundleManifest struct {
	Name    string       `json:"name"`
	Version string       `json:"version"`
	ABI     string       `json:"abi"`
	Tasks   []BundleTask `json:"tasks"`
}

type Bundle struct {
	manifest BundleManifest
	files    map[string][]byte
	digest   string
}

type bundleDigestEnvelope struct {
	Manifest BundleManifest     `json:"manifest"`
	Files    []bundleDigestFile `json:"files"`
}

type bundleDigestFile struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   int    `json:"size"`
}

func NewBundle(manifest BundleManifest, files map[string][]byte) (*Bundle, error) {
	if strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Version) == "" {
		return nil, fmt.Errorf("bundle name and version are required")
	}
	if manifest.ABI != TaskABI {
		return nil, fmt.Errorf("bundle ABI %q is not supported", manifest.ABI)
	}
	if len(manifest.Tasks) == 0 || len(files) == 0 {
		return nil, fmt.Errorf("bundle requires tasks and files")
	}
	clonedFiles := make(map[string][]byte, len(files))
	for name, body := range files {
		clean := path.Clean(strings.TrimPrefix(name, "./"))
		if clean == "." || strings.HasPrefix(clean, "../") || path.IsAbs(clean) {
			return nil, fmt.Errorf("invalid bundle file path %q", name)
		}
		if clean != name {
			return nil, fmt.Errorf("bundle file path %q is not canonical", name)
		}
		clonedFiles[name] = append([]byte(nil), body...)
	}
	manifest = cloneManifest(manifest)
	for i := range manifest.Tasks {
		normalized := normalizeTaskSpec(TaskSpec{
			ResourceClass: manifest.Tasks[i].ResourceClass,
			Retry:         manifest.Tasks[i].Retry,
		})
		manifest.Tasks[i].ResourceClass = normalized.ResourceClass
		manifest.Tasks[i].Retry = normalized.Retry
	}
	seenTasks := map[TaskKey]struct{}{}
	for _, task := range manifest.Tasks {
		if _, exists := seenTasks[task.TaskKey]; exists {
			return nil, fmt.Errorf("bundle repeats task %s@%s", task.Kind, task.Version)
		}
		seenTasks[task.TaskKey] = struct{}{}
		modulePath, exportName, err := splitEntrypoint(task.Entrypoint)
		if err != nil {
			return nil, fmt.Errorf("task %s@%s: %w", task.Kind, task.Version, err)
		}
		if _, ok := clonedFiles[modulePath]; !ok {
			return nil, fmt.Errorf("task %s@%s entrypoint file %q is missing", task.Kind, task.Version, modulePath)
		}
		if exportName == "" {
			return nil, fmt.Errorf("task %s@%s entrypoint export is required", task.Kind, task.Version)
		}
		probe := TaskSpec{
			Identity: ImplementationIdentity{
				TaskKey: task.TaskKey, BundleDigest: strings.Repeat("x", 1),
				Entrypoint: task.Entrypoint, ABI: manifest.ABI,
			},
			Inputs: task.Inputs, Outputs: task.Outputs, Modules: task.Modules,
			ResourceClass: task.ResourceClass, Retry: task.Retry,
			BudgetMaximum: cloneBudgetClaim(task.BudgetMaximum),
		}
		if err := validateTaskSpec(probe); err != nil {
			return nil, err
		}
	}

	envelope := bundleDigestEnvelope{Manifest: manifest}
	for name, body := range clonedFiles {
		sum := sha256.Sum256(body)
		envelope.Files = append(envelope.Files, bundleDigestFile{
			Path: name, Digest: "sha256:" + hex.EncodeToString(sum[:]), Size: len(body),
		})
	}
	sort.Slice(envelope.Files, func(i, j int) bool { return envelope.Files[i].Path < envelope.Files[j].Path })
	digest, err := Digest(envelope)
	if err != nil {
		return nil, err
	}
	return &Bundle{manifest: manifest, files: clonedFiles, digest: digest}, nil
}

func (b *Bundle) Digest() string {
	if b == nil {
		return ""
	}
	return b.digest
}

func (b *Bundle) Manifest() BundleManifest {
	if b == nil {
		return BundleManifest{}
	}
	return cloneManifest(b.manifest)
}

func (b *Bundle) TaskSpecs() []TaskSpec {
	if b == nil {
		return nil
	}
	ret := make([]TaskSpec, 0, len(b.manifest.Tasks))
	for _, task := range b.manifest.Tasks {
		ret = append(ret, TaskSpec{
			Identity: ImplementationIdentity{
				TaskKey: task.TaskKey, BundleDigest: b.digest,
				Entrypoint: task.Entrypoint, ABI: b.manifest.ABI,
			},
			Inputs: cloneStringMap(task.Inputs), Outputs: cloneStringMap(task.Outputs),
			Modules:       append([]string(nil), task.Modules...),
			ResourceClass: task.ResourceClass, Retry: task.Retry,
			BudgetMaximum: cloneBudgetClaim(task.BudgetMaximum),
		})
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].Identity.Kind == ret[j].Identity.Kind {
			return ret[i].Identity.Version < ret[j].Identity.Version
		}
		return ret[i].Identity.Kind < ret[j].Identity.Kind
	})
	return ret
}

func (b *Bundle) File(name string) ([]byte, bool) {
	if b == nil {
		return nil, false
	}
	body, ok := b.files[name]
	return append([]byte(nil), body...), ok
}

func splitEntrypoint(entrypoint string) (string, string, error) {
	modulePath, exportName, ok := strings.Cut(strings.TrimSpace(entrypoint), "#")
	if !ok || strings.Contains(exportName, "#") {
		return "", "", fmt.Errorf("entrypoint must be path#export")
	}
	modulePath = strings.TrimPrefix(modulePath, "./")
	modulePath = path.Clean(modulePath)
	if modulePath == "." || strings.HasPrefix(modulePath, "../") || path.IsAbs(modulePath) {
		return "", "", fmt.Errorf("entrypoint path is invalid")
	}
	return modulePath, strings.TrimSpace(exportName), nil
}

func cloneManifest(manifest BundleManifest) BundleManifest {
	ret := manifest
	ret.Tasks = make([]BundleTask, len(manifest.Tasks))
	for i, task := range manifest.Tasks {
		ret.Tasks[i] = task
		ret.Tasks[i].Inputs = cloneStringMap(task.Inputs)
		ret.Tasks[i].Outputs = cloneStringMap(task.Outputs)
		ret.Tasks[i].Modules = append([]string(nil), task.Modules...)
		ret.Tasks[i].BudgetMaximum = cloneBudgetClaim(task.BudgetMaximum)
		sort.Strings(ret.Tasks[i].Modules)
	}
	sort.Slice(ret.Tasks, func(i, j int) bool {
		if ret.Tasks[i].Kind == ret.Tasks[j].Kind {
			return ret.Tasks[i].Version < ret.Tasks[j].Version
		}
		return ret.Tasks[i].Kind < ret.Tasks[j].Kind
	})
	return ret
}

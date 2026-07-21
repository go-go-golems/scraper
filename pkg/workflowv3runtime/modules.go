package workflowv3runtime

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	database "github.com/go-go-golems/go-go-goja/modules/database"
	fetchmod "github.com/go-go-golems/go-go-goja/modules/fetch"
	fsmod "github.com/go-go-golems/go-go-goja/modules/fs"
	gggengine "github.com/go-go-golems/go-go-goja/pkg/engine"
)

// TaskModuleContext is the lease-scoped context used to construct one trusted
// module instance. Factories must not retain the temporary workspace path.
type TaskModuleContext struct {
	Request   TaskRequest
	Workspace string
}

// TaskModuleFactory creates one exact alias for one fresh task runtime.
type TaskModuleFactory struct {
	Alias    string
	Validate func() error
	Build    func(TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error)
}

// TaskModuleRegistry is an immutable set of policy-selected module aliases.
type TaskModuleRegistry struct {
	factories map[string]func(TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error)
}

func NewTaskModuleRegistry(factories ...TaskModuleFactory) (*TaskModuleRegistry, error) {
	registry := &TaskModuleRegistry{
		factories: make(map[string]func(TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error), len(factories)),
	}
	for _, factory := range factories {
		alias := strings.TrimSpace(factory.Alias)
		if alias == "" || factory.Build == nil {
			return nil, fmt.Errorf("task module alias and factory are required")
		}
		if factory.Validate != nil {
			if err := factory.Validate(); err != nil {
				return nil, fmt.Errorf("task module %q: %w", alias, err)
			}
		}
		if _, exists := registry.factories[alias]; exists {
			return nil, fmt.Errorf("task module alias %q is already registered", alias)
		}
		registry.factories[alias] = factory.Build
	}
	return registry, nil
}

func (r *TaskModuleRegistry) Aliases() []string {
	if r == nil {
		return nil
	}
	aliases := make([]string, 0, len(r.factories))
	for alias := range r.factories {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	return aliases
}

func (r *TaskModuleRegistry) build(alias string, context TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error) {
	if r == nil {
		return nil, fmt.Errorf("task module registry is required")
	}
	factory, ok := r.factories[alias]
	if !ok {
		return nil, fmt.Errorf("task requests unsupported module %q", alias)
	}
	return factory(context)
}

func FSInputModule() TaskModuleFactory {
	return TaskModuleFactory{
		Alias: "fs:input",
		Build: func(context TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error) {
			inputFS := fsmod.New(
				fsmod.WithName("fs:input"),
				fsmod.WithBackend(fsmod.NewReadOnlyFSBackend(fsmod.FSMount{
					FS: os.DirFS(context.Workspace), Root: ".", Mount: "/",
				})),
			)
			return gggengine.NativeModuleRegistrar{
				ModuleID: "workflowv3:fs-input", ModuleName: "fs:input",
				Loader: inputFS.Loader,
			}, nil
		},
	}
}

// FetchModule creates an exact fetch alias with host-owned origin, timeout,
// response-size, credential, redirect, and transport policy.
func FetchModule(alias string, policy fetchmod.Policy, client *http.Client) TaskModuleFactory {
	return TaskModuleFactory{
		Alias: alias,
		Validate: func() error {
			if len(policy.AllowedOrigins) == 0 {
				return fmt.Errorf("fetch policy requires allowed origins")
			}
			for _, origin := range policy.AllowedOrigins {
				if strings.TrimSpace(origin) == "*" {
					return fmt.Errorf("fetch policy cannot use a wildcard origin")
				}
			}
			if policy.Timeout <= 0 || policy.MaxResponseBytes <= 0 {
				return fmt.Errorf("fetch timeout and response limit must be positive")
			}
			if policy.Credentials.AllowEnv || policy.Credentials.AllowFiles ||
				len(policy.Credentials.AllowedFiles) != 0 {
				return fmt.Errorf("public fetch credential sources must be disabled")
			}
			return nil
		},
		Build: func(TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error) {
			guardedClient := &http.Client{}
			if client != nil {
				clone := *client
				guardedClient = &clone
			}
			transport := guardedClient.Transport
			if transport == nil {
				transport = http.DefaultTransport
			}
			guardedClient.Transport = publicFetchTransport{
				policy: policy, next: transport,
			}
			previousRedirect := guardedClient.CheckRedirect
			guardedClient.CheckRedirect = func(request *http.Request, via []*http.Request) error {
				if _, err := policy.CheckURL(request.URL.String()); err != nil {
					return err
				}
				if previousRedirect != nil {
					return previousRedirect(request, via)
				}
				if len(via) >= 10 {
					return fmt.Errorf("fetch stopped after 10 redirects")
				}
				return nil
			}
			module := fetchmod.New(
				fetchmod.WithName(alias),
				fetchmod.WithPolicy(policy),
				fetchmod.WithHTTPClient(guardedClient),
			)
			return gggengine.NativeModuleRegistrar{
				ModuleID: "workflowv3:" + alias, ModuleName: alias,
				Loader: module.Loader,
			}, nil
		},
	}
}

type publicFetchTransport struct {
	policy fetchmod.Policy
	next   http.RoundTripper
}

func (t publicFetchTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request == nil || request.URL == nil {
		return nil, fmt.Errorf("fetch request URL is required")
	}
	if _, err := t.policy.CheckURL(request.URL.String()); err != nil {
		return nil, err
	}
	if request.URL.User != nil {
		return nil, fmt.Errorf("fetch URL user information is not allowed")
	}
	for _, header := range []string{"Authorization", "Cookie", "Proxy-Authorization"} {
		if request.Header.Get(header) != "" {
			return nil, fmt.Errorf("fetch header %s is not allowed", header)
		}
	}
	return t.next.RoundTrip(request)
}

// DatabaseModule creates an exact alias backed by a Go-preconfigured handle.
// go-go-goja disables configure() when WithPreconfiguredDB is used.
func DatabaseModule(alias string, handle database.QueryExecer) TaskModuleFactory {
	return TaskModuleFactory{
		Alias: alias,
		Validate: func() error {
			if handle == nil {
				return fmt.Errorf("preconfigured database handle is required")
			}
			return nil
		},
		Build: func(TaskModuleContext) (gggengine.RuntimeModuleRegistrar, error) {
			module := database.New(
				database.WithName(alias),
				database.WithPreconfiguredDB(handle),
			)
			return gggengine.NativeModuleRegistrar{
				ModuleID: "workflowv3:" + alias, ModuleName: alias,
				Loader: module.Loader,
			}, nil
		},
	}
}

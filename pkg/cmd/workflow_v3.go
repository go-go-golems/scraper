package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-go-golems/scraper/pkg/researchrunner"
	"github.com/go-go-golems/scraper/pkg/workflowv3"
	"github.com/go-go-golems/scraper/pkg/workflowv3product"
	"github.com/spf13/cobra"
)

type workflowV3Options struct {
	config workflowv3product.Config
}

func newWorkflowV3Command() *cobra.Command {
	options := &workflowV3Options{config: workflowv3product.DefaultConfig()}
	command := &cobra.Command{
		Use:   "workflow",
		Short: "Author, execute, and inspect durable Workflow V3 plans",
	}
	addWorkflowV3Flags(command, options)
	command.AddCommand(newWorkflowValidateCommand(options))
	command.AddCommand(newWorkflowExplainCommand(options))
	command.AddCommand(newWorkflowCompileCommand(options))
	command.AddCommand(newWorkflowResearchctlConfigCommand(options))
	command.AddCommand(newWorkflowSubmitCommand(options, false))
	command.AddCommand(newWorkflowSubmitCommand(options, true))
	command.AddCommand(newWorkflowRunsCommand(options))
	command.AddCommand(newWorkflowServeCommand(options))
	return command
}

func newWorkflowV3WorkerCommand() *cobra.Command {
	options := &workflowV3Options{config: workflowv3product.DefaultConfig()}
	command := &cobra.Command{Use: "worker", Short: "Run Workflow V3 task workers"}
	addWorkflowV3Flags(command, options)
	command.AddCommand(&cobra.Command{
		Use:   "run",
		Short: "Continuously dispatch durable Workflow V3 work",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := workflowv3product.Open(cmd.Context(), options.config)
			if err != nil {
				return err
			}
			defer func() { _ = app.Close() }()
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			return app.RunWorker(ctx)
		},
	})
	return command
}

func newTaskPackagesCommand() *cobra.Command {
	var selected []string
	command := &cobra.Command{Use: "task-packages", Short: "Inspect Workflow V3 task packages"}
	command.PersistentFlags().StringSliceVar(&selected, "task-package", []string{"cookbook-linear"}, "Task package names to load")
	command.AddCommand(&cobra.Command{
		Use: "list", Short: "List selected immutable task packages", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			environment, err := workflowv3product.NewAuthoringEnvironment(selected)
			if err != nil {
				return err
			}
			return writeWorkflowJSON(cmd.OutOrStdout(), environment.Packages.Info())
		},
	})
	return command
}

func addWorkflowV3Flags(command *cobra.Command, options *workflowV3Options) {
	flags := command.PersistentFlags()
	flags.StringVar(&options.config.DatabasePath, "workflow-db", options.config.DatabasePath, "Workflow V3 SQLite database path")
	flags.StringVar(&options.config.ArtifactRoot, "artifact-root", options.config.ArtifactRoot, "Workflow V3 content-addressed artifact root")
	flags.StringSliceVar(&options.config.TaskPackages, "task-package", options.config.TaskPackages, "Task package names to load")
	flags.DurationVar(&options.config.LeaseDuration, "lease-duration", options.config.LeaseDuration, "Workflow attempt lease duration")
	flags.DurationVar(&options.config.PollInterval, "poll-interval", options.config.PollInterval, "Worker and follow polling interval")
	flags.StringToIntVar(&options.config.Capacities, "capacity", options.config.Capacities, "Concurrent task capacity by resource class (for example cpu.default=4)")
	flags.Int64Var(&options.config.MaxArtifactBytes, "max-artifact-bytes", options.config.MaxArtifactBytes, "Maximum artifact size")
}

func newWorkflowValidateCommand(options *workflowV3Options) *cobra.Command {
	return &cobra.Command{
		Use: "validate <workflow.js>", Short: "Validate and compile a pure Workflow V3 JavaScript plan", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			authored, err := authorWorkflowFile(cmd.Context(), options.config.TaskPackages, args[0])
			if err != nil {
				return err
			}
			return writeWorkflowJSON(cmd.OutOrStdout(), map[string]any{
				"ok": true, "name": authored.Name, "planDigest": authored.Digest,
			})
		},
	}
}

func newWorkflowExplainCommand(options *workflowV3Options) *cobra.Command {
	return &cobra.Command{
		Use: "explain <workflow.js>", Short: "Explain exact tasks, policies, inputs, and outputs", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read workflow script: %w", err)
			}
			environment, err := workflowv3product.NewAuthoringEnvironment(options.config.TaskPackages)
			if err != nil {
				return err
			}
			app := &workflowv3product.Application{Authoring: environment}
			explanation, err := app.Explain(cmd.Context(), string(source))
			if err != nil {
				return err
			}
			return writeWorkflowJSON(cmd.OutOrStdout(), explanation)
		},
	}
}

func newWorkflowCompileCommand(options *workflowV3Options) *cobra.Command {
	var output string
	command := &cobra.Command{
		Use: "compile <workflow.js>", Short: "Compile JavaScript into a canonical Workflow V3 plan", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := authorWorkflowFile(cmd.Context(), options.config.TaskPackages, args[0])
			if err != nil {
				return err
			}
			body, err := json.MarshalIndent(plan, "", "  ")
			if err != nil {
				return err
			}
			body = append(body, '\n')
			if output == "" || output == "-" {
				_, err = cmd.OutOrStdout().Write(body)
				return err
			}
			return writeFileAtomic(output, body)
		},
	}
	command.Flags().StringVarP(&output, "out", "o", "-", "Output plan path, or - for stdout")
	return command
}

func newWorkflowResearchctlConfigCommand(options *workflowV3Options) *cobra.Command {
	var bindingsPath, output string
	command := &cobra.Command{
		Use: "researchctl-config <workflow.js>", Short: "Compile a Workflow V3 script into a scraper-workflow-execution/v1 domain config", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if bindingsPath == "" {
				return fmt.Errorf("--bindings is required")
			}
			source, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read workflow script: %w", err)
			}
			environment, err := workflowv3product.NewAuthoringEnvironment(options.config.TaskPackages)
			if err != nil {
				return err
			}
			authored, err := environment.Author(cmd.Context(), string(source))
			if err != nil {
				return err
			}
			body, err := os.ReadFile(bindingsPath)
			if err != nil {
				return fmt.Errorf("read Researchctl input bindings: %w", err)
			}
			decoder := json.NewDecoder(bytes.NewReader(body))
			decoder.DisallowUnknownFields()
			var bindings map[string]researchrunner.InputBinding
			if err := decoder.Decode(&bindings); err != nil {
				return fmt.Errorf("decode Researchctl input bindings: %w", err)
			}
			var trailing any
			if err := decoder.Decode(&trailing); err != io.EOF {
				return fmt.Errorf("decode Researchctl input bindings: trailing JSON content")
			}
			execution, err := researchrunner.BuildExecution(authored.Plan, environment.Packages, bindings, researchrunner.ObservationPolicy{ExportOutputs: true, ExportExternalOperations: true})
			if err != nil {
				return err
			}
			encoded, err := json.MarshalIndent(execution, "", "  ")
			if err != nil {
				return err
			}
			encoded = append(encoded, '\n')
			if output == "" || output == "-" {
				_, err = cmd.OutOrStdout().Write(encoded)
				return err
			}
			return writeFileAtomic(output, encoded)
		},
	}
	command.Flags().StringVar(&bindingsPath, "bindings", "", "Strict JSON map from workflow input names to Researchctl artifact selectors")
	command.Flags().StringVarP(&output, "out", "o", "-", "Output domain-config path, or - for stdout")
	return command
}

func newWorkflowSubmitCommand(options *workflowV3Options, execute bool) *cobra.Command {
	var inputsPath, runID string
	use, short := "submit <workflow.js>", "Submit a durable Workflow V3 run without starting a worker"
	if execute {
		use, short = "run <workflow.js>", "Submit and execute a Workflow V3 run until terminal"
	}
	command := &cobra.Command{
		Use: use, Short: short, Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plan, err := authorWorkflowFile(cmd.Context(), options.config.TaskPackages, args[0])
			if err != nil {
				return err
			}
			inputs := map[string]workflowv3product.StagedInput{}
			baseDir := "."
			if inputsPath != "" {
				inputs, baseDir, err = workflowv3product.DecodeInputs(inputsPath)
				if err != nil {
					return err
				}
			}
			app, err := workflowv3product.Open(cmd.Context(), options.config)
			if err != nil {
				return err
			}
			defer func() { _ = app.Close() }()
			submission, err := app.Submit(cmd.Context(), plan, inputs, baseDir, workflowv3.RunID(runID))
			if err != nil {
				return err
			}
			if !execute {
				return writeWorkflowJSON(cmd.OutOrStdout(), submission)
			}
			view, err := app.RunUntilTerminal(cmd.Context(), submission.RunID)
			if err != nil {
				return err
			}
			if err := writeWorkflowJSON(cmd.OutOrStdout(), view); err != nil {
				return err
			}
			if view.Snapshot.Status != "succeeded" {
				return fmt.Errorf("workflow run %s finished with status %s", submission.RunID, view.Snapshot.Status)
			}
			return nil
		},
	}
	command.Flags().StringVar(&inputsPath, "inputs", "", "JSON staged-input manifest path")
	command.Flags().StringVar(&runID, "run-id", "", "Explicit run ID (default: generated UUID)")
	return command
}

func newWorkflowServeCommand(options *workflowV3Options) *cobra.Command {
	var address, operatorTokenEnv string
	command := &cobra.Command{
		Use: "serve", Short: "Serve the Workflow V3 operator read and cancellation API", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			app, err := workflowv3product.Open(cmd.Context(), options.config)
			if err != nil {
				return err
			}
			defer func() { _ = app.Close() }()
			operatorToken := ""
			if operatorTokenEnv != "" {
				operatorToken = os.Getenv(operatorTokenEnv)
			}
			handler, err := workflowv3product.NewHTTPHandler(app, workflowv3product.HTTPOptions{OperatorToken: operatorToken})
			if err != nil {
				return err
			}
			server := &http.Server{Addr: address, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			done := make(chan error, 1)
			go func() { done <- server.ListenAndServe() }()
			select {
			case err := <-done:
				if err == http.ErrServerClosed {
					return nil
				}
				return err
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := server.Shutdown(shutdownCtx); err != nil {
					return err
				}
				err := <-done
				if err != nil && err != http.ErrServerClosed {
					return err
				}
				return nil
			}
		},
	}
	command.Flags().StringVar(&address, "address", "127.0.0.1:8081", "Workflow V3 operator API listen address")
	command.Flags().StringVar(&operatorTokenEnv, "operator-token-env", "SCRAPER_WORKFLOW_OPERATOR_TOKEN", "Environment variable containing the bearer token for mutating operator requests")
	return command
}

func newWorkflowRunsCommand(options *workflowV3Options) *cobra.Command {
	command := &cobra.Command{Use: "runs", Short: "Inspect and control durable Workflow V3 runs"}
	var status string
	var limit int
	list := &cobra.Command{
		Use: "list", Short: "List recent runs", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return withWorkflowApp(cmd, options, func(app *workflowv3product.Application) error {
				runs, err := app.ListRuns(cmd.Context(), status, limit)
				if err != nil {
					return err
				}
				return writeWorkflowJSON(cmd.OutOrStdout(), runs)
			})
		},
	}
	list.Flags().StringVar(&status, "status", "", "Optional exact run status")
	list.Flags().IntVar(&limit, "limit", 100, "Maximum runs to return (up to 1000)")
	command.AddCommand(list)
	command.AddCommand(&cobra.Command{
		Use: "show <run-id>", Short: "Show a run snapshot and bounded operational evidence", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withWorkflowApp(cmd, options, func(app *workflowv3product.Application) error {
				view, err := app.Show(cmd.Context(), workflowv3.RunID(args[0]))
				if err != nil {
					return err
				}
				return writeWorkflowJSON(cmd.OutOrStdout(), view)
			})
		},
	})
	command.AddCommand(&cobra.Command{
		Use: "cancel <run-id>", Short: "Fence active attempts and cancel a run", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withWorkflowApp(cmd, options, func(app *workflowv3product.Application) error {
				view, err := app.Cancel(cmd.Context(), workflowv3.RunID(args[0]))
				if err != nil {
					return err
				}
				return writeWorkflowJSON(cmd.OutOrStdout(), view)
			})
		},
	})
	command.AddCommand(&cobra.Command{
		Use: "follow <run-id>", Short: "Stream changed run snapshots as NDJSON until terminal", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withWorkflowApp(cmd, options, func(app *workflowv3product.Application) error {
				return followWorkflowRun(cmd.Context(), cmd.OutOrStdout(), app, workflowv3.RunID(args[0]))
			})
		},
	})
	return command
}

func authorWorkflowFile(ctx context.Context, selected []string, path string) (workflowv3.WorkflowPlan, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return workflowv3.WorkflowPlan{}, fmt.Errorf("read workflow script: %w", err)
	}
	environment, err := workflowv3product.NewAuthoringEnvironment(selected)
	if err != nil {
		return workflowv3.WorkflowPlan{}, err
	}
	authored, err := environment.Author(ctx, string(source))
	if err != nil {
		return workflowv3.WorkflowPlan{}, err
	}
	return authored.Plan, nil
}

func withWorkflowApp(cmd *cobra.Command, options *workflowV3Options, run func(*workflowv3product.Application) error) error {
	app, err := workflowv3product.Open(cmd.Context(), options.config)
	if err != nil {
		return err
	}
	defer func() { _ = app.Close() }()
	return run(app)
}

func followWorkflowRun(ctx context.Context, out io.Writer, app *workflowv3product.Application, runID workflowv3.RunID) error {
	encoder := json.NewEncoder(out)
	ticker := time.NewTicker(app.Config.PollInterval)
	defer ticker.Stop()
	last := ""
	for {
		view, err := app.Show(ctx, runID)
		if err != nil {
			return err
		}
		body, err := json.Marshal(view)
		if err != nil {
			return err
		}
		current := string(body)
		if current != last {
			if err := encoder.Encode(view); err != nil {
				return err
			}
			last = current
		}
		if view.Snapshot.Status == "succeeded" || view.Snapshot.Status == "failed" || view.Snapshot.Status == "canceled" {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func writeWorkflowJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeFileAtomic(path string, body []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".workflow-plan-*")
	if err != nil {
		return fmt.Errorf("create temporary plan: %w", err)
	}
	name := temporary.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err := temporary.Write(body); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary plan: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary plan: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary plan: %w", err)
	}
	if err := os.Chmod(name, 0o644); err != nil {
		return fmt.Errorf("chmod temporary plan: %w", err)
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("publish plan: %w", err)
	}
	return nil
}

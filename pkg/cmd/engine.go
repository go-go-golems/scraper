package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	sqlitestore "github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/spf13/cobra"
)

type engineCommandOptions struct {
	engineDB string
}

func newEngineCommand() *cobra.Command {
	options := &engineCommandOptions{}

	engineCmd := &cobra.Command{
		Use:   "engine",
		Short: "Inspect engine database state and migrations",
	}
	engineCmd.PersistentFlags().StringVar(
		&options.engineDB,
		"engine-db",
		"state/engine.db",
		"Path to the engine SQLite database",
	)

	migrationsCmd := &cobra.Command{
		Use:   "migrations",
		Short: "Inspect engine migration status",
	}
	migrationsCmd.AddCommand(newEngineMigrationsStatusCommand(options))

	engineCmd.AddCommand(newEngineStatusCommand(options))
	engineCmd.AddCommand(migrationsCmd)

	return engineCmd
}

func newEngineStatusCommand(options *engineCommandOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show engine DB schema status and runtime counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := sqlitestore.Inspect(cmd.Context(), options.engineDB)
			if err != nil {
				return err
			}

			upToDate := "no"
			if status.MigrationsUpToDate {
				upToDate = "yes"
			}

			initialized := "no"
			if status.Initialized {
				initialized = "yes"
			}

			exists := "no"
			if status.Exists {
				exists = "yes"
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Engine DB: %s\n", status.Path)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exists: %s\n", exists)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Initialized: %s\n", initialized)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Current schema version: %s\n", formatVersion(status.Exists && status.Initialized, status.CurrentVersion))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Latest known migration: %d\n", status.LatestKnownMigration)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Migrations up to date: %s\n", upToDate)

			if !status.Exists || !status.Initialized {
				return nil
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nWorkflows: %d\n", status.WorkflowCount)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Ops:\n")
			for _, opStatus := range orderedOpStatuses() {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %d\n", opStatus, status.OpCounts[opStatus])
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Leases:\n")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  active: %d\n", status.ActiveLeases)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  expired: %d\n", status.ExpiredLeases)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Results: %d\n", status.ResultCount)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Artifacts: %d\n", status.ArtifactCount)

			return nil
		},
	}
}

func newEngineMigrationsStatusCommand(options *engineCommandOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show known engine migrations and whether they are applied",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := sqlitestore.Inspect(cmd.Context(), options.engineDB)
			if err != nil {
				return err
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Engine DB: %s\n", status.Path)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Current schema version: %s\n", formatVersion(status.Exists && status.Initialized, status.CurrentVersion))
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Latest known migration: %d\n", status.LatestKnownMigration)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nMigrations:\n")

			for _, migration := range status.Migrations {
				state := "pending"
				appliedAt := ""
				if migration.Applied {
					state = "applied"
					appliedAt = migration.AppliedAt.Format(timeFormat)
				}
				line := fmt.Sprintf("  [%s] %d %s", state, migration.Version, migration.Name)
				if appliedAt != "" {
					line += " at " + appliedAt
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
			}

			return nil
		},
	}
}

const timeFormat = "2006-01-02 15:04:05Z07:00"

func formatVersion(ok bool, version int) string {
	if !ok {
		return "n/a"
	}
	return fmt.Sprintf("%d", version)
}

func orderedOpStatuses() []model.OpStatus {
	ret := []model.OpStatus{
		model.OpStatusPending,
		model.OpStatusReady,
		model.OpStatusRunning,
		model.OpStatusSucceeded,
		model.OpStatusFailed,
		model.OpStatusCanceled,
	}
	sort.SliceStable(ret, func(i, j int) bool { return strings.Compare(string(ret[i]), string(ret[j])) < 0 })
	return []model.OpStatus{
		model.OpStatusPending,
		model.OpStatusReady,
		model.OpStatusRunning,
		model.OpStatusSucceeded,
		model.OpStatusFailed,
		model.OpStatusCanceled,
	}
}

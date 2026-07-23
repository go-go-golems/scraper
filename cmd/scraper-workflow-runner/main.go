package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-go-golems/scraper/pkg/researchrunner"
)

func main() {
	config := researchrunner.DefaultConfig()
	var taskPackages stringListFlag
	var capacities capacityFlag
	flag.StringVar(&config.StateRoot, "state-root", config.StateRoot, "durable Workflow V3 runner state root")
	flag.StringVar(&config.ArtifactRoot, "artifact-root", config.ArtifactRoot, "Workflow V3 execution artifact root")
	flag.Var(&taskPackages, "task-package", "exact task package to load (repeatable)")
	flag.Var(&capacities, "capacity", "resource capacity as name=count (repeatable)")
	flag.DurationVar(&config.LeaseDuration, "lease-duration", config.LeaseDuration, "Workflow V3 lease duration")
	flag.DurationVar(&config.PollInterval, "poll-interval", config.PollInterval, "Workflow V3 dispatch poll interval")
	flag.DurationVar(&config.CancellationTimeout, "cancellation-timeout", config.CancellationTimeout, "bounded workflow cancellation acknowledgement timeout")
	flag.Int64Var(&config.MaxRequestBytes, "max-request-bytes", config.MaxRequestBytes, "maximum protocol request bytes")
	flag.Int64Var(&config.MaxExportBytes, "max-export-bytes", config.MaxExportBytes, "maximum bytes for each exported artifact")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "runner does not accept positional arguments")
		os.Exit(2)
	}
	if len(taskPackages) > 0 {
		config.TaskPackages = taskPackages
	}
	if len(capacities) > 0 {
		config.Capacities = capacities
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := researchrunner.Run(ctx, os.Stdin, os.Stdout, config); err != nil {
		if ctx.Err() != nil {
			fmt.Fprintln(os.Stderr, "runner canceled")
		} else {
			fmt.Fprintln(os.Stderr, "runner failed")
		}
		os.Exit(1)
	}
}

type stringListFlag []string

func (f *stringListFlag) String() string { return fmt.Sprint([]string(*f)) }
func (f *stringListFlag) Set(value string) error {
	if value == "" {
		return fmt.Errorf("task package is required")
	}
	*f = append(*f, value)
	return nil
}

type capacityFlag map[string]int

func (f *capacityFlag) String() string { return fmt.Sprint(map[string]int(*f)) }
func (f *capacityFlag) Set(value string) error {
	name, rawCount, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("capacity must be name=count")
	}
	count, err := strconv.Atoi(rawCount)
	if err != nil || count < 1 {
		return fmt.Errorf("capacity must be positive")
	}
	if *f == nil {
		*f = capacityFlag{}
	}
	(*f)[name] = count
	return nil
}

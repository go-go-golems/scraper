package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-go-golems/scraper/pkg/workflowv3runtime"
)

func main() {
	var options workflowv3runtime.IsolatedWorkerOptions
	flag.StringVar(&options.BundleRoot, "bundle-root", "", "read-only staged bundle root")
	flag.StringVar(&options.InputRoot, "input-root", "", "read-only staged input artifact root")
	flag.StringVar(&options.OutputRoot, "output-root", "", "writable candidate output artifact root")
	flag.Parse()
	if flag.NArg() != 0 || options.BundleRoot == "" || options.InputRoot == "" || options.OutputRoot == "" {
		fmt.Fprintln(os.Stderr, "invalid isolated worker invocation")
		os.Exit(2)
	}
	if err := workflowv3runtime.ServeIsolatedTask(context.Background(), os.Stdin, os.Stdout, options); err != nil {
		fmt.Fprintln(os.Stderr, "isolated worker failed")
		os.Exit(1)
	}
}

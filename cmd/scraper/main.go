package main

import (
	"os"

	scrapercmd "github.com/go-go-golems/scraper/pkg/cmd"
)

var version = "dev"

func main() {
	rootCmd, err := scrapercmd.NewRootCommand(version)
	if err != nil {
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

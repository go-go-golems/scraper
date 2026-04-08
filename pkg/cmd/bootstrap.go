package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type BootstrapOptions struct {
	SitesManifestDirs []string
}

func ParseBootstrapArgs(args []string) (BootstrapOptions, error) {
	options := BootstrapOptions{}
	flagName := "--" + SitesManifestDirFlag
	flagWithEquals := flagName + "="

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == flagName:
			if i+1 >= len(args) {
				return BootstrapOptions{}, fmt.Errorf("%s requires a value", flagName)
			}
			i++
			options.SitesManifestDirs = append(options.SitesManifestDirs, args[i])
		case strings.HasPrefix(arg, flagWithEquals):
			options.SitesManifestDirs = append(options.SitesManifestDirs, strings.TrimPrefix(arg, flagWithEquals))
		}
	}

	options.SitesManifestDirs = normalizeManifestDirs(options.SitesManifestDirs)
	return options, nil
}

func CollectSitesManifestDirs(appName string, args []string) ([]string, error) {
	options, err := ParseBootstrapArgs(args)
	if err != nil {
		return nil, err
	}

	return collectSitesManifestDirs(appName, options.SitesManifestDirs)
}

func NewRootCommandFromBootstrap(version string, args []string) (*cobra.Command, error) {
	manifestDirs, err := CollectSitesManifestDirs("scraper", args)
	if err != nil {
		return nil, err
	}

	return NewRootCommand(version, manifestDirs...)
}

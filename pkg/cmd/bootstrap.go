package cmd

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type BootstrapOptions struct {
	SitesManifestDirs []string
}

func ParseBootstrapArgs(args []string) (BootstrapOptions, error) {
	options := BootstrapOptions{}
	fs := pflag.NewFlagSet("scraper-bootstrap", pflag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.ParseErrorsWhitelist.UnknownFlags = true
	fs.StringSliceVar(&options.SitesManifestDirs, SitesManifestDirFlag, nil, "Directory containing site manifests (site.yaml per subdirectory)")

	if err := fs.Parse(args); err != nil {
		return BootstrapOptions{}, err
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

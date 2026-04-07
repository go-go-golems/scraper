package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootVersionCommand(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"version"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Equal(t, "test-version\n", stdout.String())
}

func TestRootHelpLoadsEmbeddedOnboardingDocs(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"help", "scraper-new-developer-onboarding"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "New Developer Onboarding")
	require.Contains(t, stdout.String(), "Step 3")
	require.Contains(t, stdout.String(), "js-demo")
}

func TestRootAPIServeHelp(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"api", "serve", "--help"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "--address")
	require.Contains(t, stdout.String(), "--engine-db")
	require.Contains(t, stdout.String(), "--sites-dir")
}

func TestRootWorkerRunHelpIncludesHTTPProxyFlag(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"worker", "run", "--help"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "--http-proxy")
	require.Contains(t, stdout.String(), "--http-timeout")
	require.Contains(t, stdout.String(), "--metrics-address")
	require.Contains(t, stdout.String(), "--metrics-path")
}

func TestRootHelpLoadsEmbeddedHTTPAPIDoc(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"help", "scraper-http-api"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Scraper HTTP API")
	require.Contains(t, stdout.String(), "/api/v1/sites/js-demo/verbs/seed:submit")
	require.Contains(t, stdout.String(), "scraper api serve")
}

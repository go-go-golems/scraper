package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkerRunHelp(t *testing.T) {
	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{"worker", "run", "--help"})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "--worker-id")
	require.Contains(t, stdout.String(), "--max-cycles")
	require.Contains(t, stdout.String(), "--sites-dir")
}

func TestWorkerRunMaxCyclesInitializesEngineDB(t *testing.T) {
	engineDB := filepath.Join(t.TempDir(), "engine.db")
	sitesDir := t.TempDir()

	rootCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var stdout bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stdout)
	rootCmd.SetArgs([]string{
		"worker", "run",
		"--engine-db", engineDB,
		"--sites-dir", sitesDir,
		"--worker-id", "test-worker",
		"--max-cycles", "1",
	})

	err = rootCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "Worker ID: test-worker")
	require.Contains(t, stdout.String(), "Cycles: 1")
	require.Contains(t, stdout.String(), "Processed: 0")

	statusCmd, err := NewRootCommand("test-version")
	require.NoError(t, err)

	var statusOut bytes.Buffer
	statusCmd.SetOut(&statusOut)
	statusCmd.SetErr(&statusOut)
	statusCmd.SetArgs([]string{"engine", "status", "--engine-db", engineDB})

	err = statusCmd.Execute()
	require.NoError(t, err)
	require.Contains(t, statusOut.String(), "Exists: yes")
	require.Contains(t, statusOut.String(), "Initialized: yes")
}

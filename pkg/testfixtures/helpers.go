package testfixtures

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// SitesDir returns the path to the repo's sites/ directory.
// It resolves the path relative to the calling test file's location,
// so tests in any package can use it without CWD tricks.
func SitesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// This file is at pkg/testfixtures/helpers.go.
	// sites/ is at <repo_root>/sites/.
	dir := filepath.Join(filepath.Dir(thisFile), "..", "..", "sites")
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("resolve sites dir: %v", err)
	}
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		t.Fatalf("sites dir not found at %s", abs)
	}
	return abs
}

// ReadFixture reads a test fixture file from sites/<site>/fixtures/<name>.
func ReadFixture(t *testing.T, site string, name string) []byte {
	t.Helper()
	path := filepath.Join(SitesDir(t), site, "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s/%s: %v", site, name, err)
	}
	return data
}

package cmd

import "testing"

func TestRootCommandExposesOnlyWorkflowV3Product(t *testing.T) {
	root, err := NewRootCommand("test")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"workflow", "worker", "task-packages", "version"} {
		if command, _, err := root.Find([]string{name}); err != nil || command == root {
			t.Fatalf("missing canonical command %q: %v", name, err)
		}
	}
	for _, name := range []string{"legacy", "engine", "api", "site"} {
		if command, _, _ := root.Find([]string{name}); command != root {
			t.Fatalf("legacy command %q remains registered", name)
		}
	}
	if root.PersistentFlags().Lookup("sites-manifest-dir") != nil {
		t.Fatal("legacy site bootstrap flag remains registered")
	}
}

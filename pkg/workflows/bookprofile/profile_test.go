package bookprofile

import (
	"path/filepath"
	"testing"
)

func TestReport794ProfileDefaults(t *testing.T) {
	profile := Report794()
	if profile.ID != "report-794" {
		t.Fatalf("unexpected id %q", profile.ID)
	}
	if profile.QA.ExpectedPages != 30 {
		t.Fatalf("expected 30 pages, got %d", profile.QA.ExpectedPages)
	}
	if got := profile.PageTypes.KnownPages[15]; got != PageDiagram {
		t.Fatalf("expected page 15 diagram, got %q", got)
	}
	if !profile.Prompt.FigureMarkerContract {
		t.Fatalf("expected figure marker contract")
	}
}

func TestProfileYAMLRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.profile.yaml")
	want := Report794()
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.QA.ExpectedFigureCount != want.QA.ExpectedFigureCount {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestBuildPatchFromDiscovery(t *testing.T) {
	profile := Report794()
	state := DiscoveryState{Figures: []DiscoveredFigure{{Page: 15, Description: "figure"}}, ObservedPages: []ObservedPage{{Page: 31, InferredType: PageDiagram, Confidence: 0.8}}}
	patch := BuildPatch(profile, state)
	if patch.Proposals.PageTypes.Add[31] != PageDiagram {
		t.Fatalf("expected page 31 diagram proposal: %+v", patch)
	}
	if patch.Proposals.QA.ExpectedFigureCount != 1 {
		t.Fatalf("expected figure count proposal")
	}
}

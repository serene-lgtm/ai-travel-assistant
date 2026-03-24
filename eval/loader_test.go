package eval

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCases(t *testing.T) {
	root := repoRoot()
	cases, err := LoadCases(root)
	if err != nil {
		t.Fatalf("LoadCases() error = %v", err)
	}
	if len(cases) != 36 {
		t.Fatalf("len(cases) = %d, want 36", len(cases))
	}
}

func TestFilterCases(t *testing.T) {
	cases := []PromptEvalCase{{ID: "a"}, {ID: "b"}}
	got := FilterCases(cases, []string{"b"})
	if len(got) != 1 || got[0].ID != "b" {
		t.Fatalf("FilterCases() = %#v, want only b", got)
	}
}

func TestValidateCaseRejectsInvalidCategory(t *testing.T) {
	err := validateCase(PromptEvalCase{
		ID:          "x",
		Category:    "bad",
		Description: "bad",
		Rubric:      []string{"a"},
		Expected:    FixtureExpected{TravelRelated: boolPtr(true)},
		Input:       FixtureInput{Content: "hi"},
	})
	if err == nil {
		t.Fatal("validateCase() error = nil, want invalid category error")
	}
}

func TestWriteAndReadRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	run := &PromptEvalRun{RunID: "run1", TotalCases: 1}
	if err := WriteRun(path, run); err != nil {
		t.Fatalf("WriteRun() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	got, err := ReadRun(path)
	if err != nil {
		t.Fatalf("ReadRun() error = %v", err)
	}
	if got.RunID != "run1" {
		t.Fatalf("RunID = %q, want run1", got.RunID)
	}
}

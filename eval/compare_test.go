package eval

import (
	"path/filepath"
	"testing"
)

func TestCompareRuns(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "base.json")
	candPath := filepath.Join(dir, "cand.json")

	base := &PromptEvalRun{
		RunID: "base",
		Results: []PromptEvalResult{
			{
				CaseID:      "case1",
				Category:    CategoryIntent,
				Description: "intent",
				ManualScore: map[string]any{"intent_correct": "pass"},
			},
			{
				CaseID:      "case2",
				Category:    CategoryInspiration,
				Description: "insp",
				ManualScore: map[string]any{"literary_quality": 4.0},
			},
		},
	}
	cand := &PromptEvalRun{
		RunID: "cand",
		Results: []PromptEvalResult{
			{
				CaseID:      "case1",
				Category:    CategoryIntent,
				Description: "intent",
				ManualScore: map[string]any{"intent_correct": "fail"},
			},
			{
				CaseID:      "case2",
				Category:    CategoryInspiration,
				Description: "insp",
				ManualScore: map[string]any{"literary_quality": 5.0},
			},
		},
	}

	if err := WriteRun(basePath, base); err != nil {
		t.Fatalf("WriteRun(base) error = %v", err)
	}
	if err := WriteRun(candPath, cand); err != nil {
		t.Fatalf("WriteRun(cand) error = %v", err)
	}

	summary, err := CompareRuns(basePath, candPath)
	if err != nil {
		t.Fatalf("CompareRuns() error = %v", err)
	}
	if len(summary.Regressions) != 1 {
		t.Fatalf("len(Regressions) = %d, want 1", len(summary.Regressions))
	}
	if len(summary.Improvements) != 1 {
		t.Fatalf("len(Improvements) = %d, want 1", len(summary.Improvements))
	}
}

func boolPtr(v bool) *bool {
	return &v
}

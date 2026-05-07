package eval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptEvalRun(t *testing.T) {
	if os.Getenv("PROMPT_EVAL") != "1" {
		t.Skip("set PROMPT_EVAL=1 to run live prompt eval")
	}

	root := repoRoot()
	cases, err := LoadCases(root)
	if err != nil {
		t.Fatalf("LoadCases() error = %v", err)
	}

	selectedCases := splitCSV(os.Getenv("PROMPT_EVAL_CASES"))
	cases = FilterCases(cases, selectedCases)

	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("NewRunner() error = %v", err)
	}
	runner.SetRAGEnabled(os.Getenv("PROMPT_EVAL_RAG") == "1")
	runner.SetProgressLogger(func(index, total int, item PromptEvalCase) {
		t.Logf("running case %d/%d: %s (%s)", index, total, item.ID, item.Category)
	})

	run, err := runner.Run(context.Background(), cases, os.Getenv("PROMPT_EVAL_LABEL"), selectedCases)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	EvaluateRun(run, cases, func(index, total int, item PromptEvalCase) {
		t.Logf("checking case %d/%d: %s (%s)", index, total, item.ID, item.Category)
	})

	outputPath := os.Getenv("PROMPT_EVAL_OUTPUT")
	if strings.TrimSpace(outputPath) == "" {
		outputPath = DefaultOutputPath(root, run.RunID)
	}
	if err := WriteRun(outputPath, run); err != nil {
		t.Fatalf("WriteRun() error = %v", err)
	}

	t.Logf("prompt eval results written to %s", outputPath)
	passedCount := 0
	for _, result := range run.Results {
		if result.Passed {
			passedCount++
		}
		t.Logf("case=%s category=%s passed=%t", result.CaseID, result.Category, result.Passed)
		for _, assertion := range result.Assertions {
			if !assertion.Passed {
				t.Logf("case=%s assertion=%s failed: %s", result.CaseID, assertion.Name, assertion.Details)
			}
		}
		if result.Error != "" {
			t.Logf("case=%s category=%s error=%s", result.CaseID, result.Category, result.Error)
		}
	}
	failedCount := len(run.Results) - passedCount
	passRate := 0.0
	if len(run.Results) > 0 {
		passRate = float64(passedCount) * 100 / float64(len(run.Results))
	}
	t.Logf("summary total=%d passed=%d failed=%d pass_rate=%.1f%%",
		len(run.Results), passedCount, failedCount, passRate)
}

func TestPromptEvalCompare(t *testing.T) {
	if os.Getenv("PROMPT_EVAL_COMPARE") != "1" {
		t.Skip("set PROMPT_EVAL_COMPARE=1 to compare prompt eval runs")
	}

	baseline := strings.TrimSpace(os.Getenv("PROMPT_EVAL_BASELINE"))
	candidate := strings.TrimSpace(os.Getenv("PROMPT_EVAL_CANDIDATE"))
	if baseline == "" || candidate == "" {
		t.Fatal("PROMPT_EVAL_BASELINE and PROMPT_EVAL_CANDIDATE are required")
	}

	summary, err := CompareRuns(filepath.Clean(baseline), filepath.Clean(candidate))
	if err != nil {
		t.Fatalf("CompareRuns() error = %v", err)
	}

	t.Logf("compared baseline=%s candidate=%s", summary.BaselinePath, summary.CandidatePath)
	t.Logf("total_cases=%d scored_cases=%d regressions=%d improvements=%d unscored=%d",
		summary.TotalCases, summary.ScoredCases, len(summary.Regressions), len(summary.Improvements), len(summary.Unscored))
	for _, item := range summary.Regressions {
		t.Logf("regression case=%s reason=%s", item.CaseID, item.Reason)
	}
	for _, item := range summary.Improvements {
		t.Logf("improvement case=%s reason=%s", item.CaseID, item.Reason)
	}
	if len(summary.Unscored) > 0 {
		t.Logf("unscored=%s", strings.Join(summary.Unscored, ","))
	}
}

func TestPromptEvalInspirationAB(t *testing.T) {
	if os.Getenv("PROMPT_EVAL_AB") != "1" {
		t.Skip("set PROMPT_EVAL_AB=1 to run inspiration A/B prompt eval")
	}

	root := repoRoot()
	cases, err := LoadCases(root)
	if err != nil {
		t.Fatalf("LoadCases() error = %v", err)
	}

	selectedCases := splitCSV(os.Getenv("PROMPT_EVAL_CASES"))
	cases = FilterCases(cases, selectedCases)
	cases = filterInspirationCases(cases)
	if len(cases) == 0 {
		t.Fatal("no inspiration cases selected")
	}

	run, err := RunInspirationAB(context.Background(), root, cases, os.Getenv("PROMPT_EVAL_LABEL"), selectedCases)
	if err != nil {
		t.Fatalf("RunInspirationAB() error = %v", err)
	}

	outputPath := os.Getenv("PROMPT_EVAL_OUTPUT")
	if strings.TrimSpace(outputPath) == "" {
		outputPath = filepath.Join(root, "eval", "artifacts", "prompt_eval", "inspiration_ab.json")
	}
	if err := WriteInspirationABRun(outputPath, run); err != nil {
		t.Fatalf("WriteInspirationABRun() error = %v", err)
	}

	t.Logf("inspiration A/B results written to %s", outputPath)
	for _, item := range run.Items {
		if item.Query != "" {
			t.Logf("query=%s", item.Query)
		}
		if item.BaselineError != "" {
			t.Logf("baseline_error=%s", item.BaselineError)
		} else {
			t.Logf("baseline=%s", item.BaselineOutput)
		}
		if item.RAGError != "" {
			t.Logf("rag_error=%s", item.RAGError)
		} else {
			t.Logf("rag=%s", item.RAGOutput)
		}
	}
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func filterInspirationCases(cases []PromptEvalCase) []PromptEvalCase {
	out := make([]PromptEvalCase, 0, len(cases))
	for _, item := range cases {
		if item.Category == CategoryInspiration {
			out = append(out, item)
		}
	}
	return out
}

func Example_compareEnv() {
	fmt.Println("PROMPT_EVAL=1 go test ./eval -run TestPromptEvalRun -v")
}

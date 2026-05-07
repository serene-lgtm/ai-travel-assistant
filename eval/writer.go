package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func DefaultOutputPath(root, runID string) string {
	return filepath.Join(root, "artifacts", "prompt_eval", runID+".json")
}

func WriteRun(path string, run *PromptEvalRun) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir output dir: %w", err)
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write run file: %w", err)
	}
	return nil
}

func WriteInspirationABRun(path string, run *InspirationABRun) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir output dir: %w", err)
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ab run: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write ab run file: %w", err)
	}
	return nil
}

func ReadRun(path string) (*PromptEvalRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read run file: %w", err)
	}
	var run PromptEvalRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parse run file: %w", err)
	}
	return &run, nil
}

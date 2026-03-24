package eval

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ai-reading-assistant/internal/agent"
	"ai-reading-assistant/internal/config"
	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
)

type Runner struct {
	client             *llm.DeepseekClient
	intentAgent        agent.IntentAgent
	clarificationAgent agent.ClarificationAgent
	inspirationAgent   agent.InspirationAgent
	progress           func(index, total int, item PromptEvalCase)
}

func NewRunner(root string) (*Runner, error) {
	cfg, err := config.LoadConfig(filepath.Join(root, "config.json"))
	if err != nil {
		return nil, err
	}
	client, err := llm.NewDeepseekClient(cfg.Deepseek.Model, cfg.Deepseek.BaseURL, cfg.Deepseek.APIKey)
	if err != nil {
		return nil, err
	}
	return &Runner{
		client:             client,
		intentAgent:        agent.NewIntentAgent(client),
		clarificationAgent: agent.NewClarificationAgent(client),
		inspirationAgent:   agent.NewInspirationAgent(client),
	}, nil
}

func (r *Runner) SetProgressLogger(fn func(index, total int, item PromptEvalCase)) {
	r.progress = fn
}

func (r *Runner) Run(ctx context.Context, cases []PromptEvalCase, label string, selected []string) (*PromptEvalRun, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("no prompt eval cases selected")
	}

	now := time.Now().UTC()
	run := &PromptEvalRun{
		RunID:         buildRunID(now, r.client.Model),
		Label:         strings.TrimSpace(label),
		Timestamp:     now,
		Model:         r.client.Model,
		BaseURL:       r.client.BaseURL,
		GitCommit:     gitCommit(),
		SelectedCases: selected,
		TotalCases:    len(cases),
		Results:       make([]PromptEvalResult, 0, len(cases)),
	}

	for i, item := range cases {
		if r.progress != nil {
			r.progress(i+1, len(cases), item)
		}
		result := PromptEvalResult{
			CaseID:      item.ID,
			Category:    item.Category,
			Description: item.Description,
			Tags:        item.Tags,
			Expected:    item.Expected,
		}

		actual, err := r.runCase(ctx, item)
		if err != nil {
			result.Error = err.Error()
		} else {
			result.Actual = actual
		}
		run.Results = append(run.Results, result)
	}

	return run, nil
}

func (r *Runner) runCase(ctx context.Context, item PromptEvalCase) (any, error) {
	switch item.Category {
	case CategoryIntent:
		result, err := r.intentAgent.DetectTravelIntent(ctx, item.Input.Content)
		if err != nil {
			return nil, err
		}
		return IntentActual{
			TravelRelated: result.TravelRelated,
			RawResponse:   strings.TrimSpace(result.RawResponse),
		}, nil
	case CategoryClarification:
		session, field, err := buildSession(item)
		if err != nil {
			return nil, err
		}
		result, err := r.clarificationAgent.GenerateQuestion(ctx, field, session)
		if err != nil {
			return nil, err
		}
		options := make([]string, 0, len(result.Options))
		for _, option := range result.Options {
			options = append(options, option.Content)
		}
		return ClarificationActual{
			Question:    strings.TrimSpace(result.Question),
			Options:     options,
			TargetField: string(result.TargetField),
		}, nil
	case CategoryInspiration:
		session, _, err := buildSession(item)
		if err != nil {
			return nil, err
		}
		content, err := r.inspirationAgent.Generate(ctx, session)
		if err != nil {
			return nil, err
		}
		return InspirationActual{Content: strings.TrimSpace(content)}, nil
	default:
		return nil, fmt.Errorf("unsupported category %q", item.Category)
	}
}

func buildSession(item PromptEvalCase) (*model.InspirationSession, model.RequirementField, error) {
	if item.SessionState == nil {
		return nil, "", fmt.Errorf("session_state is required")
	}
	current := model.Inspiration{
		ID: "1",
		Mood: model.RequirementItem{
			Content: item.SessionState.CurrentRequest.Mood.Content,
			Score:   item.SessionState.CurrentRequest.Mood.Score,
		},
		Scene: model.RequirementItem{
			Content: item.SessionState.CurrentRequest.Scene.Content,
			Score:   item.SessionState.CurrentRequest.Scene.Score,
		},
		Focus: model.RequirementItem{
			Content: item.SessionState.CurrentRequest.Focus.Content,
			Score:   item.SessionState.CurrentRequest.Focus.Score,
		},
		SceneCandidates: append([]string(nil), item.SessionState.CurrentRequest.SceneCandidates...),
	}
	session := &model.InspirationSession{
		Status:       model.SessionStatus(item.SessionState.Status),
		Inspirations: []model.Inspiration{current},
	}
	field := model.RequirementField(item.SessionState.TargetField)
	return session, field, nil
}

func buildRunID(now time.Time, modelName string) string {
	modelName = strings.ReplaceAll(modelName, "/", "_")
	modelName = strings.ReplaceAll(modelName, " ", "_")
	return fmt.Sprintf("%s_%s", now.Format("20060102T150405"), modelName)
}

func gitCommit() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "."
		}
		wd = parent
	}
}

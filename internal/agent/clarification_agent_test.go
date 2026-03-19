package agent

import (
	"testing"

	"ai-reading-assistant/internal/model"
)

func TestClarificationAgentGenerateQuestionUsesSceneCandidates(t *testing.T) {
	agent := &clarificationAgent{
		llmClient: stubChatCaller{response: `{"question":"ignored","options":["a","b"]}`},
	}
	session := &model.InspirationSession{
		Inspirations: []model.Inspiration{
			{
				SceneCandidates: []string{"东京神保町", "北海道小樽"},
			},
		},
	}

	got, err := agent.GenerateQuestion(t.Context(), model.RequirementFieldScene, session)
	if err != nil {
		t.Fatalf("GenerateQuestion() error = %v", err)
	}
	if got.TargetField != model.RequirementFieldScene {
		t.Fatalf("TargetField = %q, want %q", got.TargetField, model.RequirementFieldScene)
	}
	if len(got.Options) != 2 {
		t.Fatalf("len(Options) = %d, want 2", len(got.Options))
	}
	if got.Options[0].Content != "东京神保町" || got.Options[1].Content != "北海道小樽" {
		t.Fatalf("Options = %#v, want scene candidates", got.Options)
	}
}

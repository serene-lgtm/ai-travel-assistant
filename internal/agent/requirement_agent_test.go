package agent

import (
	"testing"

	"ai-reading-assistant/internal/model"
)

func TestRequirementAgentAnalyzeStoresSceneCandidates(t *testing.T) {
	agent := &requirementAnalyzerAgent{
		llmClient: &sequenceChatCaller{
			responses: []string{
				`{"content":"散步、旧书店、写作","score":4}`,
				`{"content":"清冷、孤独","score":4}`,
				`{"content":"","candidates":["东京神保町","北海道小樽","神户旧居留地"],"score":4}`,
			},
		},
	}

	session := &model.InspirationSession{UserID: "u1"}
	msg := &model.InspirationMessage{Content: "想要村上春树式的孤独、清冷、适合散步的感觉"}
	if err := agent.Analyze(t.Context(), msg, session, ""); err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	current, ok := session.CurrentRequirement()
	if !ok {
		t.Fatal("CurrentRequirement() = false, want true")
	}
	if current.Scene.Content != "" {
		t.Fatalf("Scene.Content = %q, want empty", current.Scene.Content)
	}
	if current.Scene.Score != 2 {
		t.Fatalf("Scene.Score = %d, want 2", current.Scene.Score)
	}
	if len(current.SceneCandidates) != 3 {
		t.Fatalf("SceneCandidates = %#v, want 3 candidates", current.SceneCandidates)
	}
}

func TestRequirementAgentApplyUserChoiceResolvesSceneCandidate(t *testing.T) {
	agent := &requirementAnalyzerAgent{}
	current := &model.Inspiration{
		Scene:           model.RequirementItem{Score: 2},
		SceneCandidates: []string{"东京神保町", "北海道小樽"},
	}
	msg := &model.InspirationMessage{
		Options: []model.Option{{Content: "北海道小樽"}},
	}

	if err := agent.applyUserChoice(current, model.RequirementFieldScene, msg); err != nil {
		t.Fatalf("applyUserChoice() error = %v", err)
	}
	if current.Scene.Content != "北海道小樽" {
		t.Fatalf("Scene.Content = %q, want %q", current.Scene.Content, "北海道小樽")
	}
	if current.Scene.Score < requirementSatisfiedScore {
		t.Fatalf("Scene.Score = %d, want >= %d", current.Scene.Score, requirementSatisfiedScore)
	}
	if len(current.SceneCandidates) != 0 {
		t.Fatalf("SceneCandidates = %#v, want empty", current.SceneCandidates)
	}
}

type sequenceChatCaller struct {
	responses []string
	index     int
}

func (s *sequenceChatCaller) Call(_ string) (string, error) {
	if s.index >= len(s.responses) {
		return `{"content":"","score":0}`, nil
	}
	resp := s.responses[s.index]
	s.index++
	return resp, nil
}

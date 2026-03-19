package agent

import (
	"strings"
	"testing"

	"ai-reading-assistant/internal/model"
)

func TestInspirationAgentGenerateStripsMarkdown(t *testing.T) {
	agent := &inspirationAgent{
		llmClient: stubChatCaller{
			response: "# 秋日京都\n\n**概括**：她会在清晨走过哲学之道。\n- 从银阁寺附近出发，慢慢散步。\n",
		},
	}

	session := &model.InspirationSession{
		Inspirations: []model.Inspiration{
			{
				Mood:  model.RequirementItem{Content: "克制而温柔"},
				Scene: model.RequirementItem{Content: "京都秋天的清晨"},
				Focus: model.RequirementItem{Content: "散步与观察"},
			},
		},
	}

	got, err := agent.Generate(t.Context(), session)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	for _, marker := range []string{"# ", "**", "- "} {
		if strings.Contains(got, marker) {
			t.Fatalf("expected plain text output, got %q", got)
		}
	}
	if !strings.Contains(got, "秋日京都") {
		t.Fatalf("expected title to remain, got %q", got)
	}
}

func TestGenInspirationPromptRequiresPlainText(t *testing.T) {
	if !strings.Contains(genInspirationPrompt, "不要使用 markdown") {
		t.Fatalf("expected prompt to forbid markdown, got %q", genInspirationPrompt)
	}
	if !strings.Contains(genInspirationPrompt, "只围绕一个核心地点") {
		t.Fatalf("expected prompt to enforce a single location, got %q", genInspirationPrompt)
	}
	if !strings.Contains(genInspirationPrompt, "哪位作家或作品与此地有关") {
		t.Fatalf("expected prompt to require concrete literary traces, got %q", genInspirationPrompt)
	}
}

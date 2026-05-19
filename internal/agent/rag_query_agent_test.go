package agent

import (
	"reflect"
	"testing"

	"ai-reading-assistant/internal/model"
)

func TestRAGQueryAgentExtractQueries(t *testing.T) {
	agent := &ragQueryAgent{
		llmClient: stubChatCaller{
			response: `这里是分析：
{"queries":["绍兴鲁迅故里","鲁迅","近代文学","鲁迅"]}`,
		},
	}

	current := &model.Inspiration{
		Scene: model.RequirementItem{Content: "想去绍兴鲁迅故里附近走走"},
		Focus: model.RequirementItem{Content: "鲁迅、近代文学"},
		Mood:  model.RequirementItem{Content: "冷静一点"},
	}

	got, err := agent.ExtractQueries(t.Context(), current)
	if err != nil {
		t.Fatalf("ExtractQueries() error = %v", err)
	}

	want := []string{"绍兴鲁迅故里", "鲁迅", "近代文学"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ExtractQueries() = %#v, want %#v", got, want)
	}
}

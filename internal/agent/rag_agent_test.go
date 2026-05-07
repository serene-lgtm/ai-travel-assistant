package agent

import (
	"reflect"
	"testing"

	"ai-reading-assistant/internal/model"
)

func TestBuildRAGQueriesPrefersShortTerms(t *testing.T) {
	current := &model.Inspiration{
		Scene: model.RequirementItem{
			Content: "京都哲学之道的秋天",
		},
		Focus: model.RequirementItem{
			Content: "川端康成、散步与写作气息",
		},
		Mood: model.RequirementItem{
			Content: "克制而温柔",
		},
	}

	got := buildRAGQueries(current)
	want := []string{
		"京都哲学之道",
		"川端康成",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRAGQueries() = %#v, want %#v", got, want)
	}
}

func TestBuildRAGQueriesKeepsLocationAndHumanityTerms(t *testing.T) {
	current := &model.Inspiration{
		Scene: model.RequirementItem{
			Content: "想去绍兴鲁迅故里附近的旧街巷",
		},
		Focus: model.RequirementItem{
			Content: "鲁迅、旧书店与近代文学",
		},
		Mood: model.RequirementItem{
			Content: "安静地散步",
		},
	}

	got := buildRAGQueries(current)
	want := []string{
		"绍兴鲁迅故里",
		"鲁迅",
		"旧书店",
		"近代文学",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRAGQueries() = %#v, want %#v", got, want)
	}
}

func TestIsUsefulRAGQuery(t *testing.T) {
	tests := []struct {
		term string
		want bool
	}{
		{term: "哲学之道", want: true},
		{term: "鲁迅故里", want: true},
		{term: "茶马古道", want: true},
		{term: "近代文学", want: true},
		{term: "散步", want: false},
		{term: "秋天", want: false},
		{term: "写作气息", want: false},
		{term: "克制而温柔", want: false},
	}

	for _, tt := range tests {
		if got := isUsefulRAGQuery(tt.term); got != tt.want {
			t.Fatalf("isUsefulRAGQuery(%q) = %v, want %v", tt.term, got, tt.want)
		}
	}
}

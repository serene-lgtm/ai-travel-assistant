package agent

import (
	"context"
	"testing"

	"ai-reading-assistant/internal/model"
)

func TestPostProcessRAGQueriesSplitsCompositeTerms(t *testing.T) {
	got := postProcessRAGQueries([]string{
		"绍兴鲁迅故里",
		"东京神保町附近待几天",
		"找夏目漱石留下的痕迹",
		"旧时光",
	})
	required := []string{"绍兴", "鲁迅", "东京", "神保町", "夏目漱石"}

	seen := make(map[string]struct{}, len(got))
	for _, item := range got {
		seen[item] = struct{}{}
	}
	for _, want := range required {
		if _, ok := seen[want]; !ok {
			t.Fatalf("postProcessRAGQueries() missing %q, got %#v", want, got)
		}
	}
	if _, ok := seen["旧时光"]; ok {
		t.Fatalf("postProcessRAGQueries() should filter abstract term, got %#v", got)
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
		{term: "北海道", want: true},
		{term: "小樽", want: true},
		{term: "散步", want: false},
		{term: "秋天", want: false},
		{term: "咖啡馆", want: false},
		{term: "夜里", want: false},
		{term: "写作气息", want: false},
		{term: "克制而温柔", want: false},
	}

	for _, tt := range tests {
		if got := isUsefulRAGQuery(tt.term); got != tt.want {
			t.Fatalf("isUsefulRAGQuery(%q) = %v, want %v", tt.term, got, tt.want)
		}
	}
}

func TestRAGAgentBuildContextUsesLLMQueriesWithPostProcess(t *testing.T) {
	client := &stubWikipediaKnowledge{
		definitions: map[string]*model.WikiDefinition{
			"绍兴": {Title: "绍兴", Summary: "中国浙江省地级市。", FullURL: "https://zh.wikipedia.org/wiki/%E7%BB%8D%E5%85%B4"},
			"鲁迅": {Title: "鲁迅", Summary: "中国现代文学家。", FullURL: "https://zh.wikipedia.org/wiki/%E9%B2%81%E8%BF%85"},
		},
	}

	rag := &ragAgent{
		knowledge: client,
		queryExtractor: &stubRAGQueryAgent{
			queries: []string{"绍兴鲁迅故里", "鲁迅", "旧时光"},
		},
	}
	session := &model.InspirationSession{
		Inspirations: []model.Inspiration{
			{
				Scene: model.RequirementItem{Content: "想去绍兴鲁迅故里附近"},
				Focus: model.RequirementItem{Content: "鲁迅与近代文学"},
			},
		},
	}

	ctx := context.Background()
	got, err := rag.BuildContext(ctx, session)
	if err != nil {
		t.Fatalf("BuildContext() error = %v", err)
	}
	if got == nil {
		t.Fatal("BuildContext() = nil, want non-nil")
	}

	wantQueries := "绍兴 | 鲁迅"
	if got.Query != wantQueries {
		t.Fatalf("BuildContext().Query = %q, want %q", got.Query, wantQueries)
	}
	if len(got.Documents) != 2 {
		t.Fatalf("len(BuildContext().Documents) = %d, want 2", len(got.Documents))
	}
}

type stubRAGQueryAgent struct {
	queries []string
	err     error
}

func (s *stubRAGQueryAgent) ExtractQueries(_ context.Context, _ *model.Inspiration) ([]string, error) {
	return s.queries, s.err
}

type stubWikipediaKnowledge struct {
	definitions map[string]*model.WikiDefinition
}

func (s *stubWikipediaKnowledge) GetDefinition(_ context.Context, keyword string) (*model.WikiDefinition, error) {
	return s.definitions[keyword], nil
}

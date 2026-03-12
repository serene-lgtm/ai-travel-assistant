package agent

import (
	"context"
	"fmt"
	"testing"

	"ai-reading-assistant/internal/model"
	"ai-reading-assistant/internal/wikipedia"
)

func TestWikipediaAgentEnrichKeywords(t *testing.T) {
	client := &stubWikipediaClient{
		summaries: map[string]*wikipedia.Summary{
			"溶洞": {
				Title:   "溶洞",
				Extract: "石灰岩地区形成的洞穴。",
			},
		},
	}
	client.summaries["溶洞"].ContentURLs.Desktop.Page = "https://zh.wikipedia.org/wiki/%E6%BA%B6%E6%B4%9E"

	agent := &wikipediaAgent{client: client}
	keywords := []model.KeyWord{
		{Content: "溶洞"},
		{Content: "溶洞"},
		{Content: "  "},
	}

	got := agent.EnrichKeywords(t.Context(), keywords)
	if len(got) != 3 {
		t.Fatalf("unexpected keyword count: %d", len(got))
	}
	if got[0].WikiDefinition == nil || got[0].WikiDefinition.Title != "溶洞" {
		t.Fatalf("expected first keyword to be enriched: %+v", got[0])
	}
	if got[1].WikiDefinition == nil || got[1].WikiDefinition.FullURL == "" {
		t.Fatalf("expected duplicate keyword to reuse cached wiki definition: %+v", got[1])
	}
	if got[2].WikiDefinition != nil {
		t.Fatalf("expected blank keyword to skip enrichment")
	}
	if client.calls != 1 {
		t.Fatalf("expected one wikipedia lookup, got %d", client.calls)
	}
}

type stubWikipediaClient struct {
	summaries map[string]*wikipedia.Summary
	calls     int
}

func (s *stubWikipediaClient) GetSummary(_ context.Context, keyword string) (*wikipedia.Summary, error) {
	s.calls++
	summary, ok := s.summaries[keyword]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return summary, nil
}

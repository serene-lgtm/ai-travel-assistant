package agent

import (
	"context"
	"fmt"
	"strings"

	"ai-reading-assistant/internal/model"
	"ai-reading-assistant/internal/wikipedia"
)

type WikipediaAgent interface {
	GetDefinition(ctx context.Context, keyword string) (*model.WikiDefinition, error)
	EnrichKeywords(ctx context.Context, keywords []model.KeyWord) []model.KeyWord
}

type wikipediaAgent struct {
	client wikipediaSummaryGetter
}

type wikipediaSummaryGetter interface {
	GetSummary(ctx context.Context, keyword string) (*wikipedia.Summary, error)
}

func NewWikipediaAgent(client *wikipedia.Client) WikipediaAgent {
	return &wikipediaAgent{client: client}
}

func (a *wikipediaAgent) GetDefinition(ctx context.Context, keyword string) (*model.WikiDefinition, error) {
	if a.client == nil {
		return nil, fmt.Errorf("wikipedia agent client is not initialized")
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, fmt.Errorf("wikipedia keyword is empty")
	}

	summary, err := a.client.GetSummary(ctx, keyword)
	if err != nil {
		return nil, err
	}

	return &model.WikiDefinition{
		Title:   strings.TrimSpace(summary.Title),
		Summary: strings.TrimSpace(summary.Extract),
		FullURL: strings.TrimSpace(summary.ContentURLs.Desktop.Page),
	}, nil
}

func (a *wikipediaAgent) EnrichKeywords(ctx context.Context, keywords []model.KeyWord) []model.KeyWord {
	if len(keywords) == 0 {
		return make([]model.KeyWord, 0)
	}

	out := make([]model.KeyWord, 0, len(keywords))
	cache := make(map[string]*model.WikiDefinition, len(keywords))

	for _, item := range keywords {
		enriched := item
		key := strings.TrimSpace(item.Content)
		if key == "" {
			out = append(out, enriched)
			continue
		}

		if cached, ok := cache[key]; ok {
			enriched.WikiDefinition = cloneWikiDefinition(cached)
			out = append(out, enriched)
			continue
		}

		definition, err := a.GetDefinition(ctx, key)
		if err == nil && definition != nil {
			cache[key] = definition
			enriched.WikiDefinition = cloneWikiDefinition(definition)
		}

		out = append(out, enriched)
	}

	return out
}

func cloneWikiDefinition(src *model.WikiDefinition) *model.WikiDefinition {
	if src == nil {
		return nil
	}
	dst := *src
	return &dst
}

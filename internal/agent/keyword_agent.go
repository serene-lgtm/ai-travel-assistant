package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
)

type KeywordAgent interface {
	ExtractKeywordsFromOutput(ctx context.Context, output string) ([]model.KeyWord, error)
}

type keywordAgent struct {
	llmClient chatCaller
}

func NewKeywordAgent(client *llm.DeepseekClient) KeywordAgent {
	return &keywordAgent{llmClient: client}
}

func (a *keywordAgent) ExtractKeywordsFromOutput(_ context.Context, output string) ([]model.KeyWord, error) {
	if a.llmClient == nil {
		return nil, fmt.Errorf("keyword agent llm client is not initialized")
	}
	output = strings.TrimSpace(output)
	if output == "" {
		return make([]model.KeyWord, 0), nil
	}

	prompt := fmt.Sprintf(extractKeywordsFromOutputPrompt, output)
	raw, err := a.llmClient.Call(prompt)
	if err != nil {
		return nil, fmt.Errorf("extract keywords: %w", err)
	}

	var payload struct {
		Keywords []string `json:"keywords"`
	}
	if err := decodeFirstJSONObject(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse keywords: %w", err)
	}

	return buildKeywordsFromOutput(output, payload.Keywords), nil
}

func buildKeywordsFromOutput(output string, candidates []string) []model.KeyWord {
	if len(candidates) == 0 {
		return make([]model.KeyWord, 0)
	}

	out := make([]model.KeyWord, 0, minInt(len(candidates), 5))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		keyword := strings.TrimSpace(candidate)
		if keyword == "" {
			continue
		}
		if !isAllowedKeyword(keyword) {
			continue
		}
		if _, ok := seen[keyword]; ok {
			continue
		}

		start, end, ok := locateKeyword(output, keyword)
		if !ok {
			continue
		}

		seen[keyword] = struct{}{}
		out = append(out, model.KeyWord{
			Content: keyword,
			Start:   strconv.Itoa(start),
			End:     strconv.Itoa(end),
		})
		if len(out) == 5 {
			break
		}
	}
	if len(out) == 0 {
		return make([]model.KeyWord, 0)
	}
	return out
}

func isAllowedKeyword(keyword string) bool {
	if utf8.RuneCountInString(keyword) < 2 {
		return false
	}

	// Prefer general concepts over overly specific place chains like "某县某镇某村".
	if utf8.RuneCountInString(keyword) > 6 && countPlaceUnits(keyword) >= 2 {
		return false
	}

	return true
}

func countPlaceUnits(keyword string) int {
	count := 0
	for _, unit := range []string{"县", "区", "镇", "乡", "村", "州", "旗"} {
		count += strings.Count(keyword, unit)
	}
	return count
}

func locateKeyword(text, keyword string) (start int, end int, ok bool) {
	byteIndex := strings.Index(text, keyword)
	if byteIndex < 0 {
		return 0, 0, false
	}
	start = utf8.RuneCountInString(text[:byteIndex])
	end = start + utf8.RuneCountInString(keyword)
	return start, end, true
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const extractKeywordsFromOutputPrompt = `
你是一名旅行内容编辑。请从下面这段旅行灵感文案中提取 0-5 个适合做术语解释或百科释义的关键词。

要求:
1. 只提取原文中真实出现过的词或短语。
2. 关键词总数不超过 5 个，可以是 0 个。
3. 优先提取较为 general、适合做科普释义的概念:
   - 地貌/景观: 梯田、溶洞、峡湾、雪山、石窟
   - 建筑/遗址/聚落: 古村落、书院、寺院、古城
   - 历史文化概念: 朝圣、茶马古道、物哀、壁画
   - 人物/作品/机构: 梵高、海明威、莫高窟
4. 不要优先提取过于具体且 Wikipedia 命中率低的长地名短语,例如“某县某镇某村”“某核心区某村落”。
5. 不要提取重复词。
6. 不要提取过于抽象的情绪词、普通动词、普通形容词，也不要提取单字词。
7. 只返回 JSON: {"keywords":["词1","词2"]}.

原文:
"""%s"""`

package agent

import (
	"context"
	"fmt"
	"strings"

	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
)

type RAGQueryAgent interface {
	ExtractQueries(ctx context.Context, current *model.Inspiration) ([]string, error)
}

type ragQueryAgent struct {
	llmClient chatCaller
}

func NewRAGQueryAgent(client *llm.DeepseekClient) RAGQueryAgent {
	return &ragQueryAgent{llmClient: client}
}

func (a *ragQueryAgent) ExtractQueries(_ context.Context, current *model.Inspiration) ([]string, error) {
	if a == nil || a.llmClient == nil {
		return nil, fmt.Errorf("rag query agent llm client is not initialized")
	}
	if current == nil {
		return nil, fmt.Errorf("current inspiration is nil")
	}

	prompt := fmt.Sprintf(
		extractRAGQueriesPrompt,
		requirementQueryValue(current.Scene),
		requirementQueryValue(current.Focus),
		requirementQueryValue(current.Mood),
	)
	raw, err := a.llmClient.Call(prompt)
	if err != nil {
		return nil, fmt.Errorf("extract rag queries: %w", err)
	}

	var payload struct {
		Queries []string `json:"queries"`
	}
	if err := decodeFirstJSONObject(raw, &payload); err != nil {
		return nil, fmt.Errorf("parse rag queries: %w", err)
	}

	out := make([]string, 0, len(payload.Queries))
	seen := make(map[string]struct{}, len(payload.Queries))
	for _, item := range payload.Queries {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out, nil
}

const extractRAGQueriesPrompt = `
你是一名旅行知识检索助手。请从下面的旅行需求中提取 2-8 个适合用于 Wikipedia 检索的 query。

目标:
1. 优先抽取命中率高、科普性强的词。
2. 尽量使用最小但完整的词语单元，不要长复合词，不要整句。
3. 优先保留以下类型:
   - 明确人名: 鲁迅、夏目漱石、川端康成
   - 明确地名: 绍兴、东京、敦煌、小樽
   - 明确地点/遗址/景观名词: 莫高窟、哲学之道、石窟、壁画、峡湾、雪山
   - 明确历史文化概念: 近代文学、飞天、朝圣、茶马古道

严格要求:
1. 不要输出句子、短语描述或带修饰语的长串。
2. 不要输出“想去”“想看”“留下的痕迹”“在水边停留”这类动作或修饰性表达。
3. 不要输出情绪词、氛围词、季节词，如“安静”“浪漫”“黄昏”“旧时光”。
4. 如果出现复合词，优先拆成更容易命中的独立词。
   - “绍兴鲁迅故里”更倾向输出“绍兴”“鲁迅”
   - “东京神保町旧书街”更倾向输出“东京”“神保町”
5. 只返回 JSON: {"queries":["...", "..."]}，不要附加解释。

[旅行场景]
%s
[核心关注点/行动]
%s
[情感基调]
%s
`

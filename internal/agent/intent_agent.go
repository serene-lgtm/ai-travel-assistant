package agent

import (
	"context"
	"fmt"
	"strings"

	"ai-reading-assistant/internal/llm"
)

type IntentAgent interface {
	DetectTravelIntent(ctx context.Context, content string) (*IntentResult, error)
}

type IntentResult struct {
	TravelRelated bool   `json:"travel_related"`
	RawResponse   string `json:"raw_response,omitempty"`
}

type intentAgent struct {
	llmClient chatCaller
}

type chatCaller interface {
	Call(prompt string) (string, error)
}

func NewIntentAgent(client *llm.DeepseekClient) IntentAgent {
	return &intentAgent{llmClient: client}
}

func (a *intentAgent) DetectTravelIntent(_ context.Context, content string) (*IntentResult, error) {
	if a.llmClient == nil {
		return nil, fmt.Errorf("intent agent llm client is not initialized")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("intent content is empty")
	}

	prompt := fmt.Sprintf(travelRelatedPrompt, content)
	raw, err := a.llmClient.Call(prompt)
	if err != nil {
		return nil, fmt.Errorf("intent agent call llm: %w", err)
	}

	var payload struct {
		TravelRelated bool `json:"travel_related"`
	}
	if err := decodeFirstJSONObject(raw, &payload); err != nil {
		return nil, fmt.Errorf("intent agent parse response: %w", err)
	}

	return &IntentResult{
		TravelRelated: payload.TravelRelated,
		RawResponse:   strings.TrimSpace(raw),
	}, nil
}

const travelRelatedPrompt = `
你是一名审查员,请判断用户的请求是否本质上与“广义旅行、在地体验或人文探索”相关。

相关范畴包括:
1. 旅行规划: 目的地、路线、行程、交通、住宿
2. 体验设计: 文化活动、节庆、工作坊、课程、特殊体验
3. 人文探索: 历史、艺术、文学、哲学、当地生活、社群互动
4. 灵感激发: 旅行遐想、记忆重塑、跨文化想象、虚构场景
5. 身心之旅: 静修、徒步、朝圣、自然疗愈、感官唤醒
6. 旅行相关创作: 游记、旅行故事、摄影、视频脚本创作

排除范畴:
- 纯工具性操作
- 医疗、金融、法律、理工科技术问题
- 与旅行无关的日常事务和闲聊

判断原则:
- 核心目的是否与旅行体验相关
- 即使用词模糊,只要有明显旅行意图,仍判定为相关
- 对边缘案例采用宽松包容原则

只返回 JSON {"travel_related": true/false}.

原文文本:
"""%s"""`

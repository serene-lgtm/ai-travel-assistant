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
	if isExplicitNonTravelLiteraryQuery(content) {
		return &IntentResult{
			TravelRelated: false,
			RawResponse:   `{"travel_related": false}`,
		}, nil
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
你是一名审查员,请判断用户的请求是否本质上与“旅行目的地、在地体验或旅行灵感生成”相关。

相关范畴包括:
1. 旅行规划: 目的地、路线、行程、交通、住宿
2. 体验设计: 文化活动、节庆、工作坊、课程、特殊体验
3. 人文探索: 以亲自去某地体验、行走、观察为目标的历史、艺术、文学、哲学、当地生活、社群互动
4. 灵感激发: 明确指向旅行遐想、目的地方向、在地体验的跨文化想象或虚构场景
5. 身心之旅: 静修、徒步、朝圣、自然疗愈、感官唤醒
6. 旅行相关创作: 游记、旅行故事、摄影、视频脚本创作

排除范畴:
- 纯工具性操作
- 医疗、金融、法律、理工科技术问题
- 与旅行无关的日常事务和闲聊
- 纯文学概念解释,例如“物哀是什么意思”
- 纯作品讨论、人物讨论、书单推荐,例如“推荐几本关于西南边地书写的书”
- 明确否定旅行意图的输入,例如“我不想旅游,我只是想找几本书”

判断原则:
- 核心目的是否与旅行体验相关
- 即使用词模糊,只要有明显旅行意图,仍判定为相关
- 如果只是借文学、艺术或哲学话题做知识问答,而不是想去某地体验,判定为不相关
- 对边缘案例不要宽松外扩,只有当“去某地体验”的意图成立时才判定为相关

只返回 JSON {"travel_related": true/false}.

原文文本:
"""%s"""`

func isExplicitNonTravelLiteraryQuery(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}

	if strings.Contains(content, "不想旅游") {
		return true
	}

	bookQuery := containsAny(content, "找几本", "推荐几本", "书单", "什么书", "哪些书")
	literaryExplain := strings.HasPrefix(content, "请解释") || strings.Contains(content, "是什么意思")
	travelCue := containsAny(content,
		"去", "旅行", "出发", "目的地", "路线", "行程", "散步", "住上", "住几天",
		"地方", "海边", "古城", "港口", "旧城区", "徒步", "在当地",
	)

	if bookQuery && !travelCue {
		return true
	}
	if literaryExplain && containsAny(content, "文学", "诗歌", "小说", "物哀", "作家", "作品") && !travelCue {
		return true
	}

	return false
}

func containsAny(content string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(content, needle) {
			return true
		}
	}
	return false
}

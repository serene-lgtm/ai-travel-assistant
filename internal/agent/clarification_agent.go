package agent

import (
	"context"
	"fmt"
	"strings"

	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
)

type ClarificationAgent interface {
	GenerateQuestion(ctx context.Context, field model.RequirementField, session *model.InspirationSession) (*ClarificationResult, error)
}

type ClarificationResult struct {
	Question    string
	Options     []model.Option
	TargetField model.RequirementField
}

type clarificationAgent struct {
	llmClient chatCaller
}

func NewClarificationAgent(client *llm.DeepseekClient) ClarificationAgent {
	return &clarificationAgent{llmClient: client}
}

func (a *clarificationAgent) GenerateQuestion(_ context.Context, field model.RequirementField, session *model.InspirationSession) (*ClarificationResult, error) {
	label := fieldLabels[field]
	if label == "" {
		return nil, fmt.Errorf("unknown field %s", field)
	}
	if a.llmClient == nil {
		return nil, fmt.Errorf("clarification agent llm client is not initialized")
	}
	if field == model.RequirementFieldScene {
		if result := buildSceneCandidateQuestion(session); result != nil {
			return result, nil
		}
	}

	guidance := fieldGuidance[field]
	summary := summarizeRequirementState(session)

	prompt := fmt.Sprintf(`你是一位文学旅行策划师,需要继续了解用户需求。
已掌握的信息:
%s

目标: 围绕[%s]提出问题,帮助用户澄清: %s
输出要求:
1. 只返回 JSON {"question":"...", "options":["...","..."]}.
2. options 列表包含2-4个不超过20字的候选,具体且互相区分。
3. 问题要温柔、鼓励式,针对用户视角描述。
4. 不允许自由输入,请选择可执行的选项。
5. 如果目标字段是[旅行场景], options 只能写地点或连贯区域,不要混入动作、情绪、节奏、美食或文学主题。
6. 如果目标字段是[核心焦点], options 只能写行动、体验或感官关注点,不要混入地点。
7. 如果目标字段是[情感基调], options 只能写情绪、气质或氛围,不要混入地点和行动。
`, summary, label, guidance)

	raw, err := a.llmClient.Call(prompt)
	if err != nil {
		return nil, fmt.Errorf("clarify question: %w", err)
	}

	var payload struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}
	if err := decodeFirstJSONObject(raw, &payload); err != nil {
		return nil, fmt.Errorf("clarify parse: %w", err)
	}

	question := strings.TrimSpace(payload.Question)
	if question == "" {
		return nil, fmt.Errorf("clarify question empty")
	}

	opts := make([]model.Option, 0, len(payload.Options))
	for _, opt := range payload.Options {
		opt = strings.TrimSpace(opt)
		if opt == "" {
			continue
		}
		opts = append(opts, model.Option{Content: opt})
		if len(opts) == 4 {
			break
		}
	}
	if len(opts) < 2 {
		return nil, fmt.Errorf("clarify options insufficient")
	}

	return &ClarificationResult{
		Question:    question,
		Options:     opts,
		TargetField: field,
	}, nil
}

func buildSceneCandidateQuestion(session *model.InspirationSession) *ClarificationResult {
	if session == nil {
		return nil
	}
	current, ok := session.CurrentRequirement()
	if !ok || !current.HasSceneCandidates() {
		return nil
	}

	opts := make([]model.Option, 0, len(current.SceneCandidates))
	for _, candidate := range current.SceneCandidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		opts = append(opts, model.Option{Content: candidate})
	}
	if len(opts) < 2 {
		return nil
	}

	return &ClarificationResult{
		Question:    "这次旅行更想先落到哪个地方？先选一个方向，我再继续把它写具体。",
		Options:     opts,
		TargetField: model.RequirementFieldScene,
	}
}

var fieldGuidance = map[model.RequirementField]string{
	model.RequirementFieldMood:  "描述希望在旅行中体验到的情绪、哲思或文学意象,例如“温柔但克制的安静”。",
	model.RequirementFieldScene: "明确旅行发生的地点或环境特征,尽量具体到城市、街区或自然地貌。",
	model.RequirementFieldFocus: "说明旅行中的具体行动或感官体验,例如“在清晨徒步”“寻找作家的老宅”等。",
}

func summarizeRequirementState(session *model.InspirationSession) string {
	if session == nil {
		return ""
	}
	current, ok := session.CurrentRequirement()
	if !ok {
		return ""
	}
	var lines []string
	for _, field := range requirementFieldOrder {
		item := current.Get(field)
		content := defaultIfEmpty(item.Content, "暂未提供")
		if strings.TrimSpace(item.SelectedOption) != "" {
			content = fmt.Sprintf("%s\n选择: %s", content, strings.TrimSpace(item.SelectedOption))
		}
		lines = append(lines, fmt.Sprintf("%s: %s (score=%d)", fieldLabels[field], content, item.Score))
	}
	return strings.Join(lines, "\n")
}

func defaultIfEmpty(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"ai-reading-assistant/internal/model"
)

// analyzeRequirement analyzes and saves user input or user choice respectively
// if it is a user input, call llm to extract and score all 3 fields
// if it is a user choice, save it and give it a default quilified score
func (s *inspirationServiceImpl) analyzeRequirement(msg *model.InspirationMessage, session *model.InspirationSession) error {
	if msg == nil {
		return fmt.Errorf("message is nil")
	}
	if s.llmClient == nil {
		return fmt.Errorf("llm client is not initialized")
	}
	if session == nil {
		return fmt.Errorf("session is nil")
	}
	current := session.EnsureCurrentRequirement()
	if current == nil {
		return fmt.Errorf("current requirement is nil")
	}

	targetField := statusClarifyField(session.Status)
	clarifying := targetField != ""

	if clarifying && msg.Kind == model.MessageKindUserChoice && len(msg.Options) > 0 {
		selected := strings.TrimSpace(msg.Options[0].Content)
		if selected == "" {
			return fmt.Errorf("selected option content is empty")
		}
		item := current.Get(targetField)
		item.SelectedOption = selected
		if item.Score < requirementSatisfiedScore {
			item.Score = requirementSatisfiedScore
		}
		current.Set(targetField, item)
		if session.IsReadyToGenerate() {
			session.Status = model.SessionStatusCompleted
		} else if nextField, err := s.pickNextField(session); err == nil {
			session.Status = statusForField(nextField)
		}
		return nil
	}

	inputText := strings.TrimSpace(msg.Content)
	if inputText == "" {
		return fmt.Errorf("content is empty")
	}

	type requirementFlow struct {
		field            model.RequirementField
		extractionPrompt string
		scoringPrompt    string
	}

	flows := []requirementFlow{
		{field: model.RequirementFieldFocus, extractionPrompt: focusExtractionPrompt, scoringPrompt: focusScoringPrompt},
		{field: model.RequirementFieldMood, extractionPrompt: moodExtractionPrompt, scoringPrompt: moodScoringPrompt},
		{field: model.RequirementFieldScene, extractionPrompt: sceneExtractionPrompt, scoringPrompt: sceneScoringPrompt},
	}

	for _, flow := range flows {
		if targetField != "" && flow.field != targetField {
			continue
		}

		value, score, err := s.extractRequirementField(flow.field, flow.extractionPrompt, flow.scoringPrompt, inputText)
		{
			if err != nil {
				return fmt.Errorf("extract %s: %w", flow.field, err)
			}
		}

		item := current.Get(flow.field)
		item.Content = value
		item.Score = score
		current.Set(flow.field, item)
	}

	if targetField != "" && current.Get(targetField).Content == "" {
		return fmt.Errorf("clarify %s produced empty content", targetField)
	}

	return nil
}

func (s *inspirationServiceImpl) extractRequirementField(field model.RequirementField, extractionTemplate, scoringGuide, userInput string) (string, int, error) {
	extractionPrompt := fmt.Sprintf(extractionTemplate, userInput)
	prompt := fmt.Sprintf(`%s

请忽略上文原有的输出格式要求,统一只输出 JSON {"content":"...", "score": <0-5 的整数>}.
说明:
- content 是你依据上述指引为[%s]提炼出的关键信息,若无法提取请设为""。
- score 按以下标准给出0-5之间的整数,0表示信息缺失:
%s
`, extractionPrompt, fieldLabels[field], scoringGuide)

	raw, err := s.llmClient.Call(prompt)
	if err != nil {
		return "", 0, err
	}

	var payload map[string]any
	if err := decodeFirstJSONObject(raw, &payload); err != nil {
		return "", 0, err
	}

	content := ""
	if val, ok := payload["content"].(string); ok {
		content = strings.TrimSpace(val)
	}
	score := normalizeScore(payload["score"])
	return content, score, nil
}

func normalizeScore(raw any) int {
	score := 0
	switch v := raw.(type) {
	case float64:
		score = int(math.Round(v))
	case string:
		v = strings.TrimSpace(v)
		if n, err := strconv.Atoi(v); err == nil {
			score = n
		}
	case json.Number:
		if n, err := v.Int64(); err == nil {
			score = int(n)
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 5 {
		score = 5
	}
	return score
}

func statusForField(field model.RequirementField) model.SessionStatus {
	switch field {
	case model.RequirementFieldMood:
		return model.SessionStatusAskingMood
	case model.RequirementFieldScene:
		return model.SessionStatusAskingScene
	case model.RequirementFieldFocus:
		return model.SessionStatusAskingFocus
	default:
		return model.SessionStatusAskingMood
	}
}

func statusClarifyField(status model.SessionStatus) model.RequirementField {
	switch status {
	case model.SessionStatusAskingMood:
		return model.RequirementFieldMood
	case model.SessionStatusAskingScene:
		return model.RequirementFieldScene
	case model.SessionStatusAskingFocus:
		return model.RequirementFieldFocus
	default:
		return ""
	}
}

const requirementSatisfiedScore = 6

var requirementFieldOrder = []model.RequirementField{
	model.RequirementFieldMood,
	model.RequirementFieldScene,
	model.RequirementFieldFocus,
}

func (s *inspirationServiceImpl) pickNextField(session *model.InspirationSession) (model.RequirementField, error) {
	minScore := math.MaxInt32
	var selected model.RequirementField
	for _, field := range requirementFieldOrder {
		current, ok := session.CurrentRequirement()
		if !ok {
			return "", fmt.Errorf("current requirement is empty")
		}
		score := scoreForField(current, field)
		if score >= requirementSatisfiedScore {
			continue
		}
		if score < minScore {
			minScore = score
			selected = field
		}
	}
	if selected == "" {
		return "", fmt.Errorf("no requirement field needs clarification")
	}
	return selected, nil
}

func scoreForField(req *model.Inspiration, field model.RequirementField) int {
	if req == nil {
		return 0
	}
	return req.Get(field).Score
}

package orchestrator

import (
	"fmt"

	"ai-reading-assistant/internal/model"
)

// to determine which requirement field to clarify next, we pick the one with the lowest score among mood, scene and focus
var requirementFieldOrder = []model.RequirementField{
	model.RequirementFieldMood,
	model.RequirementFieldScene,
	model.RequirementFieldFocus,
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

func pickNextField(session *model.InspirationSession) (model.RequirementField, error) {
	current, ok := session.CurrentRequirement()
	if !ok {
		return "", fmt.Errorf("current requirement is empty")
	}

	if current.HasSceneCandidates() {
		return model.RequirementFieldScene, nil
	}
	if shouldPrioritizeFocus(current) {
		return model.RequirementFieldFocus, nil
	}

	minScore := int(^uint(0) >> 1)
	var selected model.RequirementField
	for _, field := range requirementFieldOrder {
		score := current.Get(field).Score
		if score >= 3 {
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

func advanceSessionStatus(session *model.InspirationSession) {
	if session == nil {
		return
	}
	if session.IsReadyToGenerate() {
		return
	}
	if nextField, err := pickNextField(session); err == nil {
		session.Status = statusForField(nextField)
	}
}

func shouldPrioritizeFocus(current *model.Inspiration) bool {
	if current == nil {
		return false
	}
	return current.Focus.Score >= 3 && current.Scene.Score < 3 && current.Mood.Score < 3
}

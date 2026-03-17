package agent

import "ai-reading-assistant/internal/model"

const requirementSatisfiedScore = 3

var requirementFieldOrder = []model.RequirementField{
	model.RequirementFieldMood,
	model.RequirementFieldScene,
	model.RequirementFieldFocus,
}

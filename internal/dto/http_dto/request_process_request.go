package http_dto

import (
	"fmt"
	"strings"
	"time"

	"ai-reading-assistant/internal/model"
)

// RequestProcessRequest captures the REST payload for creating/updating a request process.
type RequestProcessRequest struct {
	Stage string `json:"stage" binding:"required"`
}

// ToModel converts the request into a domain model ready for persistence.
func (r RequestProcessRequest) ToModel() (*model.RequestProcess, error) {
	stage := model.RequestStage(r.Stage)
	if stage != model.RequestStageDetectUserIntent &&
		stage != model.RequestStageAnalyzeRequirement &&
		stage != model.RequestStageGenerateOptions &&
		stage != model.RequestStageGenerateInspiration &&
		stage != model.RequestStageEnrichKeywords {
		return nil, fmt.Errorf("invalid stage %q", r.Stage)
	}

	if strings.TrimSpace(r.Stage) == "" {
		return nil, fmt.Errorf("stage is required")
	}

	now := time.Now().UTC()
	return &model.RequestProcess{
		Stage:     stage,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

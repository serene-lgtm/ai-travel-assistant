package model

import "time"

type RequestStage string

const (
	RequestStageDecoding   RequestStage = "decoding"
	RequestStageAnalyzing  RequestStage = "analyzing"
	RequestStageGenerating RequestStage = "generating"
	RequestStageCompleted  RequestStage = "completed"
	RequestStageFailed     RequestStage = "failed"
)

// RequestProcess tracks the progress of a ChatCompletion request
// from initial input decoding through requirement analysis to inspiration generation
type RequestProcess struct {
	ID          string       `json:"id" bson:"_id,omitempty"`
	SessionID   string       `json:"session_id" bson:"sid"`
	UserID      string       `json:"user_id" bson:"uid"`
	Stage       RequestStage `json:"stage" bson:"stg"`
	StartedAt   time.Time    `json:"started_at" bson:"sat"`
	CompletedAt time.Time    `json:"completed_at" bson:"cat"`
	CreatedAt   time.Time    `json:"created_at" bson:"cat_at"`
	UpdatedAt   time.Time    `json:"updated_at" bson:"uat"`
	Error       string       `json:"error,omitempty" bson:"err,omitempty"`
}

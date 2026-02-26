package model

import "time"

type RequestStage string

const (
	RequestStageDecoding   RequestStage = "decoding"
	RequestStageAnalyzing  RequestStage = "analyzing"
	RequestStageGenerating RequestStage = "generating"
)

type RequestProcess struct {
	ID        string       `json:"id" bson:"_id,omitempty"`
	Stage     RequestStage `json:"stage" bson:"stg"`
	CreatedAt time.Time    `json:"created_at" bson:"cat"`
	UpdatedAt time.Time    `json:"updated_at" bson:"uat"`
}

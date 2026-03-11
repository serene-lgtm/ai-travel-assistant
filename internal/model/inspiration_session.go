package model

import (
	"strconv"
	"time"
)

type SessionStatus string

type RequirementField string

const (
	SessionStatusCreated     SessionStatus    = "created"
	SessionStatusAskingMood  SessionStatus    = "askingMood"
	SessionStatusAskingScene SessionStatus    = "askingScene"
	SessionStatusAskingFocus SessionStatus    = "askingFocus"
	SessionStatusStartOver   SessionStatus    = "startOver"
	SessionStatusCompleted   SessionStatus    = "completed"
	RequirementFieldMood     RequirementField = "mood"
	RequirementFieldScene    RequirementField = "scene"
	RequirementFieldFocus    RequirementField = "focus"
)

type InspirationSession struct {
	ID           string        `json:"id" bson:"_id,omitempty"`
	UserID       string        `json:"user_id" bson:"uid"`
	MaxToken     int           `json:"max_token" bson:"mt"`
	Messages     []string      `json:"messages" bson:"msgs"` // ordered message IDs
	Inspirations []Inspiration `json:"inspirations" bson:"insps"`
	Status       SessionStatus `json:"status" bson:"st"`
	CreatedAt    time.Time     `json:"created_at" bson:"cat"`
}

// Inspiration captures one full inspiration requirement set with all structured fields.
type Inspiration struct {
	ID         string          `json:"id" bson:"id"`
	UserID     string          `json:"user_id" bson:"uid"`
	Mood       RequirementItem `json:"mood" bson:"mood"`
	Scene      RequirementItem `json:"scene" bson:"scene"`
	Focus      RequirementItem `json:"focus" bson:"focus"`
	Output     string          `json:"output" bson:"op"`
	KeyWords   []KeyWord       `json:"keywords" bson:"kws"`
	IsFavorite bool            `json:"is_favorite" bson:"if"`
}

type KeyWord struct {
	Content        string          `json:"content" bson:"cnt"`
	Start          string          `json:"start" bson:"st"` // start and end refer to the position of the keyword in the text, used for highlighting in UI
	End            string          `json:"end" bson:"ed"`
	WikiDefinition *WikiDefinition `json:"wiki_definition,omitempty" bson:"wiki_def,omitempty"`
}

type WikiDefinition struct {
	Title   string `json:"title" bson:"title"`       // 关键词（如"溶洞"）
	Summary string `json:"summary" bson:"summary"`   // 简短释义
	FullURL string `json:"full_url" bson:"full_url"` // 完整页面跳转链接
}

// RequirementItem captures both the extracted description and its score.
type RequirementItem struct {
	Content        string `json:"content" bson:"cnt"`
	Score          int    `json:"score" bson:"score"`
	SelectedOption string `json:"selected_option,omitempty" bson:"sel,omitempty"`
}

func (s *InspirationSession) EnsureRequirementInitialized() {
	if s == nil || s.Inspirations != nil {
		return
	}
	s.Inspirations = make([]Inspiration, 0)
}

func (s *InspirationSession) CurrentRequirement() (*Inspiration, bool) {
	if s == nil {
		return nil, false
	}
	if len(s.Inspirations) == 0 {
		return nil, false
	}
	return &s.Inspirations[len(s.Inspirations)-1], true
}

func (s *InspirationSession) EnsureCurrentRequirement() *Inspiration {
	if s == nil {
		return nil
	}
	s.EnsureRequirementInitialized()
	if current, ok := s.CurrentRequirement(); ok {
		return current
	}
	s.Inspirations = append(s.Inspirations, Inspiration{
		ID:     strconv.Itoa(len(s.Inspirations) + 1),
		UserID: s.UserID,
	})
	return &s.Inspirations[len(s.Inspirations)-1]
}

func (s *InspirationSession) AppendRequirement() *Inspiration {
	if s == nil {
		return nil
	}
	s.EnsureRequirementInitialized()
	s.Inspirations = append(s.Inspirations, Inspiration{
		ID:     strconv.Itoa(len(s.Inspirations) + 1),
		UserID: s.UserID,
	})
	return &s.Inspirations[len(s.Inspirations)-1]
}

func (r *Inspiration) Get(field RequirementField) RequirementItem {
	if r == nil {
		return RequirementItem{}
	}
	switch field {
	case RequirementFieldMood:
		return r.Mood
	case RequirementFieldScene:
		return r.Scene
	case RequirementFieldFocus:
		return r.Focus
	default:
		return RequirementItem{}
	}
}

func (r *Inspiration) Set(field RequirementField, item RequirementItem) {
	if r == nil {
		return
	}
	switch field {
	case RequirementFieldMood:
		r.Mood = item
	case RequirementFieldScene:
		r.Scene = item
	case RequirementFieldFocus:
		r.Focus = item
	}
}

// IsReadyToGenerate reports whether the session has enough structured info to move forward with plan generation.
func (s *InspirationSession) IsReadyToGenerate() bool {
	if s == nil {
		return false
	}
	current, ok := s.CurrentRequirement()
	if !ok {
		return false
	}

	mood := current.Mood.Score
	scene := current.Scene.Score
	focus := current.Focus.Score

	if mood >= 3 && scene >= 3 && focus >= 3 {
		return true
	}
	if (mood >= 4 && scene >= 2) || (focus >= 4 && scene >= 2) {
		return true
	}
	return false
}

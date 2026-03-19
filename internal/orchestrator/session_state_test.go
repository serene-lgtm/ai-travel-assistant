package orchestrator

import (
	"testing"

	"ai-reading-assistant/internal/model"
)

func TestPickNextFieldPrioritizesSceneCandidates(t *testing.T) {
	session := &model.InspirationSession{
		Inspirations: []model.Inspiration{
			{
				Mood:            model.RequirementItem{Score: 1},
				Scene:           model.RequirementItem{Score: 2},
				Focus:           model.RequirementItem{Score: 4},
				SceneCandidates: []string{"东京神保町", "北海道小樽"},
			},
		},
	}

	got, err := pickNextField(session)
	if err != nil {
		t.Fatalf("pickNextField() error = %v", err)
	}
	if got != model.RequirementFieldScene {
		t.Fatalf("pickNextField() = %q, want %q", got, model.RequirementFieldScene)
	}
}

func TestPickNextFieldPrioritizesFocusForBehaviorDrivenInput(t *testing.T) {
	session := &model.InspirationSession{
		Inspirations: []model.Inspiration{
			{
				Mood:  model.RequirementItem{Score: 2},
				Scene: model.RequirementItem{Score: 1},
				Focus: model.RequirementItem{Score: 4},
			},
		},
	}

	got, err := pickNextField(session)
	if err != nil {
		t.Fatalf("pickNextField() error = %v", err)
	}
	if got != model.RequirementFieldFocus {
		t.Fatalf("pickNextField() = %q, want %q", got, model.RequirementFieldFocus)
	}
}

package model

import "testing"

func TestIsReadyToGenerateBlocksUnresolvedSceneCandidates(t *testing.T) {
	session := &InspirationSession{
		Inspirations: []Inspiration{
			{
				Mood:            RequirementItem{Score: 4},
				Scene:           RequirementItem{Score: 4},
				Focus:           RequirementItem{Score: 4},
				SceneCandidates: []string{"东京神保町", "北海道小樽"},
			},
		},
	}

	if session.IsReadyToGenerate() {
		t.Fatal("IsReadyToGenerate() = true, want false")
	}
}

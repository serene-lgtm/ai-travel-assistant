package mongo_dto

import (
	"testing"
	"time"

	"ai-reading-assistant/internal/model"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestInspirationSessionDTO_RoundTripWithWikiDefinition(t *testing.T) {
	sessionID := primitive.NewObjectID().Hex()
	userID := primitive.NewObjectID().Hex()
	createdAt := time.Now().UTC().Truncate(time.Second)

	session := &model.InspirationSession{
		ID:       sessionID,
		UserID:   userID,
		MaxToken: 1024,
		Messages: []string{"m1", "m2"},
		Inspirations: []model.Inspiration{
			{
				ID:     "1",
				UserID: userID,
				Mood: model.RequirementItem{
					Content:        "放松",
					Score:          4,
					SelectedOption: "安静",
				},
				Scene: model.RequirementItem{
					Content: "自然",
					Score:   3,
				},
				Focus: model.RequirementItem{
					Content: "地貌",
					Score:   5,
				},
				Output: "去看溶洞和山地景观",
				KeyWords: []model.KeyWord{
					{
						Content: "溶洞",
						Start:   "0",
						End:     "2",
						WikiDefinition: &model.WikiDefinition{
							Title:   "溶洞",
							Summary: "石灰岩地区形成的洞穴。",
							FullURL: "https://zh.wikipedia.org/wiki/%E6%BA%B6%E6%B4%9E",
						},
					},
				},
				IsFavorite: true,
			},
		},
		Status:    model.SessionStatusCompleted,
		CreatedAt: createdAt,
	}

	dto, err := InspirationSessionToDTO(session)
	if err != nil {
		t.Fatalf("InspirationSessionToDTO error: %v", err)
	}
	if len(dto.Inspirations) != 1 || len(dto.Inspirations[0].KeyWords) != 1 {
		t.Fatalf("expected one inspiration with one keyword, got %+v", dto.Inspirations)
	}
	if dto.Inspirations[0].KeyWords[0].WikiDefinition == nil {
		t.Fatalf("expected wiki definition in dto")
	}

	got, err := InspirationSessionFromDTO(dto)
	if err != nil {
		t.Fatalf("InspirationSessionFromDTO error: %v", err)
	}
	if len(got.Inspirations) != 1 || len(got.Inspirations[0].KeyWords) != 1 {
		t.Fatalf("expected one inspiration with one keyword after decode, got %+v", got.Inspirations)
	}
	wiki := got.Inspirations[0].KeyWords[0].WikiDefinition
	if wiki == nil {
		t.Fatalf("expected wiki definition after decode")
	}
	if wiki.Title != "溶洞" || wiki.FullURL == "" {
		t.Fatalf("unexpected wiki definition after decode: %+v", wiki)
	}
}

package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var fixtureFiles = []string{
	"intent_cases.json",
	"clarification_cases.json",
	"inspiration_cases.json",
}

func LoadCases(root string) ([]PromptEvalCase, error) {
	var all []PromptEvalCase
	seen := make(map[string]struct{})

	for _, name := range fixtureFiles {
		path := filepath.Join(root, "eval", "testdata", "prompt_eval", name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read fixture %s: %w", path, err)
		}

		var cases []PromptEvalCase
		if err := json.Unmarshal(data, &cases); err != nil {
			return nil, fmt.Errorf("parse fixture %s: %w", path, err)
		}

		for _, item := range cases {
			if err := validateCase(item); err != nil {
				return nil, fmt.Errorf("fixture %s case %s: %w", path, item.ID, err)
			}
			if _, ok := seen[item.ID]; ok {
				return nil, fmt.Errorf("duplicate case id %s", item.ID)
			}
			seen[item.ID] = struct{}{}
			all = append(all, item)
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].ID < all[j].ID
	})
	return all, nil
}

func validateCase(item PromptEvalCase) error {
	if strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("id is required")
	}
	switch item.Category {
	case CategoryIntent, CategoryClarification, CategoryInspiration:
	default:
		return fmt.Errorf("invalid category %q", item.Category)
	}
	if strings.TrimSpace(item.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if len(item.Rubric) == 0 {
		return fmt.Errorf("rubric is required")
	}
	if isEmptyExpected(item.Expected) {
		return fmt.Errorf("expected is required")
	}
	switch item.Category {
	case CategoryIntent:
		if strings.TrimSpace(item.Input.Content) == "" {
			return fmt.Errorf("intent input.content is required")
		}
		if item.Expected.TravelRelated == nil {
			return fmt.Errorf("intent expected.travel_related is required")
		}
	case CategoryClarification:
		if item.SessionState == nil {
			return fmt.Errorf("clarification session_state is required")
		}
		if strings.TrimSpace(item.SessionState.TargetField) == "" {
			return fmt.Errorf("clarification session_state.target_field is required")
		}
	case CategoryInspiration:
		if item.SessionState == nil {
			return fmt.Errorf("inspiration session_state is required")
		}
	}
	return nil
}

func isEmptyExpected(expected FixtureExpected) bool {
	return expected.TravelRelated == nil &&
		expected.TargetField == "" &&
		expected.OptionCountMin == 0 &&
		expected.OptionCountMax == 0 &&
		len(expected.ContentMustInclude) == 0 &&
		len(expected.ContentMustAvoid) == 0 &&
		!expected.SingleCorePlaceRequired
}

func FilterCases(cases []PromptEvalCase, ids []string) []PromptEvalCase {
	if len(ids) == 0 {
		return cases
	}
	allowed := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		allowed[id] = struct{}{}
	}
	filtered := make([]PromptEvalCase, 0, len(allowed))
	for _, item := range cases {
		if _, ok := allowed[item.ID]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

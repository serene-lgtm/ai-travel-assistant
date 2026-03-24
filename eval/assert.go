package eval

import (
	"fmt"
	"strings"
)

func EvaluateAssertions(item PromptEvalCase, actual any, runErr error) ([]Assertion, bool) {
	if runErr != nil {
		return []Assertion{{
			Name:    "run_error",
			Passed:  false,
			Details: runErr.Error(),
		}}, false
	}

	var assertions []Assertion

	switch item.Category {
	case CategoryIntent:
		value, _ := actual.(IntentActual)
		if item.Expected.TravelRelated != nil {
			assertions = append(assertions, Assertion{
				Name:    "travel_related",
				Passed:  value.TravelRelated == *item.Expected.TravelRelated,
				Details: fmt.Sprintf("actual=%t expected=%t", value.TravelRelated, *item.Expected.TravelRelated),
			})
		}
	case CategoryClarification:
		value, _ := actual.(ClarificationActual)
		if strings.TrimSpace(item.Expected.TargetField) != "" {
			assertions = append(assertions, Assertion{
				Name:    "target_field",
				Passed:  value.TargetField == item.Expected.TargetField,
				Details: fmt.Sprintf("actual=%q expected=%q", value.TargetField, item.Expected.TargetField),
			})
		}
		if item.Expected.OptionCountMin > 0 {
			assertions = append(assertions, Assertion{
				Name:    "option_count_min",
				Passed:  len(value.Options) >= item.Expected.OptionCountMin,
				Details: fmt.Sprintf("actual=%d expected>=%d", len(value.Options), item.Expected.OptionCountMin),
			})
		}
		if item.Expected.OptionCountMax > 0 {
			assertions = append(assertions, Assertion{
				Name:    "option_count_max",
				Passed:  len(value.Options) <= item.Expected.OptionCountMax,
				Details: fmt.Sprintf("actual=%d expected<=%d", len(value.Options), item.Expected.OptionCountMax),
			})
		}
		assertions = append(assertions, matchTextAssertions([]string{
			value.Question,
			value.TargetField,
			strings.Join(value.Options, "\n"),
		}, item.Expected)...)
	case CategoryInspiration:
		value, _ := actual.(InspirationActual)
		assertions = append(assertions, matchTextAssertions([]string{value.Content}, item.Expected)...)
	}

	passed := true
	for _, assertion := range assertions {
		if !assertion.Passed {
			passed = false
			break
		}
	}
	return assertions, passed
}

func matchTextAssertions(parts []string, expected FixtureExpected) []Assertion {
	text := strings.Join(parts, "\n")
	var assertions []Assertion

	for _, needle := range expected.ContentMustInclude {
		needle = strings.TrimSpace(needle)
		if needle == "" {
			continue
		}
		assertions = append(assertions, Assertion{
			Name:    "content_must_include",
			Passed:  strings.Contains(text, needle),
			Details: fmt.Sprintf("needle=%q", needle),
		})
	}

	for _, needle := range expected.ContentMustAvoid {
		needle = strings.TrimSpace(needle)
		if needle == "" {
			continue
		}
		assertions = append(assertions, Assertion{
			Name:    "content_must_avoid",
			Passed:  !strings.Contains(text, needle),
			Details: fmt.Sprintf("needle=%q", needle),
		})
	}

	return assertions
}

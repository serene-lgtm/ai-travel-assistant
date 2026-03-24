package eval

func EvaluateRun(run *PromptEvalRun, cases []PromptEvalCase, progress func(index, total int, item PromptEvalCase)) {
	caseByID := make(map[string]PromptEvalCase, len(cases))
	for _, item := range cases {
		caseByID[item.ID] = item
	}

	for i := range run.Results {
		result := &run.Results[i]
		item, ok := caseByID[result.CaseID]
		if !ok {
			result.Assertions = []Assertion{{
				Name:    "fixture_lookup",
				Passed:  false,
				Details: "case not found in loaded fixtures",
			}}
			result.Passed = false
			continue
		}
		if progress != nil {
			progress(i+1, len(run.Results), item)
		}
		runErr := error(nil)
		if result.Error != "" {
			runErr = evalError(result.Error)
		}
		result.Assertions, result.Passed = EvaluateAssertions(item, decodeActual(item.Category, result.Actual), runErr)
	}
}

func decodeActual(category Category, raw any) any {
	switch category {
	case CategoryIntent:
		if value, ok := raw.(IntentActual); ok {
			return value
		}
		if value, ok := raw.(map[string]any); ok {
			return IntentActual{
				TravelRelated: readBool(value, "travel_related"),
				RawResponse:   readString(value, "raw_response"),
			}
		}
	case CategoryClarification:
		if value, ok := raw.(ClarificationActual); ok {
			return value
		}
		if value, ok := raw.(map[string]any); ok {
			return ClarificationActual{
				Question:    readString(value, "question"),
				Options:     readStringSlice(value, "options"),
				TargetField: readString(value, "target_field"),
			}
		}
	case CategoryInspiration:
		if value, ok := raw.(InspirationActual); ok {
			return value
		}
		if value, ok := raw.(map[string]any); ok {
			return InspirationActual{
				Content: readString(value, "content"),
			}
		}
	}
	return raw
}

type evalError string

func (e evalError) Error() string { return string(e) }

func readString(value map[string]any, key string) string {
	text, _ := value[key].(string)
	return text
}

func readBool(value map[string]any, key string) bool {
	flag, _ := value[key].(bool)
	return flag
}

func readStringSlice(value map[string]any, key string) []string {
	raw, ok := value[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if ok {
			out = append(out, text)
		}
	}
	return out
}

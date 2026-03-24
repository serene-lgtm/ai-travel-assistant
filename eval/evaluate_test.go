package eval

import "testing"

func TestEvaluateRun(t *testing.T) {
	run := &PromptEvalRun{
		Results: []PromptEvalResult{
			{
				CaseID:   "intent_1",
				Category: CategoryIntent,
				Actual: IntentActual{
					TravelRelated: true,
				},
			},
		},
	}
	cases := []PromptEvalCase{
		{
			ID:       "intent_1",
			Category: CategoryIntent,
			Expected: FixtureExpected{TravelRelated: boolPtr(true)},
		},
	}

	EvaluateRun(run, cases, nil)

	if !run.Results[0].Passed {
		t.Fatalf("Passed = false, want true")
	}
	if len(run.Results[0].Assertions) != 1 {
		t.Fatalf("len(Assertions) = %d, want 1", len(run.Results[0].Assertions))
	}
}

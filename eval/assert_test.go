package eval

import "testing"

func TestEvaluateAssertionsIntent(t *testing.T) {
	item := PromptEvalCase{
		Category: CategoryIntent,
		Expected: FixtureExpected{TravelRelated: boolPtr(true)},
	}
	assertions, passed := EvaluateAssertions(item, IntentActual{TravelRelated: true}, nil)
	if !passed {
		t.Fatalf("passed = false, assertions=%#v", assertions)
	}
}

func TestEvaluateAssertionsClarification(t *testing.T) {
	item := PromptEvalCase{
		Category: CategoryClarification,
		Expected: FixtureExpected{
			TargetField:    "scene",
			OptionCountMin: 2,
			OptionCountMax: 4,
		},
	}
	actual := ClarificationActual{
		Question:    "这次更想先落到哪个地方？",
		Options:     []string{"波尔图", "会安"},
		TargetField: "scene",
	}
	assertions, passed := EvaluateAssertions(item, actual, nil)
	if !passed {
		t.Fatalf("passed = false, assertions=%#v", assertions)
	}
}

func TestEvaluateAssertionsInspiration(t *testing.T) {
	item := PromptEvalCase{
		Category: CategoryInspiration,
		Expected: FixtureExpected{
			ContentMustInclude: []string{"小樽"},
			ContentMustAvoid:   []string{"京都"},
		},
	}
	actual := InspirationActual{Content: "小樽的港口在傍晚会慢慢安静下来。"}
	assertions, passed := EvaluateAssertions(item, actual, nil)
	if !passed {
		t.Fatalf("passed = false, assertions=%#v", assertions)
	}
}

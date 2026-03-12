package agent

import "testing"

func TestIntentAgentDetectTravelIntent(t *testing.T) {
	agent := &intentAgent{
		llmClient: stubChatCaller{
			response: `分析如下：
{"travel_related": true}`,
		},
	}

	result, err := agent.DetectTravelIntent(t.Context(), "我想去敦煌找一点历史感")
	if err != nil {
		t.Fatalf("DetectTravelIntent error: %v", err)
	}
	if !result.TravelRelated {
		t.Fatalf("expected travel related true, got false")
	}
}

type stubChatCaller struct {
	response string
	err      error
}

func (s stubChatCaller) Call(_ string) (string, error) {
	return s.response, s.err
}

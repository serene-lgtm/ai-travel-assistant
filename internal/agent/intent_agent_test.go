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

func TestIntentAgentDetectTravelIntentRejectsExplicitNonTravelBookQuery(t *testing.T) {
	agent := &intentAgent{
		llmClient: stubChatCaller{
			response: `{"travel_related": true}`,
		},
	}

	result, err := agent.DetectTravelIntent(t.Context(), "我不想旅游，我只是想找几本关于西南边地书写的书。")
	if err != nil {
		t.Fatalf("DetectTravelIntent error: %v", err)
	}
	if result.TravelRelated {
		t.Fatalf("expected travel related false, got true")
	}
}

func TestIntentAgentDetectTravelIntentRejectsLiteraryExplanationQuery(t *testing.T) {
	agent := &intentAgent{
		llmClient: stubChatCaller{
			response: `{"travel_related": true}`,
		},
	}

	result, err := agent.DetectTravelIntent(t.Context(), "请解释一下物哀是什么意思，它为什么常被拿来形容日本文学？")
	if err != nil {
		t.Fatalf("DetectTravelIntent error: %v", err)
	}
	if result.TravelRelated {
		t.Fatalf("expected travel related false, got true")
	}
}

type stubChatCaller struct {
	response string
	err      error
}

func (s stubChatCaller) Call(_ string) (string, error) {
	return s.response, s.err
}

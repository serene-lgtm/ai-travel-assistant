package agent

import (
	"testing"
)

func TestKeywordAgentExtractKeywordsFromOutput(t *testing.T) {
	agent := &keywordAgent{
		llmClient: stubChatCaller{
			response: `{"keywords":["敦煌","莫高窟","河西走廊","敦煌","风"]}`,
		},
	}

	output := "她会从敦煌出发，沿着河西走廊缓缓前行，在莫高窟停下，听风穿过崖壁。"
	got, err := agent.ExtractKeywordsFromOutput(t.Context(), output)
	if err != nil {
		t.Fatalf("ExtractKeywordsFromOutput error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 keywords, got %d", len(got))
	}
	if got[0].Content != "敦煌" || got[0].Start != "3" {
		t.Fatalf("unexpected first keyword: %+v", got[0])
	}
	if got[1].Content != "莫高窟" {
		t.Fatalf("unexpected second keyword: %+v", got[1])
	}
	if got[2].Content != "河西走廊" {
		t.Fatalf("unexpected third keyword: %+v", got[2])
	}
}

func TestBuildKeywordsFromOutputSkipsMissingAndDuplicate(t *testing.T) {
	output := "她会在京都哲学之道散步。"
	got := buildKeywordsFromOutput(output, []string{"京都", "哲学之道", "京都", "不存在"})
	if len(got) != 2 {
		t.Fatalf("expected 2 keywords, got %d", len(got))
	}
	if got[0].Content != "京都" || got[1].Content != "哲学之道" {
		t.Fatalf("unexpected keywords: %+v", got)
	}
}

func TestBuildKeywordsFromOutputSkipsOverSpecificPlaceChain(t *testing.T) {
	output := "她会前往福贡县老姆登村，最后在梯田边停下来。"
	got := buildKeywordsFromOutput(output, []string{"福贡县老姆登村", "梯田"})
	if len(got) != 1 {
		t.Fatalf("expected 1 keyword, got %d", len(got))
	}
	if got[0].Content != "梯田" {
		t.Fatalf("unexpected keyword: %+v", got[0])
	}
}

package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteInspirationABRunWritesMarkdownCompanion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ab.json")
	run := &InspirationABRun{
		Label: "ab_minimal_batch",
		Items: []InspirationABItem{
			{
				CaseID:      "insp_1",
				Description: "神保町旧书街的灵感对比",
				Input:       "想去神保町待几天。",
				Query:       "东京神保町 | 旧书店 | 怀旧",
				RAGLookups: []EvalRAGLookup{
					{
						Query:   "东京神保町",
						Title:   "神保町",
						Summary: "东京千代田区著名旧书街。",
						Source:  "https://zh.wikipedia.org/wiki/神保町",
						Hit:     true,
					},
				},
				BaselineOutput:              "旧书街的午后\n\n东京神保町适合慢慢逛旧书店，也适合写作。",
				RAGOutput:                   "标题：纸页间的旧东京\n\n东京神保町的旧书店和写作气氛，在午后尤其明显。",
				BaselineLatencyMs:           1200,
				BaselineGenerationLatencyMs: 1200,
				RAGLatencyMs:                1800,
				RAGQueryLatencyMs:           250,
				RAGWikiLatencyMs:            350,
				RAGGenerationLatencyMs:      1200,
				RAGContext:                  "context text",
				RAGDocuments: []EvalRAGDocument{
					{Title: "神保町", Source: "wikipedia", Score: 0.98},
				},
			},
		},
	}

	if err := WriteInspirationABRun(path, run); err != nil {
		t.Fatalf("WriteInspirationABRun() error = %v", err)
	}

	mdPath := filepath.Join(dir, "ab.md")
	data, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", mdPath, err)
	}

	got := string(data)
	for _, want := range []string{
		"# Inspiration A/B Run",
		"## Case 1",
		"Case ID: `insp_1`",
		"### Latency",
		"Baseline total: `1200 ms`",
		"Generation: `1200 ms`",
		"RAG total: `1800 ms`",
		"Query extraction: `250 ms`",
		"Wikipedia lookup: `350 ms`",
		"### Query Lookups",
		"Query: 东京神保町",
		"Definition Summary:",
		"东京千代田区著名旧书街。",
		"### Baseline",
		"<summary>Full text</summary>",
		"旧书街的午后",
		"### RAG",
		"纸页间的旧东京",
		"### RAG Documents",
		"神保町",
		"<summary>RAG Context</summary>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q\n%s", want, got)
		}
	}
}

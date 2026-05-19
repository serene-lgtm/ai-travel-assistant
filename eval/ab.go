package eval

import (
	"context"
	"fmt"
	"strings"
)

func RunInspirationAB(ctx context.Context, root string, cases []PromptEvalCase, label string, selected []string, progress func(index, total int, item PromptEvalCase)) (*InspirationABRun, error) {
	if len(cases) == 0 {
		return nil, fmt.Errorf("no inspiration cases selected")
	}

	baselineRunner, err := NewRunner(root)
	if err != nil {
		return nil, err
	}
	baselineRunner.SetRAGEnabled(false)

	ragRunner, err := NewRunner(root)
	if err != nil {
		return nil, err
	}
	ragRunner.SetRAGEnabled(true)

	run := &InspirationABRun{
		Label: strings.TrimSpace(label),
		Items: make([]InspirationABItem, 0, len(cases)),
	}

	for i, item := range cases {
		if progress != nil {
			progress(i+1, len(cases), item)
		}
		abItem := InspirationABItem{
			CaseID:      strings.TrimSpace(item.ID),
			Description: strings.TrimSpace(item.Description),
			Input:       strings.TrimSpace(item.Input.Content),
		}

		baselineActual, baselineErr := baselineRunner.runCase(ctx, item)
		fillABSide(&abItem, baselineActual, baselineErr, false)

		ragActual, ragErr := ragRunner.runCase(ctx, item)
		fillABSide(&abItem, ragActual, ragErr, true)

		run.Items = append(run.Items, abItem)
	}

	return run, nil
}

func fillABSide(item *InspirationABItem, actual any, runErr error, rag bool) {
	if item == nil {
		return
	}

	if runErr != nil {
		if rag {
			item.RAGError = runErr.Error()
		} else {
			item.BaselineError = runErr.Error()
		}
		return
	}

	output, ok := actual.(inspirationRunOutput)
	if !ok {
		return
	}

	if rag {
		item.RAGOutput = strings.TrimSpace(output.Actual.Content)
		if output.Trace != nil {
			item.Query = strings.TrimSpace(output.Trace.RAGQuery)
			item.RAGLookups = append([]EvalRAGLookup(nil), output.Trace.RAGLookups...)
			item.RAGDocuments = append([]EvalRAGDocument(nil), output.Trace.RAGDocuments...)
			item.RAGContext = strings.TrimSpace(output.Trace.RAGReferenceText)
			item.RAGLatencyMs = output.Trace.TotalLatencyMs
			item.RAGQueryLatencyMs = output.Trace.QueryLatencyMs
			item.RAGWikiLatencyMs = output.Trace.WikiLatencyMs
			item.RAGGenerationLatencyMs = output.Trace.GenerationLatencyMs
		}
		return
	}

	item.BaselineOutput = strings.TrimSpace(output.Actual.Content)
	if output.Trace != nil {
		item.BaselineLatencyMs = output.Trace.TotalLatencyMs
		item.BaselineGenerationLatencyMs = output.Trace.GenerationLatencyMs
	}
}

package eval

import (
	"context"
	"fmt"
	"strings"
)

func RunInspirationAB(ctx context.Context, root string, cases []PromptEvalCase, label string, selected []string) (*InspirationABRun, error) {
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

	for _, item := range cases {
		abItem := InspirationABItem{
			Input: strings.TrimSpace(item.Input.Content),
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
			item.RAGDocuments = append([]EvalRAGDocument(nil), output.Trace.RAGDocuments...)
			item.RAGContext = strings.TrimSpace(output.Trace.RAGReferenceText)
		}
		return
	}

	item.BaselineOutput = strings.TrimSpace(output.Actual.Content)
}

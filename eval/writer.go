package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const markdownWrapWidth = 88

func DefaultOutputPath(root, runID string) string {
	return filepath.Join(root, "artifacts", "prompt_eval", runID+".json")
}

func WriteRun(path string, run *PromptEvalRun) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir output dir: %w", err)
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write run file: %w", err)
	}
	return nil
}

func WriteInspirationABRun(path string, run *InspirationABRun) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir output dir: %w", err)
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ab run: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write ab run file: %w", err)
	}
	if err := writeInspirationABMarkdown(path, run); err != nil {
		return fmt.Errorf("write ab markdown file: %w", err)
	}
	return nil
}

func writeInspirationABMarkdown(path string, run *InspirationABRun) error {
	mdPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".md"
	content := renderInspirationABMarkdown(run)
	if err := os.WriteFile(mdPath, []byte(content), 0o644); err != nil {
		return err
	}
	return nil
}

func renderInspirationABMarkdown(run *InspirationABRun) string {
	if run == nil {
		return "# Inspiration A/B Run\n"
	}

	var b strings.Builder
	label := strings.TrimSpace(run.Label)
	if label == "" {
		label = "inspiration_ab"
	}

	b.WriteString("# Inspiration A/B Run\n\n")
	b.WriteString(fmt.Sprintf("- Label: `%s`\n", label))
	b.WriteString(fmt.Sprintf("- Items: `%d`\n\n", len(run.Items)))

	for i, item := range run.Items {
		b.WriteString(fmt.Sprintf("## Case %d\n\n", i+1))
		if item.CaseID != "" {
			b.WriteString(fmt.Sprintf("- Case ID: `%s`\n", item.CaseID))
		}
		if item.Description != "" {
			b.WriteString("- Description: ")
			writeWrappedLine(&b, item.Description, markdownWrapWidth, 2)
		}
		if item.CaseID != "" || item.Description != "" {
			b.WriteString("\n")
		}
		if item.Input != "" {
			b.WriteString("### Input\n\n")
			writeWrappedText(&b, item.Input, markdownWrapWidth)
			b.WriteString("\n")
		}
		if item.Query != "" {
			b.WriteString("### RAG Query\n\n")
			writeWrappedCodeLine(&b, item.Query, markdownWrapWidth)
			b.WriteString("\n")
		}
		b.WriteString("### Latency\n\n")
		writeWrappedBullet(&b, fmt.Sprintf("Baseline total: `%d ms`", item.BaselineLatencyMs), markdownWrapWidth)
		writeWrappedIndented(&b, fmt.Sprintf("Generation: `%d ms`", item.BaselineGenerationLatencyMs), markdownWrapWidth, 2)
		writeWrappedBullet(&b, fmt.Sprintf("RAG total: `%d ms`", item.RAGLatencyMs), markdownWrapWidth)
		writeWrappedIndented(&b, fmt.Sprintf("Query extraction: `%d ms`", item.RAGQueryLatencyMs), markdownWrapWidth, 2)
		writeWrappedIndented(&b, fmt.Sprintf("Wikipedia lookup: `%d ms`", item.RAGWikiLatencyMs), markdownWrapWidth, 2)
		writeWrappedIndented(&b, fmt.Sprintf("Generation: `%d ms`", item.RAGGenerationLatencyMs), markdownWrapWidth, 2)
		b.WriteString("\n")
		if len(item.RAGLookups) > 0 {
			b.WriteString("### Query Lookups\n\n")
			for _, lookup := range item.RAGLookups {
				title := strings.TrimSpace(lookup.Title)
				if title == "" {
					title = "-"
				}
				status := "miss"
				if lookup.Hit {
					status = "hit"
				}
				writeWrappedBullet(&b, fmt.Sprintf("Query: %s", fallbackText(lookup.Query)), markdownWrapWidth)
				writeWrappedIndented(&b, fmt.Sprintf("Status: %s", status), markdownWrapWidth, 2)
				writeWrappedIndented(&b, fmt.Sprintf("Title: %s", title), markdownWrapWidth, 2)
				if strings.TrimSpace(lookup.Summary) != "" {
					writeWrappedIndented(&b, "Definition Summary:", markdownWrapWidth, 2)
					writeWrappedIndented(&b, lookup.Summary, markdownWrapWidth, 4)
				}
				if strings.TrimSpace(lookup.Source) != "" {
					writeWrappedIndented(&b, fmt.Sprintf("Source: %s", lookup.Source), markdownWrapWidth, 2)
				}
			}
			b.WriteString("\n")
		}
		if len(item.RAGDocuments) > 0 {
			b.WriteString("### RAG Documents\n\n")
			for _, doc := range item.RAGDocuments {
				title := strings.TrimSpace(doc.Title)
				if title == "" {
					title = "(untitled)"
				}
				line := title
				if doc.Score != 0 {
					line += fmt.Sprintf(" (score: %.3f)", doc.Score)
				}
				if doc.Source != "" {
					line += fmt.Sprintf(" [%s]", doc.Source)
				}
				writeWrappedBullet(&b, line, markdownWrapWidth)
			}
			b.WriteString("\n")
		}

		b.WriteString("### Baseline\n\n")
		if item.BaselineError != "" {
			b.WriteString(fmt.Sprintf("Error: %s\n\n", item.BaselineError))
		} else {
			writeOutputSection(&b, item.BaselineOutput)
		}

		b.WriteString("### RAG\n\n")
		if item.RAGError != "" {
			b.WriteString(fmt.Sprintf("Error: %s\n\n", item.RAGError))
		} else {
			writeOutputSection(&b, item.RAGOutput)
		}

		if item.RAGContext != "" {
			b.WriteString("<details>\n<summary>RAG Context</summary>\n\n")
			writeWrappedText(&b, item.RAGContext, markdownWrapWidth)
			b.WriteString("\n</details>\n\n")
		}
	}

	return b.String()
}

func writeOutputSection(b *strings.Builder, output string) {
	output = strings.TrimSpace(output)
	if output == "" {
		b.WriteString("(empty)\n\n")
		return
	}

	b.WriteString(summarizeText(output, 120))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("- Length: `%d chars / %d paragraphs`\n\n", utf8.RuneCountInString(output), paragraphCount(output)))
	b.WriteString("<details>\n<summary>Full text</summary>\n\n")
	writeWrappedText(b, output, markdownWrapWidth)
	b.WriteString("\n</details>\n\n")
}

func summarizeText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	summary := strings.ReplaceAll(text, "\n\n", " ")
	summary = strings.ReplaceAll(summary, "\n", " ")
	summary = strings.Join(strings.Fields(summary), " ")
	runes := []rune(summary)
	if len(runes) <= limit {
		return summary
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}

func extractTitle(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}
	first := strings.TrimSpace(lines[0])
	first = strings.TrimPrefix(first, "标题：")
	return strings.TrimSpace(first)
}

func paragraphCount(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	parts := strings.Split(text, "\n\n")
	count := 0
	for _, part := range parts {
		if strings.TrimSpace(part) != "" {
			count++
		}
	}
	return count
}

func fallbackText(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func writeWrappedText(b *strings.Builder, text string, width int) {
	text = strings.TrimSpace(text)
	if text == "" {
		b.WriteString("(empty)\n")
		return
	}

	paragraphs := strings.Split(text, "\n")
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			b.WriteString("\n")
			continue
		}
		for _, line := range wrapText(paragraph, width) {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
}

func writeWrappedBullet(b *strings.Builder, text string, width int) {
	b.WriteString("- ")
	writeWrappedLine(b, text, width, 2)
}

func writeWrappedIndented(b *strings.Builder, text string, width int, indent int) {
	b.WriteString(strings.Repeat(" ", indent))
	writeWrappedLine(b, text, width, indent)
}

func writeWrappedLine(b *strings.Builder, text string, width int, indent int) {
	lines := wrapText(text, width-indent)
	padding := strings.Repeat(" ", indent)
	for i, line := range lines {
		if i > 0 {
			b.WriteString(padding)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
}

func writeWrappedCodeLine(b *strings.Builder, text string, width int) {
	lines := wrapText(text, width-2)
	for _, line := range lines {
		b.WriteString("`")
		b.WriteString(line)
		b.WriteString("`\n")
	}
}

func wrapText(text string, width int) []string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return []string{""}
	}
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	lines := make([]string, 0, len(words))
	current := ""
	for _, word := range words {
		if utf8.RuneCountInString(word) > width {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			lines = append(lines, breakLongWord(word, width)...)
			continue
		}
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if utf8.RuneCountInString(candidate) <= width {
			current = candidate
			continue
		}
		lines = append(lines, current)
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func breakLongWord(word string, width int) []string {
	runes := []rune(word)
	if len(runes) <= width || width <= 0 {
		return []string{word}
	}
	lines := make([]string, 0, (len(runes)+width-1)/width)
	for len(runes) > 0 {
		end := width
		if end > len(runes) {
			end = len(runes)
		}
		lines = append(lines, string(runes[:end]))
		runes = runes[end:]
	}
	return lines
}

func ReadRun(path string) (*PromptEvalRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read run file: %w", err)
	}
	var run PromptEvalRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parse run file: %w", err)
	}
	return &run, nil
}

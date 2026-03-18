package agent

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

func decodeFirstJSONObject(raw string, dst any) error {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || start > end {
		return fmt.Errorf("response missing json object: %s", raw)
	}
	return json.Unmarshal([]byte(raw[start:end+1]), dst)
}

func normalizeScore(raw any) int {
	score := 0
	switch v := raw.(type) {
	case float64:
		score = int(math.Round(v))
	case string:
		v = strings.TrimSpace(v)
		if n, err := strconv.Atoi(v); err == nil {
			score = n
		}
	case json.Number:
		if n, err := v.Int64(); err == nil {
			score = int(n)
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 5 {
		score = 5
	}
	return score
}

var markdownLinkPattern = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)

func stripMarkdownToPlainText(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "```markdown", "")
	raw = strings.ReplaceAll(raw, "```md", "")
	raw = strings.ReplaceAll(raw, "```", "")
	raw = markdownLinkPattern.ReplaceAllString(raw, "$1")

	lines := strings.Split(raw, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			cleaned = append(cleaned, "")
			continue
		}

		for _, prefix := range []string{"### ", "## ", "# ", "- ", "* ", "> "} {
			if strings.HasPrefix(line, prefix) {
				line = strings.TrimSpace(strings.TrimPrefix(line, prefix))
				break
			}
		}

		if len(line) >= 3 && line[1] == '.' && line[0] >= '0' && line[0] <= '9' {
			line = strings.TrimSpace(line[2:])
		}

		replacer := strings.NewReplacer("**", "", "__", "", "*", "", "_", "", "`", "")
		line = replacer.Replace(line)
		cleaned = append(cleaned, line)
	}

	result := strings.Join(cleaned, "\n")
	result = strings.TrimSpace(result)

	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}
	return result
}

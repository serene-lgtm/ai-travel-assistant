package agent

import (
	"encoding/json"
	"fmt"
	"math"
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

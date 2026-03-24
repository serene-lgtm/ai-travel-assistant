package eval

import (
	"fmt"
	"sort"
)

func CompareRuns(baselinePath, candidatePath string) (*CompareSummary, error) {
	baseline, err := ReadRun(baselinePath)
	if err != nil {
		return nil, err
	}
	candidate, err := ReadRun(candidatePath)
	if err != nil {
		return nil, err
	}

	baseByID := make(map[string]PromptEvalResult, len(baseline.Results))
	for _, item := range baseline.Results {
		baseByID[item.CaseID] = item
	}

	summary := &CompareSummary{
		BaselinePath:  baselinePath,
		CandidatePath: candidatePath,
		TotalCases:    len(candidate.Results),
	}

	for _, cand := range candidate.Results {
		base, ok := baseByID[cand.CaseID]
		if !ok {
			continue
		}
		regressions, improvements, scored := compareScores(base, cand)
		if !scored {
			summary.Unscored = append(summary.Unscored, cand.CaseID)
			continue
		}
		summary.ScoredCases++
		summary.Regressions = append(summary.Regressions, regressions...)
		summary.Improvements = append(summary.Improvements, improvements...)
	}

	sort.Slice(summary.Regressions, func(i, j int) bool { return summary.Regressions[i].CaseID < summary.Regressions[j].CaseID })
	sort.Slice(summary.Improvements, func(i, j int) bool { return summary.Improvements[i].CaseID < summary.Improvements[j].CaseID })
	sort.Strings(summary.Unscored)
	return summary, nil
}

func compareScores(base, cand PromptEvalResult) ([]CompareCaseChange, []CompareCaseChange, bool) {
	if len(base.ManualScore) == 0 || len(cand.ManualScore) == 0 {
		return nil, nil, false
	}

	var regressions []CompareCaseChange
	var improvements []CompareCaseChange
	scored := false

	for key, baseValue := range base.ManualScore {
		candValue, ok := cand.ManualScore[key]
		if !ok {
			continue
		}
		scored = true

		if basePass, ok := toPassFail(baseValue); ok {
			candPass, ok := toPassFail(candValue)
			if !ok {
				continue
			}
			switch {
			case basePass == "pass" && candPass == "fail":
				regressions = append(regressions, makeChange(cand, fmt.Sprintf("%s pass->fail", key)))
			case basePass == "fail" && candPass == "pass":
				improvements = append(improvements, makeChange(cand, fmt.Sprintf("%s fail->pass", key)))
			}
			continue
		}

		baseNum, ok1 := toFloat(baseValue)
		candNum, ok2 := toFloat(candValue)
		if !ok1 || !ok2 {
			continue
		}
		diff := candNum - baseNum
		switch {
		case diff <= -0.5:
			regressions = append(regressions, makeChange(cand, fmt.Sprintf("%s %.1f->%.1f", key, baseNum, candNum)))
		case diff >= 0.5:
			improvements = append(improvements, makeChange(cand, fmt.Sprintf("%s %.1f->%.1f", key, baseNum, candNum)))
		}
	}

	return regressions, improvements, scored
}

func makeChange(result PromptEvalResult, reason string) CompareCaseChange {
	return CompareCaseChange{
		CaseID:      result.CaseID,
		Category:    result.Category,
		Description: result.Description,
		Reason:      reason,
	}
}

func toPassFail(v any) (string, bool) {
	text, ok := v.(string)
	if !ok {
		return "", false
	}
	if text != "pass" && text != "fail" {
		return "", false
	}
	return text, true
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}

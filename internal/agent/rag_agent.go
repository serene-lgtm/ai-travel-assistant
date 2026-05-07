package agent

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"ai-reading-assistant/internal/config"
	"ai-reading-assistant/internal/model"
)

type RAGDocument struct {
	Title   string
	Summary string
	Source  string
	Query   string
	Score   float64
}

type RAGContext struct {
	Query         string
	Documents     []RAGDocument
	ReferenceText string
}

type RAGAgent interface {
	BuildContext(ctx context.Context, session *model.InspirationSession) (*RAGContext, error)
}

type ragAgent struct {
	knowledge wikipediaKnowledgeGetter
}

type wikipediaKnowledgeGetter interface {
	GetDefinition(ctx context.Context, keyword string) (*model.WikiDefinition, error)
}

func NewRAGAgent(knowledge wikipediaKnowledgeGetter) RAGAgent {
	return &ragAgent{knowledge: knowledge}
}

func (a *ragAgent) BuildContext(ctx context.Context, session *model.InspirationSession) (*RAGContext, error) {
	if a == nil || a.knowledge == nil || session == nil {
		return nil, nil
	}

	current, ok := session.CurrentRequirement()
	if !ok {
		return nil, nil
	}

	queries := buildRAGQueries(current)
	if len(queries) == 0 {
		return nil, nil
	}

	ragCfg := config.Global().RAG
	documents := make([]RAGDocument, 0, ragCfg.TopK)
	seenTitles := make(map[string]struct{}, ragCfg.TopK)

	for i, query := range queries {
		definition, err := a.knowledge.GetDefinition(ctx, query)
		if err != nil || definition == nil {
			continue
		}

		title := strings.TrimSpace(definition.Title)
		if title == "" {
			title = query
		}
		if _, ok := seenTitles[title]; ok {
			continue
		}
		seenTitles[title] = struct{}{}

		documents = append(documents, RAGDocument{
			Title:   title,
			Summary: strings.TrimSpace(definition.Summary),
			Source:  strings.TrimSpace(definition.FullURL),
			Query:   query,
			Score:   1.0 / float64(i+1),
		})
		if len(documents) == ragCfg.TopK {
			break
		}
	}

	if len(documents) == 0 {
		return &RAGContext{
			Query:         strings.Join(queries, " | "),
			Documents:     make([]RAGDocument, 0),
			ReferenceText: "",
		}, nil
	}

	return &RAGContext{
		Query:         strings.Join(queries, " | "),
		Documents:     documents,
		ReferenceText: formatRAGReferenceText(documents, ragCfg.MaxContextChars),
	}, nil
}

func buildRAGQueries(current *model.Inspiration) []string {
	if current == nil {
		return nil
	}

	out := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)

	appendTerms := func(raw string) {
		for _, term := range expandQueryTerms(raw) {
			if !isUsefulRAGQuery(term) {
				continue
			}
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			out = append(out, term)
		}
	}

	// Prefer location-like scene terms first, then focus terms, then mood as fallback.
	appendTerms(requirementQueryValue(current.Scene))
	appendTerms(requirementQueryValue(current.Focus))
	appendTerms(requirementQueryValue(current.Mood))

	return out
}

func BuildRAGQueryPreview(session *model.InspirationSession) []string {
	if session == nil {
		return nil
	}
	current, ok := session.CurrentRequirement()
	if !ok {
		return nil
	}
	return buildRAGQueries(current)
}

var ragQueryStopwords = map[string]struct{}{
	"风景": {},
	"景色": {},
	"地方": {},
	"城市": {},
	"小城": {},
	"古城": {},
	"古镇": {},
	"散步": {},
	"徒步": {},
	"漫步": {},
	"发呆": {},
	"看海": {},
	"看山": {},
	"拍照": {},
	"旅行": {},
	"旅游": {},
	"出发": {},
	"抵达": {},
	"休息": {},
	"放空": {},
	"秋天": {},
	"春天": {},
	"夏天": {},
	"冬天": {},
	"清晨": {},
	"早晨": {},
	"黄昏": {},
	"夜晚": {},
	"晚上": {},
	"白天": {},
	"白日": {},
}

var ragQuerySuffixStopwords = []string{
	"气息",
	"氛围",
	"感觉",
	"感受",
	"体验",
	"心情",
}

func isUsefulRAGQuery(term string) bool {
	term = strings.TrimSpace(term)
	if term == "" {
		return false
	}
	if _, ok := ragQueryStopwords[term]; ok {
		return false
	}
	for _, suffix := range ragQuerySuffixStopwords {
		if strings.HasSuffix(term, suffix) && utf8.RuneCountInString(term) <= 6 {
			return false
		}
	}
	if looksLikeLocationOrEntity(term) {
		return true
	}
	if utf8.RuneCountInString(term) < 2 {
		return false
	}
	return !looksTooAbstract(term)
}

var ragEntityMarkers = []string{
	"书院", "故居", "故里", "遗址", "石窟", "古道", "码头", "剧院", "书店",
	"左岸", "右岸", "平原", "草原", "雪山", "峡湾", "博物馆", "纪念馆", "美术馆",
	"山", "河", "湖", "海", "岛", "湾", "峡", "谷", "洞", "泉",
	"寺", "庙", "宫", "城", "镇", "村", "街", "巷", "路", "道", "桥", "塔", "馆",
}

func looksLikeLocationOrEntity(term string) bool {
	for _, marker := range ragEntityMarkers {
		if strings.Contains(term, marker) {
			return true
		}
	}
	return false
}

func looksTooAbstract(term string) bool {
	for _, marker := range []string{
		"温柔", "克制", "安静", "宁静", "安宁", "孤独", "自由", "诗意", "浪漫",
		"写作", "阅读", "观察", "生活", "历史感", "人文感",
	} {
		if strings.Contains(term, marker) {
			return true
		}
	}
	return false
}

func requirementQueryValue(item model.RequirementItem) string {
	if selected := strings.TrimSpace(item.SelectedOption); selected != "" {
		return selected
	}
	return strings.TrimSpace(item.Content)
}

func normalizeQuery(raw string) string {
	replacer := strings.NewReplacer("，", " ", "。", " ", "、", " ", "\n", " ", "\t", " ")
	raw = replacer.Replace(strings.TrimSpace(raw))
	return strings.Join(strings.Fields(raw), " ")
}

func expandQueryTerms(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	replacer := strings.NewReplacer(
		"\n", "|",
		"\t", "|",
		"，", "|",
		"。", "|",
		"、", "|",
		"；", "|",
		";", "|",
		"：", "|",
		":", "|",
		"/", "|",
		"（", "|",
		"）", "|",
		"(", "|",
		")", "|",
		"和", "|",
		"与", "|",
		"及", "|",
		"或", "|",
	)
	normalized := replacer.Replace(raw)
	parts := strings.Split(normalized, "|")
	hasSplitMarker := normalized != raw

	out := make([]string, 0, len(parts)+1)
	seen := make(map[string]struct{}, len(parts)+1)

	appendIfValid := func(term string) {
		for _, candidate := range deriveEntityCandidates(term) {
			candidate = normalizeQuery(candidate)
			if candidate == "" {
				continue
			}
			if utf8.RuneCountInString(candidate) < 2 {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			out = append(out, candidate)
		}
	}

	if !hasSplitMarker {
		appendIfValid(raw)
	}
	for _, part := range parts {
		appendIfValid(part)
	}

	return out
}

func deriveEntityCandidates(raw string) []string {
	raw = normalizeQuery(raw)
	if raw == "" {
		return nil
	}

	trimmedPrefix := trimDecorativePrefix(raw)
	trimmedSuffix := trimDecorativeSuffix(raw)
	trimmedBoth := trimDecorativeSuffix(trimmedPrefix)

	if entityPhrase := extractEntityPhrase(raw); entityPhrase != "" {
		return []string{entityPhrase}
	}

	candidates := make([]string, 0, 4)
	candidates = append(candidates, trimmedBoth, trimmedSuffix, trimmedPrefix, raw)

	out := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func trimDecorativePrefix(raw string) string {
	raw = strings.TrimSpace(raw)
	for _, prefix := range []string{
		"想去", "想看", "想找", "去", "到", "在", "逛", "走进", "走到", "寻找", "看看",
	} {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	}
	return raw
}

func trimDecorativeSuffix(raw string) string {
	raw = strings.TrimSpace(raw)
	for _, suffix := range []string{
		"的秋天", "的春天", "的夏天", "的冬天",
		"的清晨", "的早晨", "的黄昏", "的夜晚",
		"的写作气息", "的阅读气息", "的文学气息", "的人文气息",
		"附近", "一带",
		"秋天", "春天", "夏天", "冬天",
		"清晨", "早晨", "黄昏", "夜晚",
		"散步", "漫步", "徒步", "拍照",
	} {
		if strings.HasSuffix(raw, suffix) {
			raw = strings.TrimSpace(strings.TrimSuffix(raw, suffix))
		}
	}

	if head, tail, ok := strings.Cut(raw, "的"); ok && looksTooAbstract(tail) {
		return strings.TrimSpace(head)
	}
	return raw
}

func extractEntityPhrase(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	raw = trimDecorativeSuffix(trimDecorativePrefix(raw))
	for _, marker := range ragEntityMarkers {
		if !strings.Contains(raw, marker) {
			continue
		}

		idx := strings.Index(raw, marker)
		end := idx + len(marker)
		start := idx

		for start > 0 {
			r, size := utf8.DecodeLastRuneInString(raw[:start])
			if isEntityBoundaryRune(r) {
				break
			}
			start -= size
		}

		phrase := strings.TrimSpace(raw[start:end])
		if utf8.RuneCountInString(phrase) >= 2 {
			return phrase
		}
	}

	return ""
}

func isEntityBoundaryRune(r rune) bool {
	switch r {
	case ' ', ',', '，', '。', '、', ';', '；', ':', '：', '/', '(', ')', '（', '）':
		return true
	default:
		return false
	}
}

func formatRAGReferenceText(documents []RAGDocument, maxChars int) string {
	if len(documents) == 0 || maxChars <= 0 {
		return ""
	}

	var builder strings.Builder
	for _, doc := range documents {
		if strings.TrimSpace(doc.Summary) == "" {
			continue
		}

		segment := fmt.Sprintf("词条：%s\n摘要：%s", doc.Title, doc.Summary)
		if source := strings.TrimSpace(doc.Source); source != "" {
			segment += "\n来源：" + source
		}
		segment += "\n\n"

		if builder.Len()+len([]rune(segment)) > maxChars {
			break
		}
		builder.WriteString(segment)
	}
	return strings.TrimSpace(builder.String())
}

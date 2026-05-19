package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"
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

type RAGLookup struct {
	Query   string
	Title   string
	Summary string
	Source  string
	Hit     bool
}

type RAGContext struct {
	Query          string
	Lookups        []RAGLookup
	Documents      []RAGDocument
	ReferenceText  string
	QueryLatencyMs int64
	WikiLatencyMs  int64
}

type RAGAgent interface {
	BuildContext(ctx context.Context, session *model.InspirationSession) (*RAGContext, error)
}

type ragAgent struct {
	knowledge      wikipediaKnowledgeGetter
	queryExtractor RAGQueryAgent
}

type wikipediaKnowledgeGetter interface {
	GetDefinition(ctx context.Context, keyword string) (*model.WikiDefinition, error)
}

func NewRAGAgent(knowledge wikipediaKnowledgeGetter, queryExtractor RAGQueryAgent) RAGAgent {
	return &ragAgent{knowledge: knowledge, queryExtractor: queryExtractor}
}

func (a *ragAgent) BuildContext(ctx context.Context, session *model.InspirationSession) (*RAGContext, error) {
	if a == nil || a.knowledge == nil || session == nil {
		return nil, nil
	}

	current, ok := session.CurrentRequirement()
	if !ok {
		return nil, nil
	}

	queryStarted := time.Now()
	queries := a.buildQueries(ctx, current)
	queryLatencyMs := time.Since(queryStarted).Milliseconds()
	if len(queries) == 0 {
		return &RAGContext{
			QueryLatencyMs: queryLatencyMs,
			WikiLatencyMs:  0,
		}, nil
	}

	ragCfg := config.Global().RAG
	lookups := make([]RAGLookup, 0, len(queries))
	documents := make([]RAGDocument, 0, ragCfg.MaxWikiDocs)
	seenTitles := make(map[string]struct{}, ragCfg.MaxWikiDocs)
	var totalWikiLatencyMs int64

	for i, query := range queries {
		wikiStarted := time.Now()
		definition, err := a.knowledge.GetDefinition(ctx, query)
		wikiLatencyMs := time.Since(wikiStarted).Milliseconds()
		if err != nil || definition == nil {
			totalWikiLatencyMs += wikiLatencyMs
			lookups = append(lookups, RAGLookup{Query: query})
			continue
		}
		totalWikiLatencyMs += wikiLatencyMs

		title := strings.TrimSpace(definition.Title)
		if title == "" {
			title = query
		}
		lookups = append(lookups, RAGLookup{
			Query:   query,
			Title:   title,
			Summary: strings.TrimSpace(definition.Summary),
			Source:  strings.TrimSpace(definition.FullURL),
			Hit:     true,
		})
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
		if len(documents) == ragCfg.MaxWikiDocs {
			break
		}
	}

	if len(documents) == 0 {
		return &RAGContext{
			Query:          strings.Join(queries, " | "),
			Lookups:        lookups,
			Documents:      make([]RAGDocument, 0),
			ReferenceText:  "",
			QueryLatencyMs: queryLatencyMs,
			WikiLatencyMs:  totalWikiLatencyMs,
		}, nil
	}

	return &RAGContext{
		Query:          strings.Join(queries, " | "),
		Lookups:        lookups,
		Documents:      documents,
		ReferenceText:  formatRAGReferenceText(documents, ragCfg.MaxContextChars),
		QueryLatencyMs: queryLatencyMs,
		WikiLatencyMs:  totalWikiLatencyMs,
	}, nil
}

func (a *ragAgent) buildQueries(ctx context.Context, current *model.Inspiration) []string {
	if current == nil {
		return nil
	}

	if a != nil && a.queryExtractor != nil {
		if queries, err := a.queryExtractor.ExtractQueries(ctx, current); err == nil {
			return postProcessRAGQueries(queries)
		}
	}

	return nil
}

func postProcessRAGQueries(candidates []string) []string {
	if len(candidates) == 0 {
		return nil
	}

	out := make([]string, 0, len(candidates)*2)
	seen := make(map[string]struct{}, len(candidates)*2)
	appendCandidate := func(term string) {
		term = normalizeQuery(term)
		if !isUsefulRAGQuery(term) {
			return
		}
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		out = append(out, term)
	}

	for _, raw := range candidates {
		expanded := expandQueryTerms(raw)
		if len(expanded) == 0 {
			expanded = []string{raw}
		}
		for _, term := range expanded {
			derived := make([]string, 0, 4)
			derived = append(derived, deriveKnowledgeCandidates(term)...)
			derived = append(derived, deriveSceneCandidates(term)...)
			if len(derived) == 0 {
				appendCandidate(term)
				continue
			}
			for _, item := range derived {
				appendCandidate(item)
			}
		}
	}

	return out
}

var ragQueryStopwords = map[string]struct{}{
	"风景":   {},
	"景色":   {},
	"地方":   {},
	"城市":   {},
	"小城":   {},
	"古城":   {},
	"古镇":   {},
	"散步":   {},
	"徒步":   {},
	"漫步":   {},
	"发呆":   {},
	"看海":   {},
	"看山":   {},
	"拍照":   {},
	"旅行":   {},
	"旅游":   {},
	"出发":   {},
	"抵达":   {},
	"休息":   {},
	"放空":   {},
	"秋天":   {},
	"春天":   {},
	"夏天":   {},
	"冬天":   {},
	"清晨":   {},
	"早晨":   {},
	"黄昏":   {},
	"夜晚":   {},
	"晚上":   {},
	"白天":   {},
	"白日":   {},
	"夜里":   {},
	"夜里散步": {},
	"午后":   {},
	"傍晚":   {},
	"咖啡馆":  {},
	"旧时光":  {},
	"清冷":   {},
	"书店":   {},
	"旧书店":  {},
	"仓库":   {},
	"仓库群":  {},
	"港口片区": {},
	"写字":   {},
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
	if looksLikeLocationOrEntity(term) || looksLikeKnowledgeConcept(term) {
		return true
	}
	return false
}

var ragEntityMarkers = []string{
	"书院", "故居", "故里", "遗址", "石窟", "古道", "码头", "剧院", "书店",
	"左岸", "右岸", "平原", "草原", "雪山", "峡湾", "博物馆", "纪念馆", "美术馆",
	"运河", "港口", "仓库", "仓库群",
	"山", "河", "湖", "海", "岛", "湾", "峡", "谷", "洞", "泉",
	"寺", "庙", "宫", "城", "镇", "村", "街", "巷", "路", "道", "桥", "塔", "馆",
}

var ragConceptMarkers = []string{
	"文学", "历史", "文化", "文明", "艺术", "宗教", "哲学", "神话",
	"壁画", "飞天", "朝圣", "贸易", "捕捞", "铁道",
}

var ragAdminPlaceSuffixes = []string{
	"省", "市", "县", "区", "州", "国", "岛", "府", "都", "道",
}

func looksLikeLocationOrEntity(term string) bool {
	if looksLikeAdminPlace(term) || looksLikeNamedEntity(term) {
		return true
	}
	for _, marker := range ragEntityMarkers {
		if strings.Contains(term, marker) {
			return true
		}
	}
	return false
}

func looksLikeKnowledgeConcept(term string) bool {
	for _, marker := range ragConceptMarkers {
		if strings.Contains(term, marker) {
			return true
		}
	}
	return false
}

func looksTooAbstract(term string) bool {
	for _, marker := range []string{
		"温柔", "克制", "安静", "宁静", "安宁", "孤独", "自由", "诗意", "浪漫",
		"写作", "阅读", "观察", "生活", "历史感", "人文感", "旧时光", "清冷",
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

	for _, candidate := range []string{trimmedBoth, trimmedPrefix, trimmedSuffix, raw} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" {
			return []string{candidate}
		}
	}
	return nil
}

func trimDecorativePrefix(raw string) string {
	raw = strings.TrimSpace(raw)
	for _, prefix := range []string{
		"想去", "想看", "想找", "去", "到", "在", "逛", "走进", "走到", "寻找", "看看", "看", "找",
	} {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, prefix))
	}
	return raw
}

func trimDecorativeSuffix(raw string) string {
	raw = strings.TrimSpace(raw)
	for {
		changed := false
		for _, suffix := range []string{
			"的秋天", "的春天", "的夏天", "的冬天",
			"的清晨", "的早晨", "的黄昏", "的夜晚",
			"的写作气息", "的阅读气息", "的文学气息", "的人文气息",
			"留下的痕迹", "留下的生活痕迹", "留下的现实痕迹", "的痕迹",
			"附近", "一带",
			"待几天", "住几天",
			"秋天", "春天", "夏天", "冬天",
			"清晨", "早晨", "黄昏", "夜晚",
			"散步", "漫步", "徒步", "拍照", "写字",
		} {
			if strings.HasSuffix(raw, suffix) {
				raw = strings.TrimSpace(strings.TrimSuffix(raw, suffix))
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	if head, tail, ok := strings.Cut(raw, "的"); ok && looksTooAbstract(tail) {
		return strings.TrimSpace(head)
	}
	return raw
}

func buildKnowledgeQueryTerms(raw string) []string {
	base := expandQueryTerms(raw)
	out := make([]string, 0, len(base)*2)
	seen := make(map[string]struct{}, len(base)*2)
	for _, term := range base {
		for _, candidate := range deriveKnowledgeCandidates(term) {
			candidate = normalizeQuery(candidate)
			if candidate == "" {
				continue
			}
			if _, ok := seen[candidate]; ok {
				continue
			}
			seen[candidate] = struct{}{}
			out = append(out, candidate)
		}
	}
	return out
}

func deriveSceneCandidates(term string) []string {
	term = normalizeQuery(term)
	if left, right, ok := splitAdjacentNamedEntities(term); ok {
		return []string{left, right}
	}
	return deriveAtomicEntityCandidates(term)
}

func deriveKnowledgeCandidates(term string) []string {
	term = normalizeQuery(term)
	if term == "" {
		return nil
	}

	out := make([]string, 0, 3)
	appendUnique := func(value string) {
		value = normalizeQuery(value)
		if value == "" {
			return
		}
		for _, existing := range out {
			if existing == value {
				return
			}
		}
		out = append(out, value)
	}

	if looksLikeKnowledgeConcept(term) {
		appendUnique(term)
	}
	for _, candidate := range deriveAtomicEntityCandidates(term) {
		appendUnique(candidate)
	}

	return out
}

func deriveAtomicEntityCandidates(term string) []string {
	term = normalizeQuery(term)
	if term == "" {
		return nil
	}

	out := make([]string, 0, 4)
	appendUnique := func(value string) {
		value = normalizeQuery(value)
		if value == "" {
			return
		}
		for _, existing := range out {
			if existing == value {
				return
			}
		}
		out = append(out, value)
	}

	var addAtomic func(string)
	addAtomic = func(value string) {
		value = normalizeQuery(value)
		if value == "" {
			return
		}

		if broad, specific := splitLeadingAdminPlace(value); broad != "" {
			appendUnique(broad)
			addAtomic(specific)
			return
		}

		if related := relatedEntityTerms(value); len(related) > 0 {
			for _, item := range related {
				appendUnique(item)
			}
			return
		}

		if locality := localityPrefix(value); locality != "" {
			appendUnique(locality)
			return
		}

		if entity := extractEntityPhrase(value); entity != "" && entity != value {
			addAtomic(entity)
			return
		}

		if looksLikeLocationOrEntity(value) || looksLikeKnowledgeConcept(value) {
			appendUnique(value)
		}
	}

	addAtomic(term)
	return out
}

func splitLeadingAdminPlace(term string) (prefix string, rest string) {
	term = strings.TrimSpace(term)
	for _, suffix := range ragAdminPlaceSuffixes {
		idx := strings.Index(term, suffix)
		if idx <= 0 {
			continue
		}
		end := idx + len(suffix)
		if utf8.RuneCountInString(term[end:]) < 2 {
			continue
		}
		return strings.TrimSpace(term[:end]), strings.TrimSpace(term[end:])
	}
	return "", ""
}

func localityPrefix(term string) string {
	term = strings.TrimSpace(term)
	if term == "" {
		return ""
	}
	for _, marker := range ragEntityMarkers {
		if utf8.RuneCountInString(marker) < 2 {
			continue
		}
		idx := strings.Index(term, marker)
		if idx <= 0 {
			continue
		}
		prefix := strings.TrimSpace(term[:idx])
		if utf8.RuneCountInString(prefix) >= 2 && !strings.Contains(prefix, "的") && !strings.Contains(prefix, "之") && !looksTooAbstract(prefix) {
			return prefix
		}
	}
	return ""
}

func relatedEntityTerms(term string) []string {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil
	}
	for _, marker := range []string{"故里", "故居", "旧居", "纪念馆"} {
		if !strings.HasSuffix(term, marker) {
			continue
		}
		name := strings.TrimSpace(strings.TrimSuffix(term, marker))
		if left, right, ok := splitAdjacentNamedEntities(name); ok {
			return []string{left, right}
		}
		if looksLikeNamedEntity(name) {
			return []string{name}
		}
	}
	return nil
}

func splitAdjacentNamedEntities(term string) (left string, right string, ok bool) {
	runes := []rune(strings.TrimSpace(term))
	if len(runes) < 4 || len(runes) > 7 {
		return "", "", false
	}

	for split := 2; split <= len(runes)-2; split++ {
		lhs := strings.TrimSpace(string(runes[:split]))
		rhs := strings.TrimSpace(string(runes[split:]))
		if !looksLikeNamedEntity(lhs) || !looksLikeNamedEntity(rhs) {
			continue
		}
		if utf8.RuneCountInString(rhs) < utf8.RuneCountInString(lhs) {
			continue
		}
		return lhs, rhs, true
	}
	return "", "", false
}

func looksLikeAdminPlace(term string) bool {
	for _, suffix := range ragAdminPlaceSuffixes {
		if strings.HasSuffix(term, suffix) && utf8.RuneCountInString(term) >= 2 {
			return true
		}
	}
	return false
}

func looksLikeNamedEntity(term string) bool {
	if utf8.RuneCountInString(term) < 2 || utf8.RuneCountInString(term) > 6 {
		return false
	}
	if strings.Contains(term, "的") || strings.Contains(term, "之") {
		return false
	}
	if looksTooAbstract(term) {
		return false
	}
	if _, blocked := ragQueryStopwords[term]; blocked {
		return false
	}
	for _, marker := range ragEntityMarkers {
		if strings.Contains(term, marker) {
			return false
		}
	}
	for _, r := range term {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	return true
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

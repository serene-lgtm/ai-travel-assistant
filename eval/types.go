package eval

import "time"

type Category string

const (
	CategoryIntent        Category = "intent"
	CategoryClarification Category = "clarification"
	CategoryInspiration   Category = "inspiration"
)

type FixtureInput struct {
	Content string `json:"content"`
}

type FixtureRequirementState struct {
	Content string `json:"content"`
	Score   int    `json:"score"`
}

type FixtureCurrentRequirement struct {
	Mood            FixtureRequirementState `json:"mood"`
	Scene           FixtureRequirementState `json:"scene"`
	Focus           FixtureRequirementState `json:"focus"`
	SceneCandidates []string                `json:"scene_candidates"`
}

type FixtureSessionState struct {
	Status         string                    `json:"status"`
	TargetField    string                    `json:"target_field"`
	CurrentRequest FixtureCurrentRequirement `json:"current_requirement"`
}

type FixtureExpected struct {
	TravelRelated           *bool    `json:"travel_related,omitempty"`
	TargetField             string   `json:"target_field,omitempty"`
	OptionCountMin          int      `json:"option_count_min,omitempty"`
	OptionCountMax          int      `json:"option_count_max,omitempty"`
	ContentMustInclude      []string `json:"content_must_include,omitempty"`
	ContentMustAvoid        []string `json:"content_must_avoid,omitempty"`
	SingleCorePlaceRequired bool     `json:"single_core_place_required,omitempty"`
}

type PromptEvalCase struct {
	ID           string               `json:"id"`
	Category     Category             `json:"category"`
	Description  string               `json:"description"`
	Tags         []string             `json:"tags,omitempty"`
	Input        FixtureInput         `json:"input"`
	SessionState *FixtureSessionState `json:"session_state,omitempty"`
	Expected     FixtureExpected      `json:"expected"`
	Rubric       []string             `json:"rubric"`
}

type IntentActual struct {
	TravelRelated bool   `json:"travel_related"`
	RawResponse   string `json:"raw_response,omitempty"`
}

type ClarificationActual struct {
	Question    string   `json:"question,omitempty"`
	Options     []string `json:"options,omitempty"`
	TargetField string   `json:"target_field,omitempty"`
}

type InspirationActual struct {
	Content string `json:"content,omitempty"`
}

type PromptEvalTrace struct {
	RAGEnabled          bool              `json:"rag_enabled"`
	RAGQuery            string            `json:"rag_query,omitempty"`
	RAGLookups          []EvalRAGLookup   `json:"rag_lookups,omitempty"`
	RAGDocumentTitles   []string          `json:"rag_document_titles,omitempty"`
	RAGDocuments        []EvalRAGDocument `json:"rag_documents,omitempty"`
	RAGReferenceText    string            `json:"rag_reference_text,omitempty"`
	Prompt              string            `json:"prompt,omitempty"`
	TotalLatencyMs      int64             `json:"total_latency_ms,omitempty"`
	QueryLatencyMs      int64             `json:"query_latency_ms,omitempty"`
	WikiLatencyMs       int64             `json:"wiki_latency_ms,omitempty"`
	GenerationLatencyMs int64             `json:"generation_latency_ms,omitempty"`
}

type EvalRAGLookup struct {
	Query   string `json:"query,omitempty"`
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
	Source  string `json:"source,omitempty"`
	Hit     bool   `json:"hit"`
}

type EvalRAGDocument struct {
	Title   string  `json:"title,omitempty"`
	Summary string  `json:"summary,omitempty"`
	Source  string  `json:"source,omitempty"`
	Query   string  `json:"query,omitempty"`
	Score   float64 `json:"score,omitempty"`
}

type Assertion struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Details string `json:"details,omitempty"`
}

type PromptEvalResult struct {
	CaseID      string           `json:"case_id"`
	Category    Category         `json:"category"`
	Description string           `json:"description"`
	Tags        []string         `json:"tags,omitempty"`
	Expected    FixtureExpected  `json:"expected"`
	Actual      any              `json:"actual,omitempty"`
	Assertions  []Assertion      `json:"assertions,omitempty"`
	Passed      bool             `json:"passed"`
	Error       string           `json:"error,omitempty"`
	Trace       *PromptEvalTrace `json:"trace,omitempty"`
	ManualScore map[string]any   `json:"manual_score,omitempty"`
	ManualNotes string           `json:"manual_notes,omitempty"`
}

type PromptEvalRun struct {
	RunID         string             `json:"run_id"`
	Label         string             `json:"label,omitempty"`
	Timestamp     time.Time          `json:"timestamp"`
	Model         string             `json:"model"`
	BaseURL       string             `json:"base_url"`
	GitCommit     string             `json:"git_commit,omitempty"`
	SelectedCases []string           `json:"selected_cases,omitempty"`
	TotalCases    int                `json:"total_cases"`
	Results       []PromptEvalResult `json:"results"`
}

type InspirationABItem struct {
	CaseID                      string            `json:"case_id,omitempty"`
	Description                 string            `json:"description,omitempty"`
	Input                       string            `json:"input,omitempty"`
	Query                       string            `json:"query,omitempty"`
	RAGLookups                  []EvalRAGLookup   `json:"rag_lookups,omitempty"`
	RAGDocuments                []EvalRAGDocument `json:"rag_documents,omitempty"`
	RAGContext                  string            `json:"rag_context,omitempty"`
	BaselineOutput              string            `json:"baseline_output,omitempty"`
	RAGOutput                   string            `json:"rag_output,omitempty"`
	BaselineError               string            `json:"baseline_error,omitempty"`
	RAGError                    string            `json:"rag_error,omitempty"`
	BaselineLatencyMs           int64             `json:"baseline_latency_ms,omitempty"`
	RAGLatencyMs                int64             `json:"rag_latency_ms,omitempty"`
	BaselineGenerationLatencyMs int64             `json:"baseline_generation_latency_ms,omitempty"`
	RAGQueryLatencyMs           int64             `json:"rag_query_latency_ms,omitempty"`
	RAGWikiLatencyMs            int64             `json:"rag_wiki_latency_ms,omitempty"`
	RAGGenerationLatencyMs      int64             `json:"rag_generation_latency_ms,omitempty"`
}

type InspirationABRun struct {
	Label string              `json:"label,omitempty"`
	Items []InspirationABItem `json:"items"`
}

type CompareCaseChange struct {
	CaseID      string   `json:"case_id"`
	Category    Category `json:"category"`
	Description string   `json:"description"`
	Reason      string   `json:"reason"`
}

type CompareSummary struct {
	BaselinePath  string              `json:"baseline_path"`
	CandidatePath string              `json:"candidate_path"`
	TotalCases    int                 `json:"total_cases"`
	ScoredCases   int                 `json:"scored_cases"`
	Regressions   []CompareCaseChange `json:"regressions,omitempty"`
	Improvements  []CompareCaseChange `json:"improvements,omitempty"`
	Unscored      []string            `json:"unscored,omitempty"`
}

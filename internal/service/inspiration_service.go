package service

import (
	"ai-reading-assistant/internal/agent"
	"ai-reading-assistant/internal/config"
	"ai-reading-assistant/internal/dao"
	"ai-reading-assistant/internal/dto/http_dto"
	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
	"ai-reading-assistant/internal/orchestrator"
	"ai-reading-assistant/internal/wikipedia"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const max_token = 32000
const default_max_message_reserved = 15

type InspirationService interface {
	CreateInspirationSession(userID string) (string, error)
	GetInspirationSession(id string) (*model.InspirationSession, error)
	ChatCompletion(request *http_dto.InspirationMessageRequest) (*http_dto.InspirationMessageResponse, error)
	FavoriteInspiration(sessionID, inspirationID string) error
	UnfavoriteInspiration(sessionID, inspirationID string) error
	GetRequestProgress(sessionID string) (*model.RequestProcess, error)
}

type inspirationServiceImpl struct {
	llmClient    *llm.DeepseekClient
	intentAgent  agent.IntentAgent
	sessionDao   dao.InspirationSessionDao
	messageDao   dao.InspirationMessageDao
	processDao   dao.RequestProcessDao
	orchestrator orchestrator.InspirationOrchestrator
}

func NewInspirationService(client *llm.DeepseekClient, sessionDao dao.InspirationSessionDao, messageDao dao.InspirationMessageDao, processDao dao.RequestProcessDao) InspirationService {
	intentAgent := agent.NewIntentAgent(client)
	requirementAgent := agent.NewRequirementAnalyzerAgent(client)
	clarificationAgent := agent.NewClarificationAgent(client)
	inspirationAgent := agent.NewInspirationAgent(client)
	keywordAgent := agent.NewKeywordAgent(client)
	wikipediaAgent := newWikipediaAgentFromConfig(config.Global().Wikipedia)
	ragAgent := agent.NewRAGAgent(wikipediaAgent)
	return &inspirationServiceImpl{
		llmClient:   client,
		intentAgent: intentAgent,
		sessionDao:  sessionDao,
		messageDao:  messageDao,
		processDao:  processDao,
		orchestrator: orchestrator.NewInspirationOrchestrator(
			sessionDao,
			messageDao,
			processDao,
			intentAgent,
			requirementAgent,
			clarificationAgent,
			inspirationAgent,
			keywordAgent,
			wikipediaAgent,
			ragAgent,
		),
	}
}

func newWikipediaAgentFromConfig(cfg config.WikipediaConfig) agent.WikipediaAgent {
	opts := make([]wikipedia.Option, 0, 3)
	if lang := strings.TrimSpace(cfg.Language); lang != "" {
		opts = append(opts, wikipedia.WithLanguage(lang))
	}
	if proxy := strings.TrimSpace(cfg.Proxy); proxy != "" {
		opts = append(opts, wikipedia.WithProxy(proxy))
	}
	if userAgent := strings.TrimSpace(cfg.UserAgent); userAgent != "" {
		opts = append(opts, wikipedia.WithUserAgent(userAgent))
	}

	client, err := wikipedia.NewClient(opts...)
	if err != nil {
		return nil
	}
	return agent.NewWikipediaAgent(client)
}

// CreateInspirationSession creates and returns a brand new session ID.
func (s *inspirationServiceImpl) CreateInspirationSession(userID string) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("user id is required")
	}
	ctx := context.Background()
	session := &model.InspirationSession{
		MaxToken:     max_token,
		UserID:       strings.TrimSpace(userID),
		Messages:     make([]string, 0),
		Inspirations: make([]model.Inspiration, 0),
		CreatedAt:    time.Now().UTC(),
		Status:       model.SessionStatusCreated,
	}
	created, err := s.sessionDao.Create(ctx, session)
	if err != nil {
		return "", fmt.Errorf("create inspiration session %w", err)
	}
	return created.ID, nil
}

// GetInspirationSession gets session by ID
func (s *inspirationServiceImpl) GetInspirationSession(id string) (*model.InspirationSession, error) {
	if id == "" {
		return nil, fmt.Errorf("session id is required")
	}

	ctx := context.Background()
	session, err := s.sessionDao.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get inspiration session %w", err)
	}
	return session, nil
}

func (s *inspirationServiceImpl) FavoriteInspiration(sessionID, inspirationID string) error {
	return s.markInspirationFavorite(sessionID, inspirationID, true)
}

func (s *inspirationServiceImpl) UnfavoriteInspiration(sessionID, inspirationID string) error {
	return s.markInspirationFavorite(sessionID, inspirationID, false)
}

func (s *inspirationServiceImpl) markInspirationFavorite(sessionID, inspirationID string, isFavorite bool) error {
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(inspirationID) == "" {
		return fmt.Errorf("inspiration id is required")
	}

	ctx := context.Background()
	session, err := s.sessionDao.GetByID(ctx, strings.TrimSpace(sessionID))
	if err != nil {
		return fmt.Errorf("load inspiration session %w", err)
	}

	found := false
	for i := range session.Inspirations {
		if session.Inspirations[i].ID != strings.TrimSpace(inspirationID) {
			continue
		}
		session.Inspirations[i].IsFavorite = isFavorite
		found = true
		break
	}
	if !found {
		return fmt.Errorf("inspiration %s not found in session %s", inspirationID, sessionID)
	}

	if _, err := s.sessionDao.Update(ctx, session); err != nil {
		return fmt.Errorf("update inspiration session %w", err)
	}
	return nil
}

// GetResponse returns a response to user message
func (s *inspirationServiceImpl) ChatCompletion(request *http_dto.InspirationMessageRequest) (*http_dto.InspirationMessageResponse, error) {
	if s.orchestrator == nil {
		return nil, fmt.Errorf("inspiration orchestrator is not initialized")
	}
	return s.orchestrator.HandleChatCompletion(context.Background(), request)
}

// generateInspiration composes a literary travel inspiration based on the session requirements.
func (s *inspirationServiceImpl) generateInspiration(session *model.InspirationSession) (string, error) {
	if s.llmClient == nil {
		return "", fmt.Errorf("llm client is not initialized")
	}
	if session == nil {
		return "", fmt.Errorf("session is nil")
	}
	if len(session.Inspirations) == 0 {
		return "", fmt.Errorf("inspirations is empty")
	}

	// to collection information from both user input and user choice1
	current, ok := session.CurrentRequirement()
	if !ok {
		return "", fmt.Errorf("current requirement is empty")
	}

	getContent := func(field model.RequirementField) string {
		item := current.Get(field)
		content := strings.TrimSpace(item.Content)
		if strings.TrimSpace(item.SelectedOption) != "" {
			if content == "" {
				content = item.SelectedOption
			} else {
				content = content + "\n" + strings.TrimSpace(item.SelectedOption)
			}
		}
		return content
	}

	mood := getContent(model.RequirementFieldMood)
	scene := getContent(model.RequirementFieldScene)
	focus := getContent(model.RequirementFieldFocus)

	prompt := fmt.Sprintf(genInspirationPrompt, mood, scene, focus)

	resp, err := s.llmClient.Call(prompt)
	if err != nil {
		return "", fmt.Errorf("generate inspiration: %w", err)
	}
	return strings.TrimSpace(resp), nil
}

func defaultIfEmpty(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

// buildConversationContext collects the latest session messages and current input for LLM calls.
func (s *inspirationServiceImpl) buildConversationContext(ctx context.Context, session *model.InspirationSession, latestContent string) (string, error) {
	if session == nil {
		return "", fmt.Errorf("session is nil")
	}

	sessionCfg := config.Global().Session
	maxReserved := sessionCfg.MaxMessageReserved
	if maxReserved <= 0 {
		maxReserved = default_max_message_reserved
	}

	numHistory := len(session.Messages)
	numFromHistory := maxReserved - 1
	if numFromHistory > numHistory {
		numFromHistory = numHistory
	}

	var historyIDs []string
	if numFromHistory > 0 {
		historyIDs = session.Messages[numHistory-numFromHistory:]
	}

	var mergedContents []string
	if len(historyIDs) > 0 {
		historyMsgs, err := s.messageDao.GetByIDs(ctx, historyIDs)
		if err != nil {
			return "", fmt.Errorf("load inspiration messages %w", err)
		}
		for _, msg := range historyMsgs {
			if msg == nil {
				continue
			}
			content := strings.TrimSpace(msg.Content)
			if content == "" {
				continue
			}
			mergedContents = append(mergedContents, content)
		}
	}

	if trimmed := strings.TrimSpace(latestContent); trimmed != "" {
		mergedContents = append(mergedContents, trimmed)
	}
	if len(mergedContents) == 0 {
		return "", fmt.Errorf("conversation context is empty")
	}
	return strings.Join(mergedContents, "\n\n"), nil
}

func decodeFirstJSONObject(raw string, dst any) error {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || start > end {
		return fmt.Errorf("response missing json object: %s", raw)
	}
	return json.Unmarshal([]byte(raw[start:end+1]), dst)
}

func (s *inspirationServiceImpl) isTravelRelated(content string) (bool, error) {
	if strings.TrimSpace(content) == "" {
		return false, fmt.Errorf("content is empty")
	}
	if s.intentAgent == nil {
		return false, fmt.Errorf("intent agent is not initialized")
	}

	result, err := s.intentAgent.DetectTravelIntent(context.Background(), content)
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, fmt.Errorf("intent agent returned nil result")
	}
	return result.TravelRelated, nil
}

var fieldLabels = map[model.RequirementField]string{
	model.RequirementFieldMood:  "情感基调",
	model.RequirementFieldScene: "旅行场景",
	model.RequirementFieldFocus: "核心焦点",
}

var fieldGuidance = map[model.RequirementField]string{
	model.RequirementFieldMood:  "描述希望在旅行中体验到的情绪、哲思或文学意象,例如“温柔但克制的安静”。",
	model.RequirementFieldScene: "明确旅行发生的地点或环境特征,尽量具体到城市、街区或自然地貌。",
	model.RequirementFieldFocus: "说明旅行中的具体行动或感官体验,例如“在清晨徒步”“寻找作家的老宅”等。",
}

func (s *inspirationServiceImpl) generateClarifyQuestion(field model.RequirementField, session *model.InspirationSession) (string, []model.Option, error) {
	label := fieldLabels[field]
	if label == "" {
		return "", nil, fmt.Errorf("unknown field %s", field)
	}
	guidance := fieldGuidance[field]
	summary := summarizeRequirementState(session)

	prompt := fmt.Sprintf(`你是一位文学旅行策划师,需要继续了解用户需求。
已掌握的信息:
%s

目标: 围绕[%s]提出问题,帮助用户澄清: %s
输出要求:
1. 只返回 JSON {"question":"...", "options":["...","..."]}.
2. options 列表包含2-4个不超过20字的候选,具体且互相区分。
3. 问题要温柔、鼓励式,针对用户视角描述。
4. 不允许自由输入,请选择可执行的选项。
`, summary, label, guidance)

	raw, err := s.llmClient.Call(prompt)
	if err != nil {
		return "", nil, fmt.Errorf("clarify question: %w", err)
	}

	var payload struct {
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}
	if err := decodeFirstJSONObject(raw, &payload); err != nil {
		return "", nil, fmt.Errorf("clarify parse: %w", err)
	}

	question := strings.TrimSpace(payload.Question)
	if question == "" {
		return "", nil, fmt.Errorf("clarify question empty")
	}

	opts := make([]model.Option, 0, len(payload.Options))
	for _, opt := range payload.Options {
		opt = strings.TrimSpace(opt)
		if opt == "" {
			continue
		}
		opts = append(opts, model.Option{Content: opt})
		if len(opts) == 4 {
			break
		}
	}
	if len(opts) < 2 {
		return "", nil, fmt.Errorf("clarify options insufficient")
	}
	return question, opts, nil
}

func summarizeRequirementState(session *model.InspirationSession) string {
	if session == nil {
		return ""
	}
	current, ok := session.CurrentRequirement()
	if !ok {
		return ""
	}
	var lines []string
	for _, field := range requirementFieldOrder {
		item := current.Get(field)
		content := defaultIfEmpty(item.Content, "暂未提供")
		if strings.TrimSpace(item.SelectedOption) != "" {
			content = fmt.Sprintf("%s\n选择: %s", content, strings.TrimSpace(item.SelectedOption))
		}
		lines = append(lines, fmt.Sprintf("%s: %s (score=%d)", fieldLabels[field], content, item.Score))
	}
	return strings.Join(lines, "\n")
}

func toResponseOptions(opts []model.Option) []http_dto.ResponseOption {
	if len(opts) == 0 {
		return nil
	}
	out := make([]http_dto.ResponseOption, 0, len(opts))
	for _, opt := range opts {
		out = append(out, http_dto.ResponseOption{
			Content:  opt.Content,
			Selected: opt.Selected,
		})
	}
	return out
}

func toResponseInspiration(inspiration *model.Inspiration) *http_dto.InspirationPayload {
	if inspiration == nil {
		return nil
	}

	return &http_dto.InspirationPayload{
		ID:         inspiration.ID,
		Output:     inspiration.Output,
		Keywords:   toResponseKeywords(inspiration.KeyWords),
		IsFavorite: inspiration.IsFavorite,
	}
}

func toResponseKeywords(items []model.KeyWord) []http_dto.KeywordPayload {
	if len(items) == 0 {
		return nil
	}
	out := make([]http_dto.KeywordPayload, 0, len(items))
	for _, item := range items {
		out = append(out, http_dto.KeywordPayload{
			Content:        item.Content,
			Start:          item.Start,
			End:            item.End,
			WikiDefinition: toResponseWikiDefinition(item.WikiDefinition),
		})
	}
	return out
}

func toResponseWikiDefinition(item *model.WikiDefinition) *http_dto.WikiDefinitionPayload {
	if item == nil {
		return nil
	}
	return &http_dto.WikiDefinitionPayload{
		Title:   item.Title,
		Summary: item.Summary,
		FullURL: item.FullURL,
	}
}

// GetRequestProgress retrieves the current progress of a request being processed
func (s *inspirationServiceImpl) GetRequestProgress(sessionID string) (*model.RequestProcess, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}

	ctx := context.Background()
	process, err := s.processDao.GetBySessionID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get request process: %w", err)
	}
	return process, nil
}

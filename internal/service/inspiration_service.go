package service

import (
	"ai-reading-assistant/internal/config"
	"ai-reading-assistant/internal/dao"
	"ai-reading-assistant/internal/dto/http_dto"
	"ai-reading-assistant/internal/llm"
	"ai-reading-assistant/internal/model"
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
	llmClient  *llm.DeepseekClient
	sessionDao dao.InspirationSessionDao
	messageDao dao.InspirationMessageDao
	processDao dao.RequestProcessDao
}

func NewInspirationService(client *llm.DeepseekClient, sessionDao dao.InspirationSessionDao, messageDao dao.InspirationMessageDao, processDao dao.RequestProcessDao) InspirationService {
	return &inspirationServiceImpl{
		llmClient:  client,
		sessionDao: sessionDao,
		messageDao: messageDao,
		processDao: processDao,
	}
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
	ctx := context.Background()

	// Get user ID from session for tracking
	var userID string
	if session, err := s.sessionDao.GetByID(ctx, request.SessionID); err == nil && session != nil {
		userID = session.UserID
	}

	// Create request process tracking record
	requestProcess := &model.RequestProcess{
		SessionID: request.SessionID,
		UserID:    userID,
		Stage:     model.RequestStageDecoding,
		StartedAt: time.Now().UTC(),
	}

	savedProcess, err := s.processDao.Create(ctx, requestProcess)
	if err != nil {
		return nil, fmt.Errorf("create request process: %w", err)
	}
	requestProcess = savedProcess

	// Process the message and track progress through different stages
	defer func() {
		if err != nil {
			requestProcess.Stage = model.RequestStageFailed
			requestProcess.Error = err.Error()
		}
		requestProcess.CompletedAt = time.Now().UTC()
		s.processDao.Update(ctx, requestProcess)
	}()

	// Stage: Decoding - validate and decode message
	inspirationMsg, err := request.ToModel()
	if err != nil {
		err = fmt.Errorf("create inspiration message %w", err)
		return nil, err
	}
	if inspirationMsg.Kind == "" {
		err = fmt.Errorf("empty message kind")
		return nil, err
	}
	if inspirationMsg.Kind == model.MessageKindUserInput {
		if ok, err := s.isTravelRelated(inspirationMsg.Content); err != nil {
			err = fmt.Errorf("classify travel intent %w", err)
			return nil, err
		} else if !ok {
			err = fmt.Errorf("当前输入未识别为旅行请求,请更具体描述你的旅行意图")
			return nil, err
		}
	}

	session, err := s.sessionDao.GetByID(ctx, inspirationMsg.SessionID)
	if err != nil {
		err = fmt.Errorf("load inspiration session %w", err)
		return nil, err
	}
	session.EnsureRequirementInitialized()
	if inspirationMsg.StartNewInspiration {
		if session.AppendRequirement() == nil {
			err = fmt.Errorf("append requirement failed")
			return nil, err
		}
		session.Status = model.SessionStatusStartOver
	}

	// Stage: Analyzing - extract and analyze requirements
	requestProcess.Stage = model.RequestStageAnalyzing
	if _, err = s.processDao.Update(ctx, requestProcess); err != nil {
		err = fmt.Errorf("update process to analyzing: %w", err)
		return nil, err
	}

	if err := s.analyzeRequirement(inspirationMsg, session); err != nil {
		err = fmt.Errorf("anaylyze requirement %w", err)
		return nil, err
	}

	userMsg, err := s.messageDao.Create(ctx, inspirationMsg)
	if err != nil {
		err = fmt.Errorf("persist user inspiration message %w", err)
		return nil, err
	}
	session.Messages = append(session.Messages, userMsg.ID)

	var (
		assistantMsg model.InspirationMessage
		response     http_dto.InspirationMessageResponse
	)

	// Stage: Generating - generate inspiration or clarify questions
	requestProcess.Stage = model.RequestStageGenerating
	if _, err = s.processDao.Update(ctx, requestProcess); err != nil {
		err = fmt.Errorf("update process to generating: %w", err)
		return nil, err
	}

	if session.IsReadyToGenerate() {
		replyContent, err := s.generateInspiration(session)
		if err != nil {
			err = fmt.Errorf("generate inspiration %w", err)
			return nil, err
		}
		if current, ok := session.CurrentRequirement(); ok {
			current.Output = replyContent
		}
		assistantMsg = model.InspirationMessage{
			SessionID: inspirationMsg.SessionID,
			Role:      model.InspirationMessageRoleAssistant,
			Kind:      model.MessageKindAssistant,
			Content:   replyContent,
			CreatedAt: time.Now().UTC(),
		}
		response = http_dto.InspirationMessageResponse{
			SessionID: inspirationMsg.SessionID,
			Role:      string(model.InspirationMessageRoleAssistant),
			Kind:      string(model.MessageKindAssistant),
			Content:   replyContent,
		}
		session.Status = model.SessionStatusCompleted
	} else {
		// must have a field need to be clarified here, or it would be ready to generate
		field, err := s.pickNextField(session)
		if err != nil {
			err = fmt.Errorf("pick next field: %w", err)
			return nil, err
		}
		question, options, err := s.generateClarifyQuestion(field, session)
		if err != nil {
			err = fmt.Errorf("generate clarify question: %w", err)
			return nil, err
		}

		assistantMsg = model.InspirationMessage{
			SessionID: inspirationMsg.SessionID,
			Role:      model.InspirationMessageRoleAssistant,
			Kind:      model.MessageKindClarifyAsk,
			Content:   question,
			Options:   options,
			CreatedAt: time.Now().UTC(),
		}
		response = http_dto.InspirationMessageResponse{
			SessionID:   inspirationMsg.SessionID,
			Role:        string(model.InspirationMessageRoleAssistant),
			Kind:        string(model.MessageKindClarifyAsk),
			Content:     question,
			Options:     toResponseOptions(options),
			TargetField: string(field),
		}
		session.Status = statusForField(field)
	}

	savedAssistant, err := s.messageDao.Create(ctx, &assistantMsg)
	if err != nil {
		err = fmt.Errorf("persist assistant inspiration message %w", err)
		return nil, err
	}
	session.Messages = append(session.Messages, savedAssistant.ID)

	if _, err := s.sessionDao.Update(ctx, session); err != nil {
		err = fmt.Errorf("update inspiration session %w", err)
		return nil, err
	}

	// Mark as completed
	requestProcess.Stage = model.RequestStageCompleted
	requestProcess.CompletedAt = time.Now().UTC()
	s.processDao.Update(ctx, requestProcess)

	return &response, nil
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
	prompt := fmt.Sprintf(travelRelatedCriteria, content)

	raw, err := s.llmClient.Call(prompt)
	if err != nil {
		return false, err
	}
	var payload struct {
		TravelRelated bool `json:"travel_related"`
	}
	if err := decodeFirstJSONObject(raw, &payload); err != nil {
		return false, err
	}
	return payload.TravelRelated, nil
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

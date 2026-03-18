package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ai-reading-assistant/internal/agent"
	"ai-reading-assistant/internal/dao"
	"ai-reading-assistant/internal/dto/http_dto"
	"ai-reading-assistant/internal/model"
)

type InspirationOrchestrator interface {
	HandleChatCompletion(ctx context.Context, request *http_dto.InspirationMessageRequest) (*http_dto.InspirationMessageResponse, error)
}

type inspirationOrchestrator struct {
	sessionDao         dao.InspirationSessionDao
	messageDao         dao.InspirationMessageDao
	processDao         dao.RequestProcessDao
	intentAgent        agent.IntentAgent
	requirementAgent   agent.RequirementAnalyzerAgent
	clarificationAgent agent.ClarificationAgent
	inspirationAgent   agent.InspirationAgent
	keywordAgent       agent.KeywordAgent
	wikipediaAgent     agent.WikipediaAgent
}

func NewInspirationOrchestrator(
	sessionDao dao.InspirationSessionDao,
	messageDao dao.InspirationMessageDao,
	processDao dao.RequestProcessDao,
	intentAgent agent.IntentAgent,
	requirementAgent agent.RequirementAnalyzerAgent,
	clarificationAgent agent.ClarificationAgent,
	inspirationAgent agent.InspirationAgent,
	keywordAgent agent.KeywordAgent,
	wikipediaAgent agent.WikipediaAgent,
) InspirationOrchestrator {
	return &inspirationOrchestrator{
		sessionDao:         sessionDao,
		messageDao:         messageDao,
		processDao:         processDao,
		intentAgent:        intentAgent,
		requirementAgent:   requirementAgent,
		clarificationAgent: clarificationAgent,
		inspirationAgent:   inspirationAgent,
		keywordAgent:       keywordAgent,
		wikipediaAgent:     wikipediaAgent,
	}
}

func (o *inspirationOrchestrator) HandleChatCompletion(ctx context.Context, request *http_dto.InspirationMessageRequest) (*http_dto.InspirationMessageResponse, error) {
	var userID string
	if session, err := o.sessionDao.GetByID(ctx, request.SessionID); err == nil && session != nil {
		userID = session.UserID
	}

	requestProcess := &model.RequestProcess{
		SessionID: request.SessionID,
		UserID:    userID,
		Stage:     model.RequestStageDetectUserIntent,
		StartedAt: time.Now().UTC(),
	}

	savedProcess, err := o.processDao.Create(ctx, requestProcess)
	if err != nil {
		return nil, fmt.Errorf("create request process: %w", err)
	}
	requestProcess = savedProcess

	defer func() {
		requestProcess.CompletedAt = time.Now().UTC()
		_, _ = o.processDao.Update(ctx, requestProcess)
	}()

	inspirationMsg, err := request.ToModel()
	if err != nil {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = fmt.Sprintf("create inspiration message %v", err)
		return nil, fmt.Errorf("create inspiration message %w", err)
	}
	if inspirationMsg.Kind == "" {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = "empty message kind"
		return nil, fmt.Errorf("empty message kind")
	}

	session, err := o.sessionDao.GetByID(ctx, inspirationMsg.SessionID)
	if err != nil {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = fmt.Sprintf("load inspiration session %v", err)
		return nil, fmt.Errorf("load inspiration session %w", err)
	}
	session.EnsureRequirementInitialized()

	if inspirationMsg.Kind == model.MessageKindUserInput {
		ok, err := o.detectTravelIntent(ctx, inspirationMsg.Content)
		if err != nil {
			requestProcess.Stage = model.RequestStageFailed
			requestProcess.Error = fmt.Sprintf("classify travel intent %v", err)
			return nil, fmt.Errorf("classify travel intent %w", err)
		}
		if !ok {
			resp, err := o.handleNonTravelInput(ctx, session, inspirationMsg)
			if err != nil {
				requestProcess.Stage = model.RequestStageFailed
				requestProcess.Error = err.Error()
				return nil, err
			}
			requestProcess.Stage = model.RequestStageCompleted
			return resp, nil
		}
	}

	if inspirationMsg.StartNewInspiration {
		if session.AppendRequirement() == nil {
			requestProcess.Stage = model.RequestStageFailed
			requestProcess.Error = "append requirement failed"
			return nil, fmt.Errorf("append requirement failed")
		}
		session.Status = model.SessionStatusStartOver
	}

	targetField := statusClarifyField(session.Status)

	if err := o.setRequestStage(ctx, requestProcess, model.RequestStageAnalyzeRequirement); err != nil {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = fmt.Sprintf("update process to analyze requirement: %v", err)
		return nil, fmt.Errorf("update process to analyze requirement: %w", err)
	}

	if err := o.requirementAgent.Analyze(ctx, inspirationMsg, session, targetField); err != nil {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = fmt.Sprintf("anaylyze requirement %v", err)
		return nil, fmt.Errorf("anaylyze requirement %w", err)
	}
	advanceSessionStatus(session)

	userMsg, err := o.messageDao.Create(ctx, inspirationMsg)
	if err != nil {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = fmt.Sprintf("persist user inspiration message %v", err)
		return nil, fmt.Errorf("persist user inspiration message %w", err)
	}
	session.Messages = append(session.Messages, userMsg.ID)

	assistantMsg, response, err := o.buildAssistantResponse(ctx, requestProcess, session, inspirationMsg)
	if err != nil {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = err.Error()
		return nil, err
	}

	savedAssistant, err := o.messageDao.Create(ctx, assistantMsg)
	if err != nil {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = fmt.Sprintf("persist assistant inspiration message %v", err)
		return nil, fmt.Errorf("persist assistant inspiration message %w", err)
	}
	session.Messages = append(session.Messages, savedAssistant.ID)

	if _, err := o.sessionDao.Update(ctx, session); err != nil {
		requestProcess.Stage = model.RequestStageFailed
		requestProcess.Error = fmt.Sprintf("update inspiration session %v", err)
		return nil, fmt.Errorf("update inspiration session %w", err)
	}

	requestProcess.Stage = model.RequestStageCompleted
	requestProcess.CompletedAt = time.Now().UTC()
	_, _ = o.processDao.Update(ctx, requestProcess)

	return response, nil
}

func (o *inspirationOrchestrator) detectTravelIntent(ctx context.Context, content string) (bool, error) {
	if strings.TrimSpace(content) == "" {
		return false, fmt.Errorf("content is empty")
	}
	if o.intentAgent == nil {
		return false, fmt.Errorf("intent agent is not initialized")
	}

	result, err := o.intentAgent.DetectTravelIntent(ctx, content)
	if err != nil {
		return false, err
	}
	if result == nil {
		return false, fmt.Errorf("intent agent returned nil result")
	}
	return result.TravelRelated, nil
}

func (o *inspirationOrchestrator) handleNonTravelInput(ctx context.Context, session *model.InspirationSession, inspirationMsg *model.InspirationMessage) (*http_dto.InspirationMessageResponse, error) {
	userMsg, err := o.messageDao.Create(ctx, inspirationMsg)
	if err != nil {
		return nil, fmt.Errorf("persist user inspiration message %w", err)
	}
	session.Messages = append(session.Messages, userMsg.ID)

	content := "当前输入未识别为旅行请求,请更具体描述你的旅行意图"
	assistantMsg := &model.InspirationMessage{
		SessionID: inspirationMsg.SessionID,
		Role:      model.InspirationMessageRoleAssistant,
		Kind:      model.MessageKindAssistant,
		Content:   content,
		CreatedAt: time.Now().UTC(),
	}
	savedAssistant, err := o.messageDao.Create(ctx, assistantMsg)
	if err != nil {
		return nil, fmt.Errorf("persist assistant inspiration message %w", err)
	}
	session.Messages = append(session.Messages, savedAssistant.ID)

	if _, err := o.sessionDao.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("update inspiration session %w", err)
	}

	return &http_dto.InspirationMessageResponse{
		SessionID:     inspirationMsg.SessionID,
		Role:          string(model.InspirationMessageRoleAssistant),
		Kind:          string(model.MessageKindAssistant),
		Content:       content,
		IsInspiration: false,
	}, nil
}

func (o *inspirationOrchestrator) buildAssistantResponse(ctx context.Context, requestProcess *model.RequestProcess, session *model.InspirationSession, inspirationMsg *model.InspirationMessage) (*model.InspirationMessage, *http_dto.InspirationMessageResponse, error) {
	if session.IsReadyToGenerate() {
		if err := o.setRequestStage(ctx, requestProcess, model.RequestStageGenerateInspiration); err != nil {
			return nil, nil, fmt.Errorf("update process to generate inspiration: %w", err)
		}
		replyContent, err := o.inspirationAgent.Generate(ctx, session)
		if err != nil {
			return nil, nil, fmt.Errorf("generate inspiration %w", err)
		}
		current, ok := session.CurrentRequirement()
		if ok {
			current.Output = replyContent
			o.enrichGeneratedInspiration(ctx, requestProcess, current)
		}
		session.Status = model.SessionStatusCompleted

		return &model.InspirationMessage{
				SessionID: inspirationMsg.SessionID,
				Role:      model.InspirationMessageRoleAssistant,
				Kind:      model.MessageKindAssistant,
				Content:   replyContent,
				CreatedAt: time.Now().UTC(),
			}, &http_dto.InspirationMessageResponse{
				SessionID:     inspirationMsg.SessionID,
				Role:          string(model.InspirationMessageRoleAssistant),
				Kind:          string(model.MessageKindAssistant),
				Content:       replyContent,
				IsInspiration: true,
				Inspiration:   toResponseInspiration(current),
			}, nil
	}

	if err := o.setRequestStage(ctx, requestProcess, model.RequestStageGenerateOptions); err != nil {
		return nil, nil, fmt.Errorf("update process to generate options: %w", err)
	}
	field, err := pickNextField(session)
	if err != nil {
		return nil, nil, fmt.Errorf("pick next field: %w", err)
	}
	result, err := o.clarificationAgent.GenerateQuestion(ctx, field, session)
	if err != nil {
		return nil, nil, fmt.Errorf("generate clarify question: %w", err)
	}
	session.Status = statusForField(result.TargetField)

	return &model.InspirationMessage{
			SessionID: inspirationMsg.SessionID,
			Role:      model.InspirationMessageRoleAssistant,
			Kind:      model.MessageKindClarifyAsk,
			Content:   result.Question,
			Options:   result.Options,
			CreatedAt: time.Now().UTC(),
		}, &http_dto.InspirationMessageResponse{
			SessionID:     inspirationMsg.SessionID,
			Role:          string(model.InspirationMessageRoleAssistant),
			Kind:          string(model.MessageKindClarifyAsk),
			Content:       result.Question,
			Options:       toResponseOptions(result.Options),
			TargetField:   string(result.TargetField),
			IsInspiration: false,
		}, nil
}

func (o *inspirationOrchestrator) enrichGeneratedInspiration(ctx context.Context, requestProcess *model.RequestProcess, inspiration *model.Inspiration) {
	if inspiration == nil || o.keywordAgent == nil {
		return
	}
	_ = o.setRequestStage(ctx, requestProcess, model.RequestStageEnrichKeywords)

	keywords, err := o.keywordAgent.ExtractKeywordsFromOutput(ctx, inspiration.Output)
	if err != nil {
		return
	}
	if len(keywords) == 0 {
		inspiration.KeyWords = make([]model.KeyWord, 0)
		return
	}
	if o.wikipediaAgent != nil {
		keywords = o.wikipediaAgent.EnrichKeywords(ctx, keywords)
	}
	inspiration.KeyWords = keywords
}

func (o *inspirationOrchestrator) setRequestStage(ctx context.Context, requestProcess *model.RequestProcess, stage model.RequestStage) error {
	if requestProcess == nil {
		return fmt.Errorf("request process is nil")
	}
	requestProcess.Stage = stage
	_, err := o.processDao.Update(ctx, requestProcess)
	return err
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

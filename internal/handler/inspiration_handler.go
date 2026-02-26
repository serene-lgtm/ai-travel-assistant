package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"ai-reading-assistant/internal/dto/http_dto"
	"ai-reading-assistant/internal/service"
)

// InspirationHandler hosts HTTP glue for the travel plan feature
type InspirationHandler struct {
	service service.InspirationService
}

// NewInspirationPlanHandler wires the handler with its service dependency
func NewInspirationPlanHandler(svc service.InspirationService) *InspirationHandler {
	return &InspirationHandler{service: svc}
}

// CreateInspirationSession creates a new inspiration session without previous context
func (h *InspirationHandler) CreateInspirationSession(c *gin.Context) {
	var req http_dto.InspirationSessionCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID, err := h.service.CreateInspirationSession(req.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"session_id": sessionID})
}

func (h *InspirationHandler) ChatCompletion(c *gin.Context) {
	var req http_dto.InspirationMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ChatCompletion(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GetInspirationSession returns session detail by ID.
func (h *InspirationHandler) GetInspirationSession(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	session, err := h.service.GetInspirationSession(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *InspirationHandler) FavoriteInspiration(c *gin.Context) {
	var req http_dto.InspirationFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.FavoriteInspiration(req.SessionID, req.InspirationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *InspirationHandler) UnfavoriteInspiration(c *gin.Context) {
	var req http_dto.InspirationFavoriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UnfavoriteInspiration(req.SessionID, req.InspirationID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

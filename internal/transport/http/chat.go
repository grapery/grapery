package http

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// ListChatSessions GET /api/v1/chat/sessions
func (h *Handler) ListChatSessions(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if h.chatService == nil {
		InternalError(c, "chat service unavailable")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	sessions, total, err := h.chatService.ListSessions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{
		"sessions": sessions,
		"total":    total,
		"count":    len(sessions),
	})
}

// StartChatSession POST /api/v1/chat/sessions
func (h *Handler) StartChatSession(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if h.chatService == nil {
		InternalError(c, "chat service unavailable")
		return
	}
	var req service.StartChatRequest
	if !BindJSON(c, &req) {
		return
	}
	session, err := h.chatService.StartSession(c.Request.Context(), userID, req)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, session)
}

// GetChatSession GET /api/v1/chat/sessions/:id
func (h *Handler) GetChatSession(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if h.chatService == nil {
		InternalError(c, "chat service unavailable")
		return
	}
	sessionID := strings.TrimSpace(c.Param("id"))
	session, err := h.chatService.GetSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, session)
}

// ListChatMessages GET /api/v1/chat/sessions/:id/messages
func (h *Handler) ListChatMessages(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if h.chatService == nil {
		InternalError(c, "chat service unavailable")
		return
	}
	sessionID := strings.TrimSpace(c.Param("id"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	before, _ := strconv.ParseInt(c.DefaultQuery("before", "0"), 10, 64)
	messages, err := h.chatService.ListMessages(c.Request.Context(), userID, sessionID, before, limit)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"messages": messages, "count": len(messages)})
}

// SendChatMessage POST /api/v1/chat/sessions/:id/messages
func (h *Handler) SendChatMessage(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if h.chatService == nil {
		InternalError(c, "chat service unavailable")
		return
	}
	sessionID := strings.TrimSpace(c.Param("id"))
	var req service.SendChatMessageRequest
	if !BindJSON(c, &req) {
		return
	}
	result, err := h.chatService.SendMessage(c.Request.Context(), userID, sessionID, req.Content)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, result)
}

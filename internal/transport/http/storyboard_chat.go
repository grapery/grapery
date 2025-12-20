package http

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// StoryboardChatHandler handles storyboard creation chat HTTP requests
type StoryboardChatHandler struct {
	chatService *service.StoryboardChatService
	logger      *zap.Logger
}

// NewStoryboardChatHandler creates a new storyboard chat handler
func NewStoryboardChatHandler(chatService *service.StoryboardChatService, logger *zap.Logger) *StoryboardChatHandler {
	return &StoryboardChatHandler{
		chatService: chatService,
		logger:      logger,
	}
}

// ===================== Request/Response Models =====================

// StartSessionRequest request to start a new storyboard chat session
type StartSessionRequest struct {
	StoryID string `json:"storyId" binding:"required"`
}

// StartSessionResponse response for starting a session
type StartSessionResponse struct {
	Session *SessionResponse `json:"session"`
	Message *MessageResponse `json:"message,omitempty"`
}

// SessionResponse represents a storyboard chat session
type SessionResponse struct {
	ID                 string   `json:"id"`
	UserID             string   `json:"userId"`
	StoryID            string   `json:"storyId"`
	StoryboardID       string   `json:"storyboardId,omitempty"`
	CurrentStep        int      `json:"currentStep"`
	Status             string   `json:"status"`
	SelectedCharacters []string `json:"selectedCharacters,omitempty"`
	SelectedStyle      string   `json:"selectedStyle,omitempty"`
	CreatedAt          int64    `json:"createdAt"`
	UpdatedAt          int64    `json:"updatedAt"`
}

// MessageResponse represents a storyboard chat message
type MessageResponse struct {
	ID          string              `json:"id"`
	SessionID   string              `json:"sessionId"`
	MessageType string              `json:"messageType"`
	Status      string              `json:"status"`
	Step        int                 `json:"step"`
	Data        interface{}         `json:"data,omitempty"`
	Actions     []domain.ChatAction `json:"actions,omitempty"`
	Content     string              `json:"content,omitempty"`
	Timestamp   int64               `json:"timestamp"`
	IsUser      bool                `json:"isUser"`
}

// SendMessageRequest request to send a message
type SendMessageRequestBody struct {
	ActionID string      `json:"actionId,omitempty"`
	Content  string      `json:"content,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

// ===================== HTTP Handlers =====================

// StartSession starts a new storyboard creation chat session
// POST /api/agent/storyboard-chat/start
func (h *StoryboardChatHandler) StartSession(c *gin.Context) {
	var req StartSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, "Invalid request: "+err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	session, msg, err := h.chatService.StartSession(c.Request.Context(), userID.(string), req.StoryID)
	if err != nil {
		h.logger.Error("failed to start session", zap.Error(err))
		InternalError(c, "Failed to start session: "+err.Error())
		return
	}

	resp := StartSessionResponse{
		Session: h.sessionToResponse(session),
	}

	if msg != nil {
		resp.Message = h.messageToResponse(msg)
	}

	Success(c, resp)
}

// GetSession gets session details
// GET /api/agent/storyboard-chat/sessions/:id
func (h *StoryboardChatHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		InvalidParams(c, "Session ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("failed to get session", zap.Error(err))
		NotFound(c, "Session not found")
		return
	}

	// Verify ownership
	if session.UserID != userID.(string) {
		Forbidden(c, "Access denied")
		return
	}

	Success(c, h.sessionToResponse(session))
}

// GetMessages gets messages for a session
// GET /api/agent/storyboard-chat/sessions/:id/messages
func (h *StoryboardChatHandler) GetMessages(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		InvalidParams(c, "Session ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	// Verify session ownership
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		NotFound(c, "Session not found")
		return
	}

	if session.UserID != userID.(string) {
		Forbidden(c, "Access denied")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	messages, err := h.chatService.GetMessages(c.Request.Context(), sessionID, limit, offset)
	if err != nil {
		h.logger.Error("failed to get messages", zap.Error(err))
		InternalError(c, "Failed to get messages")
		return
	}

	resp := make([]MessageResponse, len(messages))
	for i, msg := range messages {
		resp[i] = *h.messageToResponse(msg)
	}

	Success(c, gin.H{
		"messages": resp,
		"total":    len(resp),
	})
}

// SendMessage sends a user message/action
// POST /api/agent/storyboard-chat/sessions/:id/send
func (h *StoryboardChatHandler) SendMessage(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		InvalidParams(c, "Session ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	// Verify session ownership
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		NotFound(c, "Session not found")
		return
	}

	if session.UserID != userID.(string) {
		Forbidden(c, "Access denied")
		return
	}

	var req SendMessageRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, "Invalid request: "+err.Error())
		return
	}

	// Convert request to domain type
	domainReq := &domain.SendMessageRequest{
		ActionID: req.ActionID,
		Content:  req.Content,
	}

	// Marshal data if provided
	if req.Data != nil {
		dataBytes, err := storyboardChatJSONMarshal(req.Data)
		if err != nil {
			InvalidParams(c, "Invalid data format")
			return
		}
		domainReq.Data = dataBytes
	}

	msg, err := h.chatService.SendMessage(c.Request.Context(), sessionID, domainReq)
	if err != nil {
		h.logger.Error("failed to send message", zap.Error(err))
		InternalError(c, "Failed to send message: "+err.Error())
		return
	}

	Success(c, h.messageToResponse(msg))
}

// GetStatus gets current generation status (for polling)
// GET /api/agent/storyboard-chat/sessions/:id/status
func (h *StoryboardChatHandler) GetStatus(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		InvalidParams(c, "Session ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	// Verify session ownership
	session, err := h.chatService.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		NotFound(c, "Session not found")
		return
	}

	if session.UserID != userID.(string) {
		Forbidden(c, "Access denied")
		return
	}

	msg, err := h.chatService.GetGenerationStatus(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.Error("failed to get status", zap.Error(err))
		InternalError(c, "Failed to get status: "+err.Error())
		return
	}

	resp := gin.H{
		"session": h.sessionToResponse(session),
	}

	if msg != nil {
		resp["message"] = h.messageToResponse(msg)
	}

	Success(c, resp)
}

// ListSessions lists user's storyboard chat sessions
// GET /api/agent/storyboard-chat/sessions
func (h *StoryboardChatHandler) ListSessions(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sessions, err := h.chatService.ListSessions(c.Request.Context(), userID.(string), limit, offset)
	if err != nil {
		h.logger.Error("failed to list sessions", zap.Error(err))
		InternalError(c, "Failed to list sessions")
		return
	}

	resp := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		resp[i] = *h.sessionToResponse(s)
	}

	Success(c, gin.H{
		"sessions": resp,
		"total":    len(resp),
	})
}

// ===================== Response Converters =====================

func (h *StoryboardChatHandler) sessionToResponse(session *domain.StoryboardChatSession) *SessionResponse {
	return &SessionResponse{
		ID:                 session.ID,
		UserID:             session.UserID,
		StoryID:            session.StoryID,
		StoryboardID:       session.StoryboardID,
		CurrentStep:        session.CurrentStep,
		Status:             session.Status,
		SelectedCharacters: session.SelectedCharacters,
		SelectedStyle:      session.SelectedStyle,
		CreatedAt:          session.CreatedAt,
		UpdatedAt:          session.UpdatedAt,
	}
}

func (h *StoryboardChatHandler) messageToResponse(msg *domain.StoryboardChatMessage) *MessageResponse {
	resp := &MessageResponse{
		ID:          msg.ID,
		SessionID:   msg.SessionID,
		MessageType: msg.MessageType,
		Status:      msg.Status,
		Step:        msg.Step,
		Actions:     msg.Actions,
		Content:     msg.Content,
		Timestamp:   msg.Timestamp,
		IsUser:      msg.IsUser,
	}

	// Parse data JSON to interface
	if len(msg.Data) > 0 {
		var data interface{}
		if err := storyboardChatJSONUnmarshal(msg.Data, &data); err == nil {
			resp.Data = data
		}
	}

	return resp
}

// RegisterRoutes registers storyboard chat routes
func (h *StoryboardChatHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	chatGroup := r.Group("/storyboard-chat")
	chatGroup.Use(authMiddleware)

	chatGroup.POST("/start", h.StartSession)
	chatGroup.GET("/sessions", h.ListSessions)
	chatGroup.GET("/sessions/:id", h.GetSession)
	chatGroup.GET("/sessions/:id/messages", h.GetMessages)
	chatGroup.POST("/sessions/:id/send", h.SendMessage)
	chatGroup.GET("/sessions/:id/status", h.GetStatus)
}

// Helper functions for JSON marshaling
func storyboardChatJSONMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func storyboardChatJSONUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}


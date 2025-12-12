package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// AgentChatHandler 处理Agent聊天相关的HTTP请求
type AgentChatHandler struct {
	chatService *service.AgentChatService
	logger      *zap.Logger
}

// NewAgentChatHandler 创建Agent聊天处理器
func NewAgentChatHandler(chatService *service.AgentChatService, logger *zap.Logger) *AgentChatHandler {
	return &AgentChatHandler{
		chatService: chatService,
		logger:      logger,
	}
}

// ===================== Request/Response Models =====================

// SendMessageRequest 发送消息请求
type SendMessageRequest struct {
	CharacterID string  `json:"characterId" binding:"required"`
	Content     string  `json:"content" binding:"required"`
	ThreadID    *string `json:"threadId,omitempty"`
	Image       string  `json:"image,omitempty"`
}

// SendMessageResponse 发送消息响应
type SendMessageResponse struct {
	UserMessage  ChatMessageResponse  `json:"userMessage"`
	AgentReply   *ChatMessageResponse `json:"agentReply,omitempty"`
	ThreadID     string               `json:"threadId"`
	Success      bool                 `json:"success"`
	ErrorMessage string               `json:"errorMessage,omitempty"`
}

// ChatMessageResponse 聊天消息响应
type ChatMessageResponse struct {
	ID           string `json:"id"`
	ThreadID     string `json:"threadId"`
	SenderID     string `json:"senderId"`
	SenderName   string `json:"senderName"`
	SenderAvatar string `json:"senderAvatar"`
	Content      string `json:"content"`
	Image        string `json:"image,omitempty"`
	Timestamp    int64  `json:"timestamp"`
	IsUser       bool   `json:"isUser"`
}

// ChatThreadResponse 聊天线程响应
type ChatThreadResponse struct {
	ID                   string  `json:"id"`
	CharacterID          string  `json:"characterId"`
	CharacterName        string  `json:"characterName"`
	CharacterAvatar      string  `json:"characterAvatar"`
	StoryTitle           string  `json:"storyTitle,omitempty"`
	LastMessage          string  `json:"lastMessage"`
	LastMessageTime      int64   `json:"lastMessageTime"`
	UnreadCount          int     `json:"unreadCount"`
	MessageCount         int     `json:"messageCount"`
	InteractionFrequency float64 `json:"interactionFrequency"`
	CreatedAt            int64   `json:"createdAt"`
}

// ===================== HTTP Handlers =====================

// SendMessage 发送消息给Agent
// POST /api/agent/chat/send
func (h *AgentChatHandler) SendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// 从认证中间件获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 发送消息并获取回复
	userMessage, agentReply, err := h.chatService.SendMessageToAgent(
		c.Request.Context(),
		userID.(string),
		req.CharacterID,
		req.Content,
		req.ThreadID,
	)

	if err != nil {
		h.logger.Error("failed to send message", zap.Error(err))

		// 如果有用户消息但Agent回复失败
		if userMessage != nil {
			c.JSON(http.StatusOK, SendMessageResponse{
				UserMessage: ChatMessageResponse{
					ID:           userMessage.ID,
					ThreadID:     userMessage.ThreadID,
					SenderID:     userMessage.SenderID,
					SenderName:   userMessage.SenderName,
					SenderAvatar: userMessage.SenderAvatar,
					Content:      userMessage.Content,
					Image:        userMessage.Image,
					Timestamp:    userMessage.Timestamp,
					IsUser:       userMessage.IsUser,
				},
				ThreadID:     userMessage.ThreadID,
				Success:      false,
				ErrorMessage: "AI生成回复失败: " + err.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message: " + err.Error()})
		return
	}

	// 构建响应
	resp := SendMessageResponse{
		UserMessage: ChatMessageResponse{
			ID:           userMessage.ID,
			ThreadID:     userMessage.ThreadID,
			SenderID:     userMessage.SenderID,
			SenderName:   userMessage.SenderName,
			SenderAvatar: userMessage.SenderAvatar,
			Content:      userMessage.Content,
			Image:        userMessage.Image,
			Timestamp:    userMessage.Timestamp,
			IsUser:       userMessage.IsUser,
		},
		ThreadID: userMessage.ThreadID,
		Success:  true,
	}

	if agentReply != nil {
		resp.AgentReply = &ChatMessageResponse{
			ID:           agentReply.ID,
			ThreadID:     agentReply.ThreadID,
			SenderID:     agentReply.SenderID,
			SenderName:   agentReply.SenderName,
			SenderAvatar: agentReply.SenderAvatar,
			Content:      agentReply.Content,
			Image:        agentReply.Image,
			Timestamp:    agentReply.Timestamp,
			IsUser:       agentReply.IsUser,
		}
	}

	c.JSON(http.StatusOK, resp)
}

// GetChatHistory 获取聊天历史
// GET /api/agent/chat/history/:threadId
func (h *AgentChatHandler) GetChatHistory(c *gin.Context) {
	threadID := c.Param("threadId")
	if threadID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thread ID is required"})
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 分页参数
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	messages, err := h.chatService.GetChatHistory(
		c.Request.Context(),
		userID.(string),
		threadID,
		limit,
		offset,
	)

	if err != nil {
		h.logger.Error("failed to get chat history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get chat history"})
		return
	}

	// 转换为响应格式
	resp := make([]ChatMessageResponse, len(messages))
	for i, msg := range messages {
		resp[i] = ChatMessageResponse{
			ID:           msg.ID,
			ThreadID:     msg.ThreadID,
			SenderID:     msg.SenderID,
			SenderName:   msg.SenderName,
			SenderAvatar: msg.SenderAvatar,
			Content:      msg.Content,
			Image:        msg.Image,
			Timestamp:    msg.Timestamp,
			IsUser:       msg.IsUser,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"messages": resp,
		"total":    len(resp),
	})
}

// ListChatThreads 列出用户的聊天线程
// GET /api/agent/chat/threads
func (h *AgentChatHandler) ListChatThreads(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	threads, err := h.chatService.ListUserChatThreads(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.Error("failed to list chat threads", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list chat threads"})
		return
	}

	// 转换为响应格式
	resp := make([]ChatThreadResponse, len(threads))
	for i, thread := range threads {
		resp[i] = ChatThreadResponse{
			ID:                   thread.ID,
			CharacterID:          thread.CharacterID,
			CharacterName:        thread.CharacterName,
			CharacterAvatar:      thread.CharacterAvatar,
			StoryTitle:           thread.StoryTitle,
			LastMessage:          thread.LastMessage,
			LastMessageTime:      thread.LastMessageTime,
			UnreadCount:          thread.UnreadCount,
			MessageCount:         thread.MessageCount,
			InteractionFrequency: thread.InteractionFrequency,
			CreatedAt:            thread.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"threads": resp,
		"total":   len(resp),
	})
}

// GetAgent 获取角色的Agent信息
// GET /api/agent/character/:characterId
func (h *AgentChatHandler) GetAgent(c *gin.Context) {
	characterID := c.Param("characterId")
	if characterID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Character ID is required"})
		return
	}

	agent, err := h.chatService.GetAgentByCharacter(c.Request.Context(), characterID)
	if err != nil {
		h.logger.Error("failed to get agent", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// UpdateAgentConfig 更新Agent配置（管理员功能）
// PUT /api/agent/:agentId/config
func (h *AgentChatHandler) UpdateAgentConfig(c *gin.Context) {
	agentID := c.Param("agentId")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	var config struct {
		SystemPrompt string  `json:"systemPrompt"`
		Temperature  float64 `json:"temperature"`
		MaxTokens    int     `json:"maxTokens"`
		Provider     string  `json:"provider"`
		Model        string  `json:"model"`
	}

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// 获取现有Agent
	agent, err := h.chatService.GetAgentByCharacter(c.Request.Context(), agentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	// 更新配置
	if config.SystemPrompt != "" {
		agent.SystemPrompt = config.SystemPrompt
	}
	if config.Temperature > 0 {
		agent.Temperature = config.Temperature
	}
	if config.MaxTokens > 0 {
		agent.MaxTokens = config.MaxTokens
	}
	if config.Provider != "" {
		agent.Provider = config.Provider
	}
	if config.Model != "" {
		agent.Model = config.Model
	}

	if err := h.chatService.UpdateAgentConfig(c.Request.Context(), agentID, agent); err != nil {
		h.logger.Error("failed to update agent config", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Agent config updated successfully",
		"agent":   agent,
	})
}

// RegisterRoutes 注册Agent聊天相关的路由
func (h *AgentChatHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	agentGroup := r.Group("/agent")
	agentGroup.Use(authMiddleware) // 需要认证

	// 聊天相关
	agentGroup.POST("/chat/send", h.SendMessage)
	agentGroup.GET("/chat/history/:threadId", h.GetChatHistory)
	agentGroup.GET("/chat/threads", h.ListChatThreads)

	// Agent管理
	agentGroup.GET("/character/:characterId", h.GetAgent)
	agentGroup.PUT("/:agentId/config", h.UpdateAgentConfig) // 需要管理员权限
}

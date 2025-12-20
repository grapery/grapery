package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
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
	CharacterID        string  `json:"characterId" binding:"required"`
	Content            string  `json:"content" binding:"required"`
	ThreadID           *string `json:"threadId,omitempty"`
	Image              string  `json:"image,omitempty"`
	StoryboardBranchID *string `json:"storyboardBranchId,omitempty"` // Leaf node storyboard ID
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
	ID           string                    `json:"id"`
	ThreadID     string                    `json:"threadId"`
	SenderID     string                    `json:"senderId"`
	SenderName   string                    `json:"senderName"`
	SenderAvatar string                    `json:"senderAvatar"`
	Content      string                    `json:"content"`
	Image        string                    `json:"image,omitempty"`
	Timestamp    int64                     `json:"timestamp"`
	IsUser       bool                      `json:"isUser"`
	Reactions    []MessageReactionResponse `json:"reactions,omitempty"`
	TokenUsage   *TokenUsageResponse       `json:"tokenUsage,omitempty"`
}

// MessageReactionResponse 消息互动响应
type MessageReactionResponse struct {
	ID           string `json:"id"`
	UserID       string `json:"userId"`
	ReactionType string `json:"reactionType"`
	EmojiCode    string `json:"emojiCode,omitempty"`
	CreatedAt    int64  `json:"createdAt"`
}

// TokenUsageResponse Token使用统计响应
type TokenUsageResponse struct {
	InputTokens  int    `json:"inputTokens"`
	OutputTokens int    `json:"outputTokens"`
	TotalTokens  int    `json:"totalTokens"`
	Model        string `json:"model,omitempty"`
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
	SelectedStoryboardID string  `json:"selectedStoryboardId,omitempty"`
	TotalTokensUsed      int64   `json:"totalTokensUsed"`
	IsArchived           bool    `json:"isArchived"`
	Summary              string  `json:"summary,omitempty"`
	CreatedAt            int64   `json:"createdAt"`
	UpdatedAt            int64   `json:"updatedAt"`
}

// CreateThreadRequest 创建聊天线程请求
type CreateThreadRequest struct {
	CharacterID string `json:"characterId" binding:"required"`
}

// ===================== HTTP Handlers =====================

// SendMessage 发送消息给Agent
// POST /api/agent/chat/send
func (h *AgentChatHandler) SendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, "Invalid request: "+err.Error())
		return
	}

	// 从认证中间件获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	// 如果提供了故事板分支，先选择它
	if req.StoryboardBranchID != nil && *req.StoryboardBranchID != "" {
		var threadID string
		if req.ThreadID != nil && *req.ThreadID != "" {
			threadID = *req.ThreadID
		} else {
			// 需要先创建或获取线程
			threads, err := h.chatService.ListUserChatThreads(c.Request.Context(), userID.(string))
			if err == nil {
				for _, t := range threads {
					if t.CharacterID == req.CharacterID {
						threadID = t.ID
						break
					}
				}
			}
		}
		if threadID != "" {
			if err := h.chatService.SelectStoryboardBranch(c.Request.Context(), threadID, *req.StoryboardBranchID, req.CharacterID); err != nil {
				h.logger.Warn("failed to select storyboard branch", zap.Error(err))
			}
		}
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
			Success(c, SendMessageResponse{
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

		InternalError(c, "Failed to send message: "+err.Error())
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
		agentResp := ChatMessageResponse{
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
		if agentReply.Reactions != nil {
			agentResp.Reactions = make([]MessageReactionResponse, len(agentReply.Reactions))
			for i, r := range agentReply.Reactions {
				agentResp.Reactions[i] = MessageReactionResponse{
					ID:           r.ID,
					UserID:       r.UserID,
					ReactionType: r.ReactionType,
					EmojiCode:    r.EmojiCode,
					CreatedAt:    r.CreatedAt,
				}
			}
		}
		if agentReply.TokenUsage != nil {
			agentResp.TokenUsage = &TokenUsageResponse{
				InputTokens:  agentReply.TokenUsage.InputTokens,
				OutputTokens: agentReply.TokenUsage.OutputTokens,
				TotalTokens:  agentReply.TokenUsage.TotalTokens,
				Model:        agentReply.TokenUsage.Model,
			}
		}
		resp.AgentReply = &agentResp
	}

	Success(c, resp)
}

// GetChatHistory 获取聊天历史
// GET /api/agent/chat/history/:threadId
func (h *AgentChatHandler) GetChatHistory(c *gin.Context) {
	threadID := c.Param("threadId")
	if threadID == "" {
		InvalidParams(c, "Thread ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
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
		InternalError(c, "Failed to get chat history")
		return
	}

	// 转换为响应格式
	resp := make([]ChatMessageResponse, len(messages))
	for i, msg := range messages {
		msgResp := ChatMessageResponse{
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
		if msg.Reactions != nil {
			msgResp.Reactions = make([]MessageReactionResponse, len(msg.Reactions))
			for j, r := range msg.Reactions {
				msgResp.Reactions[j] = MessageReactionResponse{
					ID:           r.ID,
					UserID:       r.UserID,
					ReactionType: r.ReactionType,
					EmojiCode:    r.EmojiCode,
					CreatedAt:    r.CreatedAt,
				}
			}
		}
		if msg.TokenUsage != nil {
			msgResp.TokenUsage = &TokenUsageResponse{
				InputTokens:  msg.TokenUsage.InputTokens,
				OutputTokens: msg.TokenUsage.OutputTokens,
				TotalTokens:  msg.TokenUsage.TotalTokens,
				Model:        msg.TokenUsage.Model,
			}
		}
		resp[i] = msgResp
	}

	Success(c, gin.H{
		"messages": resp,
		"total":    len(resp),
	})
}

// ListChatThreads 列出用户的聊天线程
// GET /api/agent/chat/threads
// Query params: characterId (optional) - filter by character
func (h *AgentChatHandler) ListChatThreads(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	characterID := c.Query("characterId")

	threads, err := h.chatService.ListUserChatThreads(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.Error("failed to list chat threads", zap.Error(err))
		InternalError(c, "Failed to list chat threads")
		return
	}

	// Filter by characterId if provided
	if characterID != "" {
		filtered := make([]*domain.ChatThread, 0)
		for _, t := range threads {
			if t.CharacterID == characterID {
				filtered = append(filtered, t)
			}
		}
		threads = filtered
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
			SelectedStoryboardID: thread.SelectedStoryboardID,
			TotalTokensUsed:      thread.TotalTokensUsed,
			IsArchived:           thread.IsArchived,
			Summary:              thread.Summary,
			CreatedAt:            thread.CreatedAt,
			UpdatedAt:            thread.UpdatedAt,
		}
	}

	Success(c, gin.H{
		"threads": resp,
		"total":   len(resp),
	})
}

// CreateChatThread 创建聊天线程
// POST /api/agent/chat/threads
func (h *AgentChatHandler) CreateChatThread(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	var req CreateThreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, "Invalid request: "+err.Error())
		return
	}

	thread, err := h.chatService.CreateChatThread(c.Request.Context(), userID.(string), req.CharacterID)
	if err != nil {
		h.logger.Error("failed to create chat thread", zap.Error(err))
		InternalError(c, "Failed to create chat thread: "+err.Error())
		return
	}

	Success(c, ChatThreadResponse{
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
		SelectedStoryboardID: thread.SelectedStoryboardID,
		TotalTokensUsed:      thread.TotalTokensUsed,
		IsArchived:           thread.IsArchived,
		Summary:              thread.Summary,
		CreatedAt:            thread.CreatedAt,
		UpdatedAt:            thread.UpdatedAt,
	})
}

// ArchiveThread 归档聊天线程
// POST /api/agent/chat/threads/:id/archive
func (h *AgentChatHandler) ArchiveThread(c *gin.Context) {
	threadID := c.Param("id")
	if threadID == "" {
		InvalidParams(c, "Thread ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	// Verify thread belongs to user and archive it
	if err := h.chatService.ArchiveThread(c.Request.Context(), threadID, userID.(string)); err != nil {
		h.logger.Error("failed to archive thread", zap.Error(err))
		InternalError(c, "Failed to archive thread: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "Thread archived successfully"})
}

// GetAgent 获取角色的Agent信息
// GET /api/agent/character/:characterId
func (h *AgentChatHandler) GetAgent(c *gin.Context) {
	characterID := c.Param("characterId")
	if characterID == "" {
		InvalidParams(c, "Character ID is required")
		return
	}

	agent, err := h.chatService.GetAgentByCharacter(c.Request.Context(), characterID)
	if err != nil {
		h.logger.Error("failed to get agent", zap.Error(err))
		NotFound(c, "Agent not found")
		return
	}

	Success(c, agent)
}

// UpdateAgentConfig 更新Agent配置（管理员功能）
// PUT /api/agent/:agentId/config
func (h *AgentChatHandler) UpdateAgentConfig(c *gin.Context) {
	agentID := c.Param("agentId")
	if agentID == "" {
		InvalidParams(c, "Agent ID is required")
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
		InvalidParams(c, "Invalid request")
		return
	}

	// 获取现有Agent
	agent, err := h.chatService.GetAgentByCharacter(c.Request.Context(), agentID)
	if err != nil {
		NotFound(c, "Agent not found")
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
		InternalError(c, "Failed to update agent config")
		return
	}

	Success(c, gin.H{
		"message": "Agent config updated successfully",
		"agent":   agent,
	})
}

// SendMessageStream HTTP/2流式发送消息
// POST /api/agent/chat/send-stream
func (h *AgentChatHandler) SendMessageStream(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, "Invalid request: "+err.Error())
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	// 设置HTTP/2流式响应头
	c.Header("Content-Type", "application/json")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)

	// 获取流式数据
	chunkChan, messageChan, errChan := h.chatService.SendMessageWithStream(
		c.Request.Context(),
		userID.(string),
		req.CharacterID,
		req.Content,
		req.Image,
		req.ThreadID,
		req.StoryboardBranchID,
	)

	// 创建JSON编码器
	encoder := json.NewEncoder(c.Writer)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		InternalError(c, "Streaming not supported")
		return
	}

	// 发送流式数据块
	for {
		select {
		case chunk, ok := <-chunkChan:
			if !ok {
				chunkChan = nil
			} else {
				chunkData := map[string]interface{}{
					"type": "chunk",
					"data": chunk,
				}
				if err := encoder.Encode(chunkData); err != nil {
					h.logger.Error("failed to encode chunk", zap.Error(err))
					return
				}
				flusher.Flush()
			}

		case message, ok := <-messageChan:
			if !ok {
				messageChan = nil
			} else {
				completeData := map[string]interface{}{
					"type":    "complete",
					"message": message,
				}
				if err := encoder.Encode(completeData); err != nil {
					h.logger.Error("failed to encode complete message", zap.Error(err))
					return
				}
				flusher.Flush()
			}

		case err, ok := <-errChan:
			if !ok {
				errChan = nil
			} else {
				errorData := map[string]interface{}{
					"type":  "error",
					"error": err.Error(),
				}
				encoder.Encode(errorData)
				flusher.Flush()
				return
			}

		case <-c.Request.Context().Done():
			return
		}

		if chunkChan == nil && messageChan == nil && errChan == nil {
			break
		}
	}
}

// GetStoryboardBranches 获取可用的故事板分支
// GET /api/agent/chat/threads/:id/storyboard-branches
func (h *AgentChatHandler) GetStoryboardBranches(c *gin.Context) {
	threadID := c.Param("id")
	if threadID == "" {
		InvalidParams(c, "Thread ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	// 验证线程所有权（通过获取线程列表验证）
	threads, err := h.chatService.ListUserChatThreads(c.Request.Context(), userID.(string))
	if err != nil {
		InternalError(c, "Failed to get threads")
		return
	}

	var characterID string
	var found bool
	for _, t := range threads {
		if t.ID == threadID {
			characterID = t.CharacterID
			found = true
			break
		}
	}

	if !found {
		NotFound(c, "Thread not found")
		return
	}

	// 获取叶子节点
	leafNodes, err := h.chatService.GetStoryboardLeafNodes(c.Request.Context(), characterID)
	if err != nil {
		h.logger.Error("failed to get leaf nodes", zap.Error(err))
		InternalError(c, "Failed to get storyboard branches")
		return
	}

	Success(c, gin.H{
		"branches": leafNodes,
		"total":    len(leafNodes),
	})
}

// SelectStoryboardBranch 选择故事板分支
// POST /api/agent/chat/threads/:id/select-branch
func (h *AgentChatHandler) SelectStoryboardBranch(c *gin.Context) {
	threadID := c.Param("id")
	if threadID == "" {
		InvalidParams(c, "Thread ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	var req struct {
		StoryboardBranchID string `json:"storyboardBranchId" binding:"required"`
		CharacterID        string `json:"characterId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, "Invalid request: "+err.Error())
		return
	}

	// Verify thread belongs to user
	threads, err := h.chatService.ListUserChatThreads(c.Request.Context(), userID.(string))
	if err != nil {
		InternalError(c, "Failed to verify thread ownership")
		return
	}

	var found bool
	for _, t := range threads {
		if t.ID == threadID {
			found = true
			break
		}
	}

	if !found {
		Forbidden(c, "Thread does not belong to user")
		return
	}

	if err := h.chatService.SelectStoryboardBranch(c.Request.Context(), threadID, req.StoryboardBranchID, req.CharacterID); err != nil {
		h.logger.Error("failed to select branch", zap.Error(err))
		InternalError(c, "Failed to select branch: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "Branch selected successfully"})
}

// ReactToMessage 消息互动
// POST /api/agent/chat/messages/:id/react
func (h *AgentChatHandler) ReactToMessage(c *gin.Context) {
	messageID := c.Param("id")
	if messageID == "" {
		InvalidParams(c, "Message ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	var req struct {
		ReactionType string `json:"reactionType" binding:"required"` // like, dislike, emoji
		EmojiCode    string `json:"emojiCode,omitempty"`             // For emoji reactions
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.chatService.ReactToMessage(c.Request.Context(), messageID, userID.(string), req.ReactionType, req.EmojiCode); err != nil {
		h.logger.Error("failed to react to message", zap.Error(err))
		InternalError(c, "Failed to react: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "Reaction added successfully"})
}

// ArchiveMessage 归档消息
// POST /api/agent/chat/messages/:id/archive
func (h *AgentChatHandler) ArchiveMessage(c *gin.Context) {
	messageID := c.Param("id")
	if messageID == "" {
		InvalidParams(c, "Message ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	if err := h.chatService.ArchiveMessage(c.Request.Context(), messageID, userID.(string)); err != nil {
		h.logger.Error("failed to archive message", zap.Error(err))
		InternalError(c, "Failed to archive: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "Message archived successfully"})
}

// GetThreadStats 获取会话统计
// GET /api/agent/chat/threads/:id/stats
func (h *AgentChatHandler) GetThreadStats(c *gin.Context) {
	threadID := c.Param("id")
	if threadID == "" {
		InvalidParams(c, "Thread ID is required")
		return
	}

	stats, err := h.chatService.GetThreadStats(c.Request.Context(), threadID)
	if err != nil {
		h.logger.Error("failed to get stats", zap.Error(err))
		InternalError(c, "Failed to get stats: "+err.Error())
		return
	}

	Success(c, stats)
}

// LoadMoreMessages 加载更多消息
// GET /api/agent/chat/threads/:id/messages
func (h *AgentChatHandler) LoadMoreMessages(c *gin.Context) {
	threadID := c.Param("id")
	if threadID == "" {
		InvalidParams(c, "Thread ID is required")
		return
	}

	userID, exists := c.Get("userID")
	if !exists {
		Unauthorized(c, "Unauthorized")
		return
	}

	beforeMessageID := c.Query("before")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if beforeMessageID == "" {
		// 使用原有的GetChatHistory
		messages, err := h.chatService.GetChatHistory(c.Request.Context(), userID.(string), threadID, limit, 0)
		if err != nil {
			h.logger.Error("failed to get messages", zap.Error(err))
			InternalError(c, "Failed to get messages")
			return
		}

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

		Success(c, gin.H{
			"messages": resp,
			"total":    len(resp),
		})
		return
	}

	// 加载更多消息（before message）
	messages, err := h.chatService.LoadMoreMessages(c.Request.Context(), threadID, beforeMessageID, limit)
	if err != nil {
		h.logger.Error("failed to load more messages", zap.Error(err))
		InternalError(c, "Failed to load messages: "+err.Error())
		return
	}

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

	Success(c, gin.H{
		"messages": resp,
		"total":    len(resp),
	})
}

// RegisterRoutes 注册Agent聊天相关的路由
func (h *AgentChatHandler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	agentGroup := r.Group("/agent")
	agentGroup.Use(authMiddleware) // 需要认证

	// 聊天相关
	agentGroup.POST("/chat/send", h.SendMessage)
	agentGroup.POST("/chat/send-stream", h.SendMessageStream) // HTTP/2 streaming
	agentGroup.GET("/chat/history/:threadId", h.GetChatHistory)
	agentGroup.GET("/chat/threads", h.ListChatThreads)
	agentGroup.POST("/chat/threads", h.CreateChatThread) // Create new thread
	agentGroup.GET("/chat/threads/:id/storyboard-branches", h.GetStoryboardBranches)
	agentGroup.POST("/chat/threads/:id/select-branch", h.SelectStoryboardBranch)
	agentGroup.GET("/chat/threads/:id/messages", h.LoadMoreMessages)
	agentGroup.GET("/chat/threads/:id/stats", h.GetThreadStats)
	agentGroup.POST("/chat/threads/:id/archive", h.ArchiveThread) // Archive thread
	agentGroup.POST("/chat/messages/:id/react", h.ReactToMessage)
	agentGroup.POST("/chat/messages/:id/archive", h.ArchiveMessage)

	// Agent管理
	agentGroup.GET("/character/:characterId", h.GetAgent)
	agentGroup.PUT("/:agentId/config", h.UpdateAgentConfig) // 需要管理员权限

	// agent—ui 相关的接口
	agentGroup.GET("/agent-ui/chat/threads", h.ListChatThreads)

}

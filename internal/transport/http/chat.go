package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// ========== Chat Thread Handlers ==========

// ListChatThreads 获取用户的聊天列表
// GET /api/chats
func (h *Handler) ListChatThreads(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	threads, err := h.svc.ListChatThreads(c.Request.Context(), userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"threads": threads,
		"count":   len(threads),
	})
}

// GetChatThread 获取聊天线程详情
// GET /api/chats/:id
func (h *Handler) GetChatThread(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	threadID := c.Param("id")
	thread, err := h.svc.GetChatThread(c.Request.Context(), threadID, userID)
	if err != nil {
		NotFound(c, "chat thread not found")
		return
	}

	Success(c, thread)
}

// CreateChatThread 创建聊天线程（与角色开始聊天）
// POST /api/chats
// Body: { "characterId": "xxx" }
func (h *Handler) CreateChatThread(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		CharacterID string `json:"characterId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	thread, err := h.svc.CreateChatThread(c.Request.Context(), userID, req.CharacterID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, thread)
}

// DeleteChatThread 删除聊天线程
// DELETE /api/chats/:id
func (h *Handler) DeleteChatThread(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	threadID := c.Param("id")
	if err := h.svc.DeleteChatThread(c.Request.Context(), threadID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "chat thread deleted successfully"})
}

// MarkChatThreadAsRead 标记聊天为已读
// POST /api/chats/:id/read
func (h *Handler) MarkChatThreadAsRead(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	threadID := c.Param("id")
	if err := h.svc.MarkChatThreadAsRead(c.Request.Context(), threadID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "marked as read"})
}

// GetUnreadChatCount 获取未读聊天数
// GET /api/chats/unread/count
func (h *Handler) GetUnreadChatCount(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	count, err := h.svc.GetUnreadChatCount(c.Request.Context(), userID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"count": count})
}

// ========== Chat Message Handlers ==========

// ListChatMessages 获取聊天消息列表
// GET /api/chats/:id/messages
func (h *Handler) ListChatMessages(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	threadID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	messages, err := h.svc.ListChatMessages(c.Request.Context(), threadID, userID, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"messages": messages,
		"count":    len(messages),
	})
}

// SendChatMessage 发送聊天消息
// POST /api/chats/:id/messages
// Body: { "content": "message text", "image": "optional image url" }
func (h *Handler) SendChatMessage(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	threadID := c.Param("id")

	var req struct {
		Content string `json:"content" binding:"required"`
		Image   string `json:"image"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	message, err := h.svc.SendChatMessage(c.Request.Context(), threadID, userID, req.Content, req.Image)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, message)
}

// DeleteChatMessage 删除聊天消息
// DELETE /api/chats/:threadId/messages/:messageId
func (h *Handler) DeleteChatMessage(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	messageID := c.Param("messageId")
	if err := h.svc.DeleteChatMessage(c.Request.Context(), messageID, userID); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "chat message deleted successfully"})
}

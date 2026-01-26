package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ========== Writers Room Handlers ==========

// GetOrCreateWritersRoom gets or creates a writers room for a story
// GET /api/stories/:storyId/writers-room
func (h *Handler) GetOrCreateWritersRoom(c *gin.Context) {
	_, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID, ok := RequireParam(c, "storyId")
	if !ok {
		return
	}

	room, err := h.writersRoomSvc.GetOrCreateRoom(c.Request.Context(), storyID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, room)
}

// GetWritersRoom gets a writers room details
// GET /api/writers-rooms/:roomId
func (h *Handler) GetWritersRoom(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	roomID, ok := RequireParam(c, "roomId")
	if !ok {
		return
	}

	room, err := h.writersRoomSvc.GetRoom(c.Request.Context(), roomID, userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, room)
}

// GetWritersRoomMessages gets messages in a writers room
// GET /api/writers-rooms/:roomId/messages?limit=50&offset=0
func (h *Handler) GetWritersRoomMessages(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	roomID, ok := RequireParam(c, "roomId")
	if !ok {
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	response, err := h.writersRoomSvc.ListMessages(c.Request.Context(), roomID, userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, response)
}

// SendWritersRoomMessage sends a message to a writers room
// POST /api/writers-rooms/:roomId/messages
// Body: { "content": "message", "messageType": "text", "attachments": [], "mentions": [], "replyToMessageId": "" }
func (h *Handler) SendWritersRoomMessage(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	roomID, ok := RequireParam(c, "roomId")
	if !ok {
		return
	}

	var req domain.WritersRoomSendMessageRequest
	if !BindJSON(c, &req) {
		return
	}

	req.RoomID = roomID

	message, err := h.writersRoomSvc.SendMessage(c.Request.Context(), &req, userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, message)
}

// DeleteWritersRoomMessage deletes a message from a writers room
// DELETE /api/writers-rooms/:roomId/messages/:messageId
func (h *Handler) DeleteWritersRoomMessage(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	_, ok = RequireParam(c, "roomId")
	if !ok {
		return
	}

	messageID, ok := RequireParam(c, "messageId")
	if !ok {
		return
	}

	if err := h.writersRoomSvc.DeleteMessage(c.Request.Context(), messageID, userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "message deleted successfully"})
}

// MarkWritersRoomMessageAsRead marks a message as read
// POST /api/writers-rooms/:roomId/messages/:messageId/read
func (h *Handler) MarkWritersRoomMessageAsRead(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	_, ok = RequireParam(c, "roomId")
	if !ok {
		return
	}

	messageID, ok := RequireParam(c, "messageId")
	if !ok {
		return
	}

	if err := h.writersRoomSvc.MarkMessageAsRead(c.Request.Context(), messageID, userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "message marked as read"})
}

// MarkAllWritersRoomMessagesAsRead marks all messages in a room as read
// POST /api/writers-rooms/:roomId/read-all
func (h *Handler) MarkAllWritersRoomMessagesAsRead(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	roomID, ok := RequireParam(c, "roomId")
	if !ok {
		return
	}

	if err := h.writersRoomSvc.MarkAllAsRead(c.Request.Context(), roomID, userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "all messages marked as read"})
}

// AddWritersRoomMessageReaction adds a reaction to a message
// POST /api/writers-rooms/:roomId/messages/:messageId/reactions
// Body: { "reactionType": "emoji", "emojiCode": "👍" }
func (h *Handler) AddWritersRoomMessageReaction(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	_, ok = RequireParam(c, "roomId")
	if !ok {
		return
	}

	messageID, ok := RequireParam(c, "messageId")
	if !ok {
		return
	}

	var req struct {
		ReactionType string `json:"reactionType" binding:"required"`
		EmojiCode     string `json:"emojiCode"`
	}
	if !BindJSON(c, &req) {
		return
	}

	if err := h.writersRoomSvc.AddReaction(c.Request.Context(), messageID, userID, req.ReactionType, req.EmojiCode); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "reaction added successfully"})
}

// RemoveWritersRoomMessageReaction removes a reaction from a message
// DELETE /api/writers-rooms/:roomId/messages/:messageId/reactions
func (h *Handler) RemoveWritersRoomMessageReaction(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	_, ok = RequireParam(c, "roomId")
	if !ok {
		return
	}

	messageID, ok := RequireParam(c, "messageId")
	if !ok {
		return
	}

	if err := h.writersRoomSvc.RemoveReaction(c.Request.Context(), messageID, userID); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "reaction removed successfully"})
}

// GetWritersRoomUnreadCount gets unread message count for a user in a room
// GET /api/writers-rooms/:roomId/unread-count
func (h *Handler) GetWritersRoomUnreadCount(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	roomID, ok := RequireParam(c, "roomId")
	if !ok {
		return
	}

	count, err := h.writersRoomSvc.UnreadCount(c.Request.Context(), roomID, userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"unreadCount": count})
}

// RegisterWritersRoomRoutes registers writers room routes
func (h *Handler) RegisterWritersRoomRoutes(storiesGroup *gin.RouterGroup, roomsGroup *gin.RouterGroup) {
	// Story-related routes
	storiesGroup.GET("/:storyId/writers-room", h.GetOrCreateWritersRoom)

	// Room-related routes
	roomsGroup.GET("/:roomId", h.GetWritersRoom)
	roomsGroup.GET("/:roomId/messages", h.GetWritersRoomMessages)
	roomsGroup.GET("/:roomId/unread-count", h.GetWritersRoomUnreadCount)
	roomsGroup.POST("/:roomId/messages", h.SendWritersRoomMessage)
	roomsGroup.DELETE("/:roomId/messages/:messageId", h.DeleteWritersRoomMessage)
	roomsGroup.POST("/:roomId/messages/:messageId/read", h.MarkWritersRoomMessageAsRead)
	roomsGroup.POST("/:roomId/read-all", h.MarkAllWritersRoomMessagesAsRead)
	roomsGroup.POST("/:roomId/messages/:messageId/reactions", h.AddWritersRoomMessageReaction)
	roomsGroup.DELETE("/:roomId/messages/:messageId/reactions", h.RemoveWritersRoomMessageReaction)
}

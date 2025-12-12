package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ListNotifications(c *gin.Context) {
	userID, _ := c.Get("userID")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, err := h.svc.ListNotifications(c.Request.Context(), userID.(string), limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"notifications": notifications, "count": len(notifications)})
}

func (h *Handler) UnreadCount(c *gin.Context) {
	userID, _ := c.Get("userID")
	count, err := h.svc.UnreadCount(c.Request.Context(), userID.(string))
	if err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"count": count})
}

func (h *Handler) MarkAsRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	if err := h.svc.MarkAsRead(c.Request.Context(), id, userID.(string)); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "marked as read"})
}

func (h *Handler) MarkAllAsRead(c *gin.Context) {
	userID, _ := c.Get("userID")
	if err := h.svc.MarkAllAsRead(c.Request.Context(), userID.(string)); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "all marked as read"})
}

func (h *Handler) DeleteNotification(c *gin.Context) {
	userID, _ := c.Get("userID")
	id := c.Param("id")
	if err := h.svc.DeleteNotification(c.Request.Context(), id, userID.(string)); err != nil {
		InternalError(c, err.Error())
		return
	}
	Success(c, gin.H{"message": "notification deleted"})
}

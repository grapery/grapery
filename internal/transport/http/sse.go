package http

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// SSEChatStream 实时聊天消息流（SSE）
// GET /api/sse/chat/:threadId
func (h *Handler) SSEChatStream(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	threadID := c.Param("threadId")

	// 验证用户是否有权限访问该聊天线程
	_, err := h.svc.GetChatThread(c.Request.Context(), threadID, userID)
	if err != nil {
		NotFound(c, "chat thread not found")
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // 禁用 Nginx 缓冲

	// 发送初始连接成功消息
	fmt.Fprintf(c.Writer, "event: connected\ndata: {\"threadId\":\"%s\",\"status\":\"connected\"}\n\n", threadID)
	c.Writer.Flush()

	// 创建一个ticker来定期检查新消息
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 发送心跳的ticker
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// 记录最后一条消息的时间
	var lastMessageTime int64
	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			// 客户端断开连接
			return

		case <-heartbeat.C:
			// 发送心跳保持连接
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()

		case <-ticker.C:
			// 检查新消息
			messages, err := h.svc.ListChatMessages(c.Request.Context(), threadID, userID, 10, 0)
			if err != nil {
				continue
			}

			// 发送新消息
			for _, msg := range messages {
				if msg.Timestamp > lastMessageTime {
					messageData, _ := json.Marshal(msg)
					fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", messageData)
					c.Writer.Flush()
					lastMessageTime = msg.Timestamp
				}
			}
		}
	}
}

// SSENotificationStream 实时通知流（SSE）
// GET /api/sse/notifications
func (h *Handler) SSENotificationStream(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 发送初始连接成功消息
	fmt.Fprintf(c.Writer, "event: connected\ndata: {\"status\":\"connected\",\"userId\":\"%s\"}\n\n", userID)
	c.Writer.Flush()

	// 创建ticker来定期检查新通知
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	var lastCheckTime int64 = time.Now().Unix()
	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			return

		case <-heartbeat.C:
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()

		case <-ticker.C:
			// 检查新通知
			notifications, err := h.svc.ListNotifications(c.Request.Context(), userID, 20, 0)
			if err != nil {
				continue
			}

			// 发送新通知
			for _, notif := range notifications {
				if notif.CreatedAt > lastCheckTime {
					notifData, _ := json.Marshal(notif)
					fmt.Fprintf(c.Writer, "event: notification\ndata: %s\n\n", notifData)
					c.Writer.Flush()
				}
			}
			lastCheckTime = time.Now().Unix()

			// 同时发送未读计数
			unreadCount, _ := h.svc.UnreadCount(c.Request.Context(), userID)
			countData := map[string]int{"unreadCount": unreadCount}
			countJSON, _ := json.Marshal(countData)
			fmt.Fprintf(c.Writer, "event: unread_count\ndata: %s\n\n", countJSON)
			c.Writer.Flush()
		}
	}
}

// SSEActivityStream 活动流（SSE）
// GET /api/sse/activities
func (h *Handler) SSEActivityStream(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	fmt.Fprintf(c.Writer, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
	c.Writer.Flush()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	ctx := c.Request.Context()

	for {
		select {
		case <-ctx.Done():
			return

		case <-heartbeat.C:
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()

		case <-ticker.C:
			// 获取最新活动
			activities, err := h.svc.GetUserActivities(c.Request.Context(), userID, 10)
			if err != nil {
				continue
			}

			if len(activities) > 0 {
				activitiesData, _ := json.Marshal(activities)
				fmt.Fprintf(c.Writer, "event: activities\ndata: %s\n\n", activitiesData)
				c.Writer.Flush()
			}
		}
	}
}

// ========== SSE Helper Functions ==========

// SSEEvent SSE事件结构
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
	ID    string      `json:"id,omitempty"`
	Retry int         `json:"retry,omitempty"`
}

// SendSSEEvent 发送SSE事件
func SendSSEEvent(c *gin.Context, event string, data interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, dataJSON)
	c.Writer.Flush()
	return nil
}

// SendSSEHeartbeat 发送心跳
func SendSSEHeartbeat(c *gin.Context) {
	fmt.Fprintf(c.Writer, ": heartbeat\n\n")
	c.Writer.Flush()
}

// ========== HTTP/2 Server Push Support ==========

// PushChatMessage 使用 HTTP/2 Server Push 推送聊天消息
func (h *Handler) PushChatMessage(c *gin.Context) {
	// HTTP/2 Server Push 示例
	// 这需要在支持 HTTP/2 的服务器上运行
	if pusher, ok := c.Writer.(gin.ResponseWriter); ok {
		// 推送相关资源
		_ = pusher
		// pusher.Push("/api/chats/:id/messages", nil)
	}
}

// ========== Long Polling Fallback ==========

// LongPollChatMessages Long Polling 方式获取新消息（SSE的降级方案）
// GET /api/poll/chat/:threadId/messages
func (h *Handler) LongPollChatMessages(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	threadID := c.Param("threadId")
	lastMessageID := c.Query("lastMessageId")
	timeout := 30 * time.Second

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// 超时，返回空结果
			Success(c, gin.H{
				"messages": []interface{}{},
				"hasNew":   false,
			})
			return

		case <-ticker.C:
			// 检查新消息
			messages, err := h.svc.ListChatMessages(c.Request.Context(), threadID, userID, 10, 0)
			if err != nil {
				continue
			}

			// 如果有新消息，立即返回
			if len(messages) > 0 {
				latestID := messages[0].ID
				if latestID != lastMessageID {
					Success(c, gin.H{
						"messages": messages,
						"hasNew":   true,
					})
					return
				}
			}
		}
	}
}

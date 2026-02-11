package http

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

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
			// V1/V2 MVP: 活动流已移除 Group 相关功能
			// 未来 V2 可能会添加点赞、评论、关注等社交活动
			// 目前发送空的活动列表保持连接
			activities := []interface{}{}
			activitiesData, _ := json.Marshal(activities)
			fmt.Fprintf(c.Writer, "event: activities\ndata: %s\n\n", activitiesData)
			c.Writer.Flush()
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

// ========== Long Polling Support ==========

// LongPollMessages Long Polling 方式获取新消息（SSE的降级方案）
// 注意：此功能已废弃，保留作为占位符
func (h *Handler) LongPollMessages(c *gin.Context) {
	// 返回空结果，功能已废弃
	Success(c, gin.H{
		"messages": []interface{}{},
		"hasNew":   false,
	})
}

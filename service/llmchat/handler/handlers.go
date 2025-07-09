package llmchathandler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	llmchatpkg "github.com/grapery/grapery/pkg/llmchat"
	llmchatservice "github.com/grapery/grapery/service/llmchat"
	"github.com/grapery/grapery/service/llmchat/middleware"
)

// RegisterLLMChatRoutes 注册llmchat相关路由，使用鉴权和限流中间件
func RegisterLLMChatRoutes(r *gin.Engine) {
	api := r.Group("/api/llmchat")
	api.Use(middleware.AuthMiddleware())
	api.Use(middleware.RateLimitMiddleware())
	{
		api.POST("/session", CreateSessionHandler)
		api.POST("/message", SendMessageHandler)
		api.POST("/message/:id/retry", RetryMessageHandler)
		api.POST("/message/:id/feedback", FeedbackMessageHandler)
		api.POST("/message/:id/interrupt", InterruptMessageHandler)
	}
}

// CreateSessionHandler 创建会话
func CreateSessionHandler(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	userID := c.GetInt64("userID")
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := llmchatservice.CreateSessionService(c.Request.Context(), userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// SendMessageHandler 发送消息，支持SSE
func SendMessageHandler(c *gin.Context) {
	var req struct {
		SessionID int64  `json:"session_id" binding:"required"`
		Content   string `json:"content" binding:"required"`
		ParentID  *int64 `json:"parent_id"`
	}
	userID := c.GetInt64("userID")
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if c.GetHeader("Accept") == "text/event-stream" {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()
		ctx := c.Request.Context()
		_ = llmchatpkg.LLMStreamChat(ctx, req.Content, func(chunk string) error {
			_, err := c.Writer.Write([]byte("data: " + chunk + "\n\n"))
			c.Writer.Flush()
			return err
		})
		c.Writer.Write([]byte("event: done\ndata: [DONE]\n\n"))
		c.Writer.Flush()
		return
	}
	res, err := llmchatservice.SendMessageService(c.Request.Context(), req.SessionID, userID, req.Content, req.ParentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// RetryMessageHandler 重试消息，支持SSE
func RetryMessageHandler(c *gin.Context) {
	var req struct {
		SessionID int64  `json:"session_id" binding:"required"`
		Msg       string `json:"msg" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if c.GetHeader("Accept") == "text/event-stream" {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()
		ctx := c.Request.Context()
		_ = llmchatpkg.LLMStreamChat(ctx, req.Msg, func(chunk string) error {
			_, err := c.Writer.Write([]byte("data: " + chunk + "\n\n"))
			c.Writer.Flush()
			return err
		})
		c.Writer.Write([]byte("event: done\ndata: [DONE]\n\n"))
		c.Writer.Flush()
		return
	}
	c.JSON(http.StatusNotImplemented, gin.H{"msg": "仅支持SSE流式重试"})
}

// FeedbackMessageHandler 消息反馈
func FeedbackMessageHandler(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}
	var req struct {
		Type    string `json:"type" binding:"required"` // like/dislike/complaint
		Content string `json:"content"`
	}
	userID := c.GetInt64("userID")
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := llmchatservice.FeedbackMessageService(c.Request.Context(), msgID, userID, req.Type, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// InterruptMessageHandler 中断消息
func InterruptMessageHandler(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid message id"})
		return
	}
	err = llmchatservice.InterruptMessageService(c.Request.Context(), msgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"msg": "interrupted"})
}

package llmchathandler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/cloud/coze"
	llmchatpkg "github.com/grapery/grapery/pkg/llmchat"
	llmchatservice "github.com/grapery/grapery/service/llmchat"
	"github.com/grapery/grapery/service/llmchat/middleware"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

// 统一API响应结构体
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// SSEPayload 用于SSE推送的结构体，包含event和data字段，便于前端解析
type SSEPayload struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// RegisterLLMChatRoutes 注册llmchat相关路由，使用鉴权和限流中间件
func RegisterLLMChatRoutes(r *gin.Engine) {
	api := r.Group("/api/llmchat")
	//api.Use(middleware.AuthMiddleware())
	api.Use(middleware.RateLimitMiddleware())
	{
		api.POST("/session", CreateSessionHandler)
		api.POST("/role/session", CreateRoleSessionHandler)
		api.POST("/session/:id/messages", SessionMessageHandler)
		api.POST("/role/session/:id/clear", ClearSessionHandler)
		api.POST("/message", SendMessageHandler)
		api.POST("/message/:id/retry", RetryMessageHandler)
		api.POST("/message/:id/feedback", FeedbackMessageHandler)
		api.POST("/message/:id/interrupt", InterruptMessageHandler)
	}
}

// CreateSessionHandler 创建会话
func CreateSessionHandler(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		RoleId string `json:"role_id" binding:"required"`
		BotId  string `json:"bot_id" binding:"required"`
	}
	userID := c.GetInt64("userID")
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	res, err := llmchatservice.CreateSessionService(c.Request.Context(), userID, req.Name, req.RoleId, req.BotId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: res})
}

// CreateRoleSessionHandler 创建角色会话
func CreateRoleSessionHandler(c *gin.Context) {
	var req struct {
		RoleID int64  `json:"role_id" binding:"required"`
		UserID int64  `json:"user_id" binding:"required"`
		Title  string `json:"title" binding:"required"`
		Desc   string `json:"desc" binding:"required"`
	}
	userID := c.GetInt64("userID")
	if userID != req.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "user_id not match"})
		return
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
}

func ClearSessionHandler(c *gin.Context) {
	sessionID := c.Param("id")
	err := llmchatservice.ClearSessionService(c.Request.Context(), sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: struct{}{}})
}

// SendMessageHandler 发送消息，支持SSE
func SendMessageHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[SendMessageHandler] handler入口")
	var req struct {
		SessionID string `json:"session_id" binding:"required"`
		Content   string `json:"content" binding:"required"`
	}
	userID := c.GetInt64("userID")
	log.Info("[SendMessageHandler] 解析参数", zap.Int64("userID", userID))
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[SendMessageHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[SendMessageHandler] 参数绑定成功", zap.String("session_id", req.SessionID), zap.String("content", req.Content))
	res, err := llmchatservice.SendMessageService(c.Request.Context(), req.SessionID, userID, req.Content)
	if err != nil {
		log.Error("[SendMessageHandler] SendMessageService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	msg, _ := res.(*models.LLMChatMsg)
	log.Info("[SendMessageHandler] SendMessageService成功", zap.String("session_id", req.SessionID), zap.String("message_id", msg.MessageId), zap.Int64("userID", userID))
	// 设置SSE响应头，必须在最前面
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()
	ctx := c.Request.Context()
	streamChan := make(chan string, 100)
	answerMap := make(map[string][]coze.AnswerOrFollowUp)
	log.Info("[SendMessageHandler] 启动流式SendMessageStream", zap.String("session_id", req.SessionID), zap.String("message_id", msg.MessageId), zap.Int64("userID", userID), zap.String("content", req.Content))
	go func() {
		err = llmchatpkg.GetLLMChatEngine().SendMessageStream(ctx, req.SessionID, msg.MessageId, userID, req.Content, streamChan, answerMap)
		if err != nil {
			log.Error("[SendMessageHandler] SendMessageStream失败", zap.Error(err))
			c.Writer.Write([]byte("event: error\ndata: " + err.Error() + "\n\n"))
			c.Writer.Flush()
			return
		}
		log.Info("[SendMessageHandler] SendMessageStream成功")
	}()
	log.Info("[SendMessageHandler] 流式通道启动成功，开始推送SSE数据")
	for {
		chunk, ok := <-streamChan
		if !ok {
			log.Info("[SendMessageHandler] streamChan已关闭，结束SSE推送")
			// 推送结束事件，event为done，data为[DONE]
			payload := SSEPayload{
				Event: "done",
				Data:  "[DONE]",
			}
			b, _ := json.Marshal(payload)
			c.Writer.Write(b) // SSE事件块分隔
			c.Writer.Write([]byte("\n\n"))
			c.Writer.Flush()
			break
		}
		log.Info("[SendMessageHandler] 推送SSE chunk", zap.String("chunk", chunk))
		// 普通流式内容，event为conversation.message.delta
		payload := SSEPayload{
			Event: "conversation.message.delta",
			Data:  chunk,
		}
		b, _ := json.Marshal(payload)
		c.Writer.Write(b) // SSE事件块分隔
		c.Writer.Write([]byte("\n\n"))
		c.Writer.Flush()
	}
	log.Info("[SendMessageHandler] SSE流结束，推送event: done")
	c.Writer.Write([]byte("event: done\ndata: [DONE]\n\n"))
	c.Writer.Flush()
	log.Info("[SendMessageHandler] SSE流式响应完成，answerMap收集内容", zap.Any("answerMap", answerMap))
	// 统一结构体返回
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: res})
}

// RetryMessageHandler 重试消息，支持SSE
func RetryMessageHandler(c *gin.Context) {
	var req struct {
		SessionID int64  `json:"session_id" binding:"required"`
		Msg       string `json:"msg" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	if c.GetHeader("Accept") == "text/event-stream" {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Flush()
		c.Writer.Write([]byte("event: error\ndata: 暂不支持流式重试\n\n"))
		c.Writer.Flush()
		return
	}
	c.JSON(http.StatusNotImplemented, APIResponse{Code: http.StatusNotImplemented, Message: "仅支持SSE流式重试", Data: struct{}{}})
}

// FeedbackMessageHandler 消息反馈
func FeedbackMessageHandler(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: "invalid message id", Data: struct{}{}})
		return
	}
	var req struct {
		Type    string `json:"type" binding:"required"` // like/dislike/complaint
		Content string `json:"content"`
	}
	userID := c.GetInt64("userID")
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	res, err := llmchatservice.FeedbackMessageService(c.Request.Context(), msgID, userID, req.Type, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: res})
}

// InterruptMessageHandler 中断消息
func InterruptMessageHandler(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: "invalid message id", Data: struct{}{}})
		return
	}
	err = llmchatservice.InterruptMessageService(c.Request.Context(), msgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "interrupted", Data: struct{}{}})
}

func SessionMessageHandler(c *gin.Context) {
	sessionID := c.Param("id")
	params := c.Request.URL.Query()
	page, _ := strconv.Atoi(params.Get("page"))
	pageSize, _ := strconv.Atoi(params.Get("page_size"))
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}
	msgs, hasMore, err := llmchatservice.SessionMessageService(c.Request.Context(), sessionID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: gin.H{"msgs": msgs, "has_more": hasMore}})
}

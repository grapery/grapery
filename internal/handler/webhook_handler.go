package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// WebhookHandler 处理来自 AI 提供商的 Webhook 回调
type WebhookHandler struct {
	asyncVideoCompletion *service.AsyncVideoCompletionService
	logger               *zap.Logger
	webhookSecret        string // 用于验证 webhook 签名
}

// NewWebhookHandler 创建 Webhook 处理器
func NewWebhookHandler(asyncVideoCompletion *service.AsyncVideoCompletionService, logger *zap.Logger, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		asyncVideoCompletion: asyncVideoCompletion,
		logger:               logger,
		webhookSecret:        webhookSecret,
	}
}

// VideoCompletionWebhookRequest 视频完成 Webhook 请求
type VideoCompletionWebhookRequest struct {
	TaskID    string `json:"task_id" binding:"required"`
	Status    string `json:"status" binding:"required"`
	VideoURL  string `json:"video_url,omitempty"`
	Error     string `json:"error,omitempty"`
	Progress  int    `json:"progress,omitempty"`
	Provider  string `json:"provider,omitempty"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Signature string `json:"signature,omitempty"` // HMAC 签名
}

// VideoCompletionWebhook 处理视频完成 Webhook
// POST /webhooks/video-completion
func (h *WebhookHandler) VideoCompletionWebhook(c *gin.Context) {
	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read webhook request body",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// 解析请求
	var req VideoCompletionWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("failed to parse webhook request",
			zap.Error(err),
			zap.String("body", string(body)))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse request"})
		return
	}

	h.logger.Info("received video completion webhook",
		zap.String("taskID", req.TaskID),
		zap.String("status", req.Status),
		zap.String("videoURL", req.VideoURL),
		zap.String("provider", req.Provider))

	// 验证签名（如果配置了密钥）
	if h.webhookSecret != "" {
		expectedSig := generateSignature(body, h.webhookSecret)
		if !hmacEqual([]byte(req.Signature), []byte(expectedSig)) {
			h.logger.Warn("invalid webhook signature",
				zap.String("taskID", req.TaskID),
				zap.String("received", req.Signature),
				zap.String("expected", expectedSig))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
	}

	// 处理不同状态
	switch req.Status {
	case "completed", "success":
		// 视频生成成功
		h.handleVideoCompletion(c, &req)
	case "failed", "error":
		// 视频生成失败
		h.handleVideoFailure(c, &req)
	default:
		// 其他状态（如 processing），返回 202 Accepted
		c.JSON(http.StatusAccepted, gin.H{
			"message": "webhook received",
			"taskID":  req.TaskID,
			"status":  req.Status,
		})
	}
}

// handleVideoCompletion 处理视频完成
func (h *WebhookHandler) handleVideoCompletion(c *gin.Context, req *VideoCompletionWebhookRequest) {
	// 这里我们需要从异步完成服务中找到对应的任务
	// 并更新其状态

	h.logger.Info("video generation completed via webhook",
		zap.String("taskID", req.TaskID),
		zap.String("videoURL", req.VideoURL))

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"message": "video completion processed",
		"taskID":  req.TaskID,
		"status":  "completed",
	})

	// 注意：实际的 token 扣减和记录更新应该由轮询服务处理
	// Webhook 只是一个通知机制，可以用来加速处理
	// 如果需要立即处理，可以在这里调用 asyncVideoCompletion 的处理方法
}

// handleVideoFailure 处理视频失败
func (h *WebhookHandler) handleVideoFailure(c *gin.Context, req *VideoCompletionWebhookRequest) {
	h.logger.Warn("video generation failed via webhook",
		zap.String("taskID", req.TaskID),
		zap.String("error", req.Error))

	// 返回成功响应（已接收失败通知）
	c.JSON(http.StatusOK, gin.H{
		"message": "video failure processed",
		"taskID":  req.TaskID,
		"status":  "failed",
	})

	// 注意：实际的处理应该由轮询服务处理
}

// HealthCheck Webhook 健康检查
// GET /webhooks/health
func (h *WebhookHandler) HealthCheck(c *gin.Context) {
	pollingTasks := h.asyncVideoCompletion.GetPollingTasks()

	c.JSON(http.StatusOK, gin.H{
		"status":          "healthy",
		"timestamp":       time.Now().Unix(),
		"pollingTaskCount": len(pollingTasks),
	})
}

// generateSignature 生成 HMAC 签名
func generateSignature(data []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// hmacEqual 安全比较 HMAC，防止时序攻击
func hmacEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}

	return result == 0
}

// ProviderWebhook 通用提供商 Webhook
// POST /webhooks/:provider
func (h *WebhookHandler) ProviderWebhook(c *gin.Context) {
	provider := c.Param("provider")

	h.logger.Info("received provider webhook",
		zap.String("provider", provider))

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("failed to read webhook request body",
			zap.String("provider", provider),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// 根据不同的提供商处理不同的 webhook 格式
	switch provider {
	case "hailuo", "hailuo-ai":
		h.handleHailuoWebhook(c, body)
	case "huoshan", "volcengine":
		h.handleHuoshanWebhook(c, body)
	case "gemini", "google":
		h.handleGeminiWebhook(c, body)
	default:
		h.logger.Warn("unknown provider webhook",
			zap.String("provider", provider))
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown provider"})
	}
}

// handleHailuoWebhook 处理 Hailuo Webhook
func (h *WebhookHandler) handleHailuoWebhook(c *gin.Context, body []byte) {
	var req VideoCompletionWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("failed to parse hailuo webhook",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse request"})
		return
	}

	req.Provider = "hailuo"
	h.handleVideoCompletion(c, &req)
}

// handleHuoshanWebhook 处理火山引擎 Webhook
func (h *WebhookHandler) handleHuoshanWebhook(c *gin.Context, body []byte) {
	var req VideoCompletionWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("failed to parse huoshan webhook",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse request"})
		return
	}

	req.Provider = "huoshan"
	h.handleVideoCompletion(c, &req)
}

// handleGeminiWebhook 处理 Gemini Webhook
func (h *WebhookHandler) handleGeminiWebhook(c *gin.Context, body []byte) {
	var req VideoCompletionWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("failed to parse gemini webhook",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse request"})
		return
	}

	req.Provider = "gemini"
	h.handleVideoCompletion(c, &req)
}

// TestWebhook 测试 Webhook 端点（用于测试）
// POST /webhooks/test
func (h *WebhookHandler) TestWebhook(c *gin.Context) {
	var req VideoCompletionWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("test webhook received",
		zap.String("taskID", req.TaskID),
		zap.String("status", req.Status))

	// 获取当前轮询任务信息
	pollingTasks := h.asyncVideoCompletion.GetPollingTasks()

	c.JSON(http.StatusOK, gin.H{
		"message":         "test webhook received",
		"received":        req,
		"pollingTaskCount": len(pollingTasks),
		"timestamp":       time.Now().Unix(),
	})
}

// SetupWebhookRoutes 设置 Webhook 路由
func SetupWebhookRoutes(router *gin.RouterGroup, handler *WebhookHandler) {
	webhooks := router.Group("/webhooks")
	{
		// 视频完成 Webhook
		webhooks.POST("/video-completion", handler.VideoCompletionWebhook)

		// 通用提供商 Webhook
		webhooks.POST("/:provider", handler.ProviderWebhook)

		// 测试 Webhook
		webhooks.POST("/test", handler.TestWebhook)

		// 健康检查
		webhooks.GET("/health", handler.HealthCheck)
	}
}

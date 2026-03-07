package pay

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/sirupsen/logrus"
)

// UsageMiddleware Token用量检查中间件
type UsageMiddleware struct {
	tokenUsageService TokenUsageService
	logger            *logrus.Logger
}

// NewUsageMiddleware 创建用量检查中间件
func NewUsageMiddleware(tokenUsageService TokenUsageService, logger *logrus.Logger) *UsageMiddleware {
	return &UsageMiddleware{
		tokenUsageService: tokenUsageService,
		logger:            logger,
	}
}

// CheckUsageLimit 检查用量限制的中间件
func (m *UsageMiddleware) CheckUsageLimit(usageType paymodels.TokenUsageType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从上下文获取用户ID
		userIDInterface, exists := c.Get("userID")
		if !exists {
			m.logger.Error("User ID not found in context")
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "User not authenticated",
			})
			c.Abort()
			return
		}

		userID, ok := userIDInterface.(int64)
		if !ok {
			m.logger.Error("Invalid user ID type in context")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "Invalid user ID",
			})
			c.Abort()
			return
		}

		// 检查用量限制
		permission, err := m.tokenUsageService.CanPerformAction(c.Request.Context(), userID, usageType)
		if err != nil {
			m.logger.WithError(err).Error("Failed to check usage limit")
			c.JSON(http.StatusInternalServerError, gin.H{
				"code": 500,
				"msg":  "Failed to check usage limit",
			})
			c.Abort()
			return
		}

		if !permission.CanPerform {
			m.logger.WithFields(logrus.Fields{
				"user_id":    userID,
				"usage_type": usageType,
				"reason":     permission.Reason,
			}).Warn("Usage limit exceeded")

			// 返回用量超限错误，但不影响会员状态
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":               429,
				"msg":                "Usage limit exceeded",
				"error_code":         permission.ErrorCode,
				"error_message":      permission.ErrorMessage,
				"upgrade_suggestion": permission.UpgradeSuggestion,
				"usage_type":         usageType,
			})
			c.Abort()
			return
		}

		// 将权限信息存储到上下文，供后续处理使用
		c.Set("usage_permission", permission)
		c.Next()
	}
}

// RecordUsage 记录用量的中间件
func (m *UsageMiddleware) RecordUsage(usageType paymodels.TokenUsageType) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 执行后续处理
		c.Next()

		// 从上下文获取用户ID
		userIDInterface, exists := c.Get("userID")
		if !exists {
			return
		}

		userID, ok := userIDInterface.(int64)
		if !ok {
			return
		}

		// 检查响应状态
		if c.Writer.Status() >= 400 {
			// 记录失败请求
			err := m.tokenUsageService.RecordFailedRequest(c.Request.Context(), userID, usageType)
			if err != nil {
				m.logger.WithError(err).Error("Failed to record failed request")
			}
			return
		}

		// 从响应中获取Token使用量（需要业务逻辑提供）
		inputTokens, outputTokens, modelName, featureName := m.extractUsageFromResponse(c)

		// 记录成功请求的Token使用
		if inputTokens > 0 || outputTokens > 0 {
			err := m.tokenUsageService.RecordTokenUsage(c.Request.Context(), userID, usageType, inputTokens, outputTokens, modelName, featureName)
			if err != nil {
				m.logger.WithError(err).Error("Failed to record token usage")
			}
		}
	}
}

// extractUsageFromResponse 从响应中提取用量信息
func (m *UsageMiddleware) extractUsageFromResponse(c *gin.Context) (inputTokens, outputTokens int64, modelName, featureName string) {
	// 这里需要根据具体的业务响应格式来解析
	// 示例实现，实际使用时需要根据API响应结构调整

	// 尝试从响应头获取用量信息
	if inputTokensStr := c.GetHeader("X-Input-Tokens"); inputTokensStr != "" {
		if tokens, err := strconv.ParseInt(inputTokensStr, 10, 64); err == nil {
			inputTokens = tokens
		}
	}

	if outputTokensStr := c.GetHeader("X-Output-Tokens"); outputTokensStr != "" {
		if tokens, err := strconv.ParseInt(outputTokensStr, 10, 64); err == nil {
			outputTokens = tokens
		}
	}

	if model := c.GetHeader("X-Model-Name"); model != "" {
		modelName = model
	}

	if feature := c.GetHeader("X-Feature-Name"); feature != "" {
		featureName = feature
	}

	// 如果没有从头部获取到，尝试从响应体获取
	if inputTokens == 0 && outputTokens == 0 {
		// 这里可以解析响应体来获取用量信息
		// 具体实现取决于业务API的响应格式
	}

	return inputTokens, outputTokens, modelName, featureName
}

// UsageLimitHandler 用量限制处理器
type UsageLimitHandler struct {
	tokenUsageService TokenUsageService
	logger            *logrus.Logger
}

// NewUsageLimitHandler 创建用量限制处理器
func NewUsageLimitHandler(tokenUsageService TokenUsageService, logger *logrus.Logger) *UsageLimitHandler {
	return &UsageLimitHandler{
		tokenUsageService: tokenUsageService,
		logger:            logger,
	}
}

// GetUsageStats 获取用户用量统计
func (h *UsageLimitHandler) GetUsageStats(c *gin.Context) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "User not authenticated",
		})
		return
	}

	userID, ok := userIDInterface.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Invalid user ID",
		})
		return
	}

	// 获取周期参数
	periodStr := c.DefaultQuery("period", "monthly")
	var period paymodels.TokenUsagePeriod
	switch periodStr {
	case "daily":
		period = paymodels.TokenUsagePeriodDaily
	case "weekly":
		period = paymodels.TokenUsagePeriodWeekly
	case "monthly":
		period = paymodels.TokenUsagePeriodMonthly
	case "yearly":
		period = paymodels.TokenUsagePeriodYearly
	default:
		period = paymodels.TokenUsagePeriodMonthly
	}

	// 获取用量统计
	stats, err := h.tokenUsageService.GetUserUsageStats(c.Request.Context(), userID, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get usage stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Failed to get usage stats",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"period": periodStr,
			"stats":  stats,
		},
	})
}

// GetUsageByType 获取用户各类型用量
func (h *UsageLimitHandler) GetUsageByType(c *gin.Context) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "User not authenticated",
		})
		return
	}

	userID, ok := userIDInterface.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Invalid user ID",
		})
		return
	}

	// 获取周期参数
	periodStr := c.DefaultQuery("period", "monthly")
	var period paymodels.TokenUsagePeriod
	switch periodStr {
	case "daily":
		period = paymodels.TokenUsagePeriodDaily
	case "weekly":
		period = paymodels.TokenUsagePeriodWeekly
	case "monthly":
		period = paymodels.TokenUsagePeriodMonthly
	case "yearly":
		period = paymodels.TokenUsagePeriodYearly
	default:
		period = paymodels.TokenUsagePeriodMonthly
	}

	// 获取各类型用量
	usageByType, err := h.tokenUsageService.GetUserUsageByType(c.Request.Context(), userID, period)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get usage by type")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Failed to get usage by type",
		})
		return
	}

	// 转换类型为字符串键
	usageMap := make(map[string]interface{})
	for usageType, stats := range usageByType {
		usageMap[getUsageTypeString(usageType)] = stats
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": gin.H{
			"period": periodStr,
			"usage":  usageMap,
		},
	})
}

// CheckUsageLimit 检查特定类型的用量限制
func (h *UsageLimitHandler) CheckUsageLimit(c *gin.Context) {
	userIDInterface, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code": 401,
			"msg":  "User not authenticated",
		})
		return
	}

	userID, ok := userIDInterface.(int64)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Invalid user ID",
		})
		return
	}

	// 获取用量类型参数
	usageTypeStr := c.Param("type")
	usageType := getUsageTypeFromString(usageTypeStr)
	if usageType == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": 400,
			"msg":  "Invalid usage type",
		})
		return
	}

	// 检查用量限制
	limitResult, err := h.tokenUsageService.CheckUserUsageLimit(c.Request.Context(), userID, usageType)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check usage limit")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": 500,
			"msg":  "Failed to check usage limit",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
		"data": limitResult,
	})
}

// getUsageTypeString 获取用量类型字符串
func getUsageTypeString(usageType paymodels.TokenUsageType) string {
	switch usageType {
	case paymodels.TokenUsageTypeChat:
		return "chat"
	case paymodels.TokenUsageTypeImageGen:
		return "image_gen"
	case paymodels.TokenUsageTypeVideoGen:
		return "video_gen"
	case paymodels.TokenUsageTypeStoryGen:
		return "story_gen"
	case paymodels.TokenUsageTypeRoleGen:
		return "role_gen"
	case paymodels.TokenUsageTypeContextGen:
		return "context_gen"
	case paymodels.TokenUsageTypeOther:
		return "other"
	default:
		return "unknown"
	}
}

// getUsageTypeFromString 从字符串获取用量类型
func getUsageTypeFromString(usageTypeStr string) paymodels.TokenUsageType {
	switch usageTypeStr {
	case "chat":
		return paymodels.TokenUsageTypeChat
	case "image_gen":
		return paymodels.TokenUsageTypeImageGen
	case "video_gen":
		return paymodels.TokenUsageTypeVideoGen
	case "story_gen":
		return paymodels.TokenUsageTypeStoryGen
	case "role_gen":
		return paymodels.TokenUsageTypeRoleGen
	case "context_gen":
		return paymodels.TokenUsageTypeContextGen
	case "other":
		return paymodels.TokenUsageTypeOther
	default:
		return 0
	}
}

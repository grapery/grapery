package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// UserQuotaHandler 用户配额 Handler
type UserQuotaHandler struct {
	quotaService *service.UserQuotaService
	logger       *zap.Logger
}

// NewUserQuotaHandler 创建用户配额 Handler
func NewUserQuotaHandler(quotaService *service.UserQuotaService, logger *zap.Logger) *UserQuotaHandler {
	return &UserQuotaHandler{
		quotaService: quotaService,
		logger:       logger,
	}
}

// requireAuth 鉴权中间件
func (h *UserQuotaHandler) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("userID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// requireOwnership 检查用户所有权中间件
// 用于 /users/:userId/* 端点，确保当前用户只能访问自己的信息
func (h *UserQuotaHandler) requireOwnership() gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUserID := c.GetString("userID")
		if currentUserID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		targetUserID := c.Param("userId")
		if targetUserID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
			c.Abort()
			return
		}

		// 检查是否是用户自己的信息
		if currentUserID != targetUserID {
			c.JSON(http.StatusForbidden, gin.H{"error": "access denied: you can only access your own information"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetUserQuotaInfo 获取用户配额信息
// GET /users/:userId/quota
// 需要鉴权 + 所有权检查
func (h *UserQuotaHandler) GetUserQuotaInfo(c *gin.Context) {
	userID := c.Param("userId") // 从路径参数获取，中间件已验证所有权

	quotaInfo, err := h.quotaService.GetUserQuotaInfo(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user quota info",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quota info"})
		return
	}

	c.JSON(http.StatusOK, quotaInfo)
}

// GetUserDashboardInfo 获取用户主页综合信息
// GET /users/:userId/dashboard
// 需要鉴权 + 所有权检查
func (h *UserQuotaHandler) GetUserDashboardInfo(c *gin.Context) {
	userID := c.Param("userId") // 从路径参数获取，中间件已验证所有权

	dashboardInfo, err := h.quotaService.GetUserDashboardInfo(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user dashboard info",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get dashboard info"})
		return
	}

	c.JSON(http.StatusOK, dashboardInfo)
}

// GetUserMembershipInfo 获取用户会员信息
// GET /users/:userId/membership
// 需要鉴权 + 所有权检查
func (h *UserQuotaHandler) GetUserMembershipInfo(c *gin.Context) {
	userID := c.Param("userId") // 从路径参数获取，中间件已验证所有权

	membershipInfo, err := h.quotaService.GetUserMembershipInfo(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user membership info",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get membership info"})
		return
	}

	c.JSON(http.StatusOK, membershipInfo)
}

// GetUserRechargeInfo 获取用户充值信息
// GET /users/:userId/recharge
// 需要鉴权 + 所有权检查
func (h *UserQuotaHandler) GetUserRechargeInfo(c *gin.Context) {
	userID := c.Param("userId") // 从路径参数获取，中间件已验证所有权

	rechargeInfo, err := h.quotaService.GetUserRechargeInfo(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user recharge info",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recharge info"})
		return
	}

	c.JSON(http.StatusOK, rechargeInfo)
}

// GetUserUsageStatistics 获取用户使用统计
// GET /users/:userId/usage?period=today|week|month|year|all
// 需要鉴权 + 所有权检查
func (h *UserQuotaHandler) GetUserUsageStatistics(c *gin.Context) {
	userID := c.Param("userId") // 从路径参数获取，中间件已验证所有权
	period := c.DefaultQuery("period", "month")

	usageStats, err := h.quotaService.GetUserUsageStatistics(c.Request.Context(), userID, period)
	if err != nil {
		h.logger.Error("failed to get user usage statistics",
			zap.String("userID", userID),
			zap.String("period", period),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get usage statistics"})
		return
	}

	c.JSON(http.StatusOK, usageStats)
}

// GetTokenUsageHistory 获取 Token 使用历史
// GET /users/:userId/token-history?page=1&pageSize=20
// 需要鉴权 + 所有权检查
func (h *UserQuotaHandler) GetTokenUsageHistory(c *gin.Context) {
	userID := c.Param("userId") // 从路径参数获取，中间件已验证所有权
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	history, err := h.quotaService.GetTokenUsageHistory(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		h.logger.Error("failed to get token usage history",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get token history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

// GetMembershipTiers 获取所有会员等级信息
// GET /memberships/tiers
func (h *UserQuotaHandler) GetMembershipTiers(c *gin.Context) {
	tiers, err := h.quotaService.GetMembershipTiers(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get membership tiers",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get membership tiers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tiers": tiers,
	})
}

// SetupUserQuotaRoutes 设置用户配额相关路由
// 注意：这些路由需要鉴权中间件
func SetupUserQuotaRoutes(router *gin.RouterGroup, handler *UserQuotaHandler) {
	// /users/:userId/* 端点需要鉴权 + 所有权检查
	users := router.Group("/users")
	users.Use(handler.requireAuth())
	users.Use(handler.requireOwnership())
	{
		// 用户配额信息
		users.GET(":userId/quota", handler.GetUserQuotaInfo)

		// 用户主页综合信息
		users.GET(":userId/dashboard", handler.GetUserDashboardInfo)

		// 用户会员信息
		users.GET(":userId/membership", handler.GetUserMembershipInfo)

		// 用户充值信息
		users.GET(":userId/recharge", handler.GetUserRechargeInfo)

		// 用户使用统计
		users.GET(":userId/usage", handler.GetUserUsageStatistics)

		// Token 使用历史
		users.GET(":userId/token-history", handler.GetTokenUsageHistory)
	}

	// 会员等级信息（公开，无需鉴权）
	router.GET("/memberships/tiers", handler.GetMembershipTiers)
}

// SetupCurrentUserQuotaRoutes 设置当前用户配额路由
// 推荐使用这些路由，更简洁且自动获取当前用户 ID
func SetupCurrentUserQuotaRoutes(router *gin.RouterGroup, handler *UserQuotaHandler) {
	// /me/* 端点需要鉴权
	currentUser := router.Group("/me")
	currentUser.Use(handler.requireAuth())
	{
		// 当前用户配额信息
		currentUser.GET("/quota", handler.GetCurrentUserQuotaInfo)

		// 当前用户主页信息
		currentUser.GET("/dashboard", handler.GetCurrentUserDashboardInfo)

		// 当前用户会员信息
		currentUser.GET("/membership", handler.GetCurrentUserMembershipInfo)

		// 当前用户充值信息
		currentUser.GET("/recharge", handler.GetCurrentUserRechargeInfo)

		// 当前用户使用统计
		currentUser.GET("/usage", handler.GetCurrentUserUsageStatistics)

		// 当前用户 Token 历史记录
		currentUser.GET("/token-history", handler.GetCurrentUserTokenHistory)
	}
}

// ============== 当前用户的相关方法 ==============
// 这些方法使用 /me/* 路径，自动从鉴权中间件获取当前用户 ID

// GetCurrentUserQuotaInfo 获取当前用户配额信息
// GET /me/quota
func (h *UserQuotaHandler) GetCurrentUserQuotaInfo(c *gin.Context) {
	userID := c.GetString("userID") // 从鉴权中间件获取

	quotaInfo, err := h.quotaService.GetUserQuotaInfo(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get current user quota info",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get quota info"})
		return
	}

	c.JSON(http.StatusOK, quotaInfo)
}

// GetCurrentUserDashboardInfo 获取当前用户主页信息
// GET /me/dashboard
func (h *UserQuotaHandler) GetCurrentUserDashboardInfo(c *gin.Context) {
	userID := c.GetString("userID") // 从鉴权中间件获取

	dashboardInfo, err := h.quotaService.GetUserDashboardInfo(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get current user dashboard info",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get dashboard info"})
		return
	}

	c.JSON(http.StatusOK, dashboardInfo)
}

// GetCurrentUserMembershipInfo 获取当前用户会员信息
// GET /me/membership
func (h *UserQuotaHandler) GetCurrentUserMembershipInfo(c *gin.Context) {
	userID := c.GetString("userID") // 从鉴权中间件获取

	membershipInfo, err := h.quotaService.GetUserMembershipInfo(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get current user membership info",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get membership info"})
		return
	}

	c.JSON(http.StatusOK, membershipInfo)
}

// GetCurrentUserRechargeInfo 获取当前用户充值信息
// GET /me/recharge
func (h *UserQuotaHandler) GetCurrentUserRechargeInfo(c *gin.Context) {
	userID := c.GetString("userID") // 从鉴权中间件获取

	rechargeInfo, err := h.quotaService.GetUserRechargeInfo(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get current user recharge info",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get recharge info"})
		return
	}

	c.JSON(http.StatusOK, rechargeInfo)
}

// GetCurrentUserUsageStatistics 获取当前用户使用统计
// GET /me/usage?period=today|week|month|year|all
func (h *UserQuotaHandler) GetCurrentUserUsageStatistics(c *gin.Context) {
	userID := c.GetString("userID") // 从鉴权中间件获取
	period := c.DefaultQuery("period", "month")

	usageStats, err := h.quotaService.GetUserUsageStatistics(c.Request.Context(), userID, period)
	if err != nil {
		h.logger.Error("failed to get current user usage statistics",
			zap.String("userID", userID),
			zap.String("period", period),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get usage statistics"})
		return
	}

	c.JSON(http.StatusOK, usageStats)
}

// GetCurrentUserTokenHistory 获取当前用户 Token 历史记录
// GET /me/token-history?page=1&pageSize=20
func (h *UserQuotaHandler) GetCurrentUserTokenHistory(c *gin.Context) {
	userID := c.GetString("userID") // 从鉴权中间件获取
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	history, err := h.quotaService.GetTokenUsageHistory(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		h.logger.Error("failed to get current user token history",
			zap.String("userID", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get token history"})
		return
	}

	c.JSON(http.StatusOK, history)
}

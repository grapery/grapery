package http

import (
	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// Handler handles HTTP requests
type Handler struct {
	svc       *service.Service
	aiService *service.AIService
	logger    *zap.Logger
}

// NewHandler creates a new HTTP handler
func NewHandler(svc *service.Service, aiService *service.AIService, logger *zap.Logger) *Handler {
	return &Handler{
		svc:       svc,
		aiService: aiService,
		logger:    logger,
	}
}

// SetupRouter configures and returns the Gin router
func SetupRouter(h *Handler, logger *zap.Logger) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	// 健康检查
	router.GET("/health", h.Health)

	// 静态文件服务 - 上传的文件
	router.GET("/uploads/*filepath", ServeUploadedFile)

	// API 路由组
	api := router.Group("/api")
	{
		// 认证路由（无需认证）
		auth := api.Group("/auth")
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/password/request-reset", h.RequestPasswordReset)
			auth.POST("/password/reset", h.ResetPassword)
			auth.POST("/refresh", h.RefreshToken)
		}

		// 需要认证的路由
		authenticated := api.Group("")
		authenticated.Use(authPkg.AuthMiddleware())
		{
			// 公开路由迁移（现在需要认证）
			authenticated.GET("/search", h.Search)
			authenticated.GET("/users/:id", h.GetUserProfile)
			authenticated.GET("/users/:id/followers", h.GetFollowers)
			authenticated.GET("/users/:id/following", h.GetFollowing)
			authenticated.GET("/users/:id/stories", h.GetUserStories)
			authenticated.GET("/users/:id/characters", h.GetUserCharacters)
			authenticated.GET("/users/:id/liked-stories", h.GetLikedStories)
			authenticated.GET("/users/:id/liked-characters", h.GetLikedCharacters)
			authenticated.GET("/users/:id/liked-storyboards", h.GetLikedStoryboards)
			authenticated.GET("/users/:id/draft-storyboards", h.GetDraftStoryboards)
			authenticated.GET("/users/:id/activities", h.GetUserActivityList)
			authenticated.GET("/users/:id/activities/heatmap", h.GetUserActivityHeatmapByID)
			authenticated.GET("/users/:id/stats", h.GetUserStats)
			authenticated.GET("/tags/popular", h.GetPopularTags)
			authenticated.GET("/tags/:id/stories", h.GetStoriesByTag)
			authenticated.GET("/stories", h.ListStories)
			authenticated.GET("/stories/:id", h.GetStory)
			authenticated.GET("/stories/:id/tags", h.GetStoryTags)
			authenticated.GET("/stories/:id/stats", h.GetStoryStats)
			authenticated.GET("/storyboards", h.ListStoryboards)
			authenticated.GET("/storyboards/feed", h.GetStoryboardFeed) // Community storyboard feed
			authenticated.GET("/storyboards/:id", h.GetStoryboard)
			authenticated.GET("/storyboards/:id/children", h.GetStoryboardChildren)
			authenticated.GET("/storyboards/:id/tree", h.GetStoryboardTree)
			authenticated.GET("/comments", h.ListComments)
			authenticated.GET("/comments/:id", h.GetComment)
			authenticated.GET("/comments/:id/replies", h.GetCommentReplies)
			authenticated.GET("/comments/:id/tree", h.GetCommentTree)
			authenticated.GET("/characters", h.ListCharacters)
			authenticated.GET("/characters/:id", h.GetCharacter)
			authenticated.GET("/characters/:id/analytics", h.GetCharacterAnalytics)
			authenticated.GET("/characters/:id/posters", h.GetCharacterPosters)
			authenticated.GET("/characters/:id/storyboards", h.GetCharacterStoryboards)
			authenticated.GET("/groups", h.ListGroups)
			authenticated.GET("/groups/:id", h.GetGroup)
			authenticated.GET("/groups/:id/members", h.GetGroupMembers)
			authenticated.GET("/groups/:id/activities", h.GetGroupActivities)
			authenticated.GET("/groups/:id/activities/heatmap", h.GetGroupActivityHeatmap)
			authenticated.GET("/activities/global", h.GetGlobalActivities)

			// 用户相关
			authenticated.GET("/auth/me", h.CurrentUser)
			authenticated.POST("/auth/password/change", h.ChangePassword)
			authenticated.PUT("/users/:id", h.UpdateUserProfile)
			authenticated.PUT("/users/:id/avatar", h.UpdateUserAvatar)
			authenticated.PUT("/users/:id/background", h.UpdateUserBackground)

			// 故事相关
			authenticated.POST("/stories", h.CreateStory)
			authenticated.PUT("/stories/:id", h.UpdateStory)
			authenticated.DELETE("/stories/:id", h.DeleteStory)
			authenticated.POST("/stories/:id/like", h.LikeStory)
			authenticated.DELETE("/stories/:id/like", h.UnlikeStory)
			authenticated.POST("/stories/:id/follow", h.FollowStory)
			authenticated.DELETE("/stories/:id/follow", h.UnfollowStory)

			// 故事渲染和发布
			authenticated.POST("/stories/:id/render", h.RenderStory)            // AI渲染（同步）- 丰富描述+生成图片
			authenticated.POST("/stories/:id/render-media", h.RenderStoryMedia) // 媒体渲染（异步）- 视频/图片集/动画
			authenticated.GET("/stories/:id/render-status", h.GetRenderTaskStatus)
			authenticated.POST("/stories/:id/publish", h.PublishStory)
			authenticated.POST("/stories/:id/unpublish", h.UnpublishStory)
			authenticated.GET("/stories/:id/contributors", h.GetStoryContributors)
			authenticated.POST("/stories/:id/contributors", h.InviteStoryContributor)
			authenticated.DELETE("/stories/:id/contributors/:userId", h.RemoveStoryContributor)

			// Storyboard 相关
			authenticated.POST("/storyboards", h.CreateStoryboard)
			authenticated.PUT("/storyboards/:id", h.UpdateStoryboard)
			authenticated.DELETE("/storyboards/:id", h.DeleteStoryboard)
			authenticated.POST("/storyboards/:id/fork", h.ForkStoryboard)
			authenticated.POST("/storyboards/:id/like", h.LikeStoryboard)
			authenticated.DELETE("/storyboards/:id/like", h.UnlikeStoryboard)

			// Storyboard AI Generation 相关
			authenticated.POST("/storyboards/:id/generate/content", h.GenerateContent)
			authenticated.POST("/storyboards/:id/generate/scene-details", h.GenerateSceneDetails)
			authenticated.POST("/storyboards/:id/generate/image", h.GenerateStoryboardImage)
			authenticated.POST("/storyboards/:id/generate/video", h.GenerateStoryboardVideo)
			authenticated.GET("/storyboards/:id/generation-progress", h.GetGenerationProgress)
			authenticated.POST("/storyboards/:id/publish", h.PublishStoryboard)

			// 评论相关
			authenticated.POST("/comments", h.CreateComment)
			authenticated.PUT("/comments/:id", h.UpdateComment)
			authenticated.DELETE("/comments/:id", h.DeleteComment)
			authenticated.PUT("/comments/:id/like", h.ToggleLikeComment) // Toggle like/unlike
			authenticated.POST("/comments/:id/like", h.LikeComment)
			authenticated.POST("/comments/:id/dislike", h.DislikeComment)
			authenticated.DELETE("/comments/:id/like", h.UnlikeComment)

			// 通知相关
			authenticated.GET("/notifications", h.ListNotifications)
			authenticated.GET("/notifications/unread/count", h.UnreadCount)
			authenticated.POST("/notifications/:id/read", h.MarkAsRead)
			authenticated.POST("/notifications/read-all", h.MarkAllAsRead)
			authenticated.DELETE("/notifications/:id", h.DeleteNotification)

			// 用户关注相关
			authenticated.POST("/users/:id/follow", h.FollowUser)
			authenticated.DELETE("/users/:id/follow", h.UnfollowUser)

			// 标签相关
			authenticated.POST("/stories/:id/tags", h.AddStoryTags)

			// 角色相关
			authenticated.POST("/characters", h.CreateCharacter)
			authenticated.POST("/characters/generate", h.GenerateCharacterAttributes) // AI生成角色属性
			authenticated.PUT("/characters/:id", h.UpdateCharacter)
			authenticated.DELETE("/characters/:id", h.DeleteCharacter)
			authenticated.POST("/characters/:id/follow", h.FollowCharacter)
			authenticated.DELETE("/characters/:id/follow", h.UnfollowCharacter)
			authenticated.POST("/characters/:id/skills", h.AddCharacterSkill)
			authenticated.DELETE("/characters/:id/skills/:skill", h.RemoveCharacterSkill)
			authenticated.POST("/characters/:id/posters", h.CreateCharacterPoster)
			authenticated.POST("/posters/:id/like", h.LikeCharacterPoster)
			authenticated.POST("/posters/:id/share", h.ShareCharacterPoster)
			authenticated.DELETE("/posters/:id", h.DeleteCharacterPoster)

			// 群组相关
			authenticated.POST("/groups", h.CreateGroup)
			authenticated.PUT("/groups/:id", h.UpdateGroup)
			authenticated.DELETE("/groups/:id", h.DeleteGroup)
			authenticated.POST("/groups/:id/avatar", h.UpdateGroupAvatar)
			authenticated.POST("/groups/:id/invite", h.InviteMember)
			authenticated.DELETE("/groups/:id/members/:userId", h.RemoveMember)
			authenticated.POST("/groups/:id/members/:userId/role", h.UpdateMemberRole)
			authenticated.POST("/groups/:id/leave", h.LeaveGroup)

			// 邀请相关
			authenticated.GET("/invitations/pending", h.GetPendingInvitations)
			authenticated.POST("/invitations/:id/accept", h.AcceptInvitation)
			authenticated.POST("/invitations/:id/reject", h.RejectInvitation)

			// 聊天相关
			authenticated.GET("/chats", h.ListChatThreads)
			authenticated.GET("/chats/unread/count", h.GetUnreadChatCount)
			authenticated.GET("/chats/:id", h.GetChatThread)
			authenticated.POST("/chats", h.CreateChatThread)
			authenticated.DELETE("/chats/:id", h.DeleteChatThread)
			authenticated.POST("/chats/:id/read", h.MarkChatThreadAsRead)
			authenticated.GET("/chats/:id/messages", h.ListChatMessages)
			authenticated.POST("/chats/:id/messages", h.SendChatMessage)
			authenticated.DELETE("/chats/:id/messages/:messageId", h.DeleteChatMessage)

			// 文件上传相关
			authenticated.POST("/upload/image", h.UploadImage)
			authenticated.POST("/upload/avatar", h.UploadAvatar)
			authenticated.POST("/upload/cover", h.UploadCover)
			authenticated.POST("/upload/video", h.UploadVideo)
			authenticated.POST("/upload/from-url", h.UploadImageFromURL)
			authenticated.POST("/upload/multiple", h.UploadMultiple)
			authenticated.DELETE("/upload", h.DeleteUpload)
			authenticated.GET("/upload/sts-token", h.GetSTSToken)
			authenticated.GET("/upload/image-levels", h.GetImageLevels)

			// 统计相关
			authenticated.GET("/dashboard/stats", h.GetDashboardStats)

			// 活动流相关
			authenticated.GET("/activities", h.GetUserActivities)

			// 资产管理相关
			authenticated.GET("/assets", h.ListAssets)
			authenticated.GET("/assets/:id", h.GetAsset)
			authenticated.POST("/assets", h.CreateAsset)
			authenticated.PUT("/assets/:id", h.UpdateAsset)
			authenticated.DELETE("/assets/:id", h.DeleteAsset)

			// SSE 实时推送（HTTP Server-Sent Events）
			authenticated.GET("/sse/chat/:threadId", h.SSEChatStream)
			authenticated.GET("/sse/notifications", h.SSENotificationStream)
			authenticated.GET("/sse/activities", h.SSEActivityStream)

			// Long Polling 降级方案
			authenticated.GET("/poll/chat/:threadId/messages", h.LongPollChatMessages)

			// 设备管理（APNs推送）
			authenticated.POST("/devices/register", h.RegisterDevice)
			authenticated.POST("/devices/unregister", h.UnregisterDevice)
			authenticated.POST("/devices/badge", h.UpdateBadge)
			authenticated.POST("/devices/test-push", h.TestPushNotification)

			// AI 生成功能
			authenticated.POST("/ai/generate-story", h.GenerateStory)
			authenticated.POST("/ai/enhance-prompt", h.EnhancePrompt)
			authenticated.POST("/ai/generate-image", h.GenerateImage)
			authenticated.POST("/ai/generate-video", h.GenerateVideo)
			authenticated.GET("/ai/tasks/:id", h.GetTaskStatus)
			authenticated.GET("/ai/tasks/:id/result", h.GetTaskResult)
			authenticated.DELETE("/ai/tasks/:id", h.CancelTask)

			// 风格配置相关
			authenticated.GET("/styles", h.GetStyleConfigs)
			authenticated.GET("/styles/options", h.GetStyleOptions)
			authenticated.GET("/styles/search", h.SearchStyleConfigs)
			authenticated.GET("/styles/:id", h.GetStyleConfigByID)
			authenticated.GET("/styles/by-name/:style", h.GetStyleConfigByStyle)
			authenticated.POST("/styles", h.CreateStyleConfig)
			authenticated.PUT("/styles/:id", h.UpdateStyleConfig)
			authenticated.DELETE("/styles/:id", h.DeleteStyleConfig)
			authenticated.POST("/styles/initialize", h.InitializeDefaultStyles)
		}

	}

	logger.Info("router configured successfully")
	return router
}

// Health returns service health status
func (h *Handler) Health(c *gin.Context) {
	Success(c, h.svc.Health())
}

// ========== 认证相关端点 ==========

// Register 用户注册
func (h *Handler) Register(c *gin.Context) {
	var req service.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, resp)
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, resp)
}

// ChangePassword 修改密码
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req service.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	if err := h.svc.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "password changed successfully"})
}

// RequestPasswordReset 请求密码重置
func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req service.PasswordResetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	if err := h.svc.RequestPasswordReset(c.Request.Context(), &req); err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "password reset email sent"})
}

// ResetPassword 重置密码
func (h *Handler) ResetPassword(c *gin.Context) {
	var req service.PasswordResetConfirm
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	if err := h.svc.ResetPassword(c.Request.Context(), &req); err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, gin.H{"message": "password reset successfully"})
}

// RefreshToken 刷新访问令牌
func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	resp, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, resp)
}

// CurrentUser 获取当前用户信息
func (h *Handler) CurrentUser(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	user, err := h.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		Error(c, CodeError, err.Error())
		return
	}

	Success(c, user)
}

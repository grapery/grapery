package http

import (
	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/middleware"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// Handler handles HTTP requests
type Handler struct {
	svc               *service.Service
	aiService         *service.AIService
	storyboardPathSvc *service.StoryboardPathService
	shareSigner       *service.ShareLinkSigner
	logger            *zap.Logger
}

// HandlerDependencies 包含所有 handler 依赖的服务
type HandlerDependencies struct {
	Service               *service.Service
	AIService             *service.AIService
	StoryboardPathService *service.StoryboardPathService
	InteractionService    service.InteractionService
	UserSettingsService   service.UserSettingsService
	GenreCatalogService   *service.GenreCatalogService
	FeedbackService       service.FeedbackService
	Logger                *zap.Logger
	Cache                 cache.Cache
	ShareSigner           *service.ShareLinkSigner
}

// NewHandler creates a new HTTP handler (legacy constructor)
func NewHandler(svc *service.Service, aiService *service.AIService, logger *zap.Logger) *Handler {
	return &Handler{
		svc:       svc,
		aiService: aiService,
		logger:    logger,
	}
}

// NewHandlerWithDeps creates a new HTTP handler with all dependencies
func NewHandlerWithDeps(deps *HandlerDependencies) *Handler {
	return &Handler{
		svc:               deps.Service,
		aiService:         deps.AIService,
		storyboardPathSvc: deps.StoryboardPathService,
		shareSigner:       deps.ShareSigner,
		logger:            deps.Logger,
	}
}

// SetupRouter configures and returns the Gin router
func SetupRouter(deps *HandlerDependencies) *gin.Engine {
	h := NewHandlerWithDeps(deps)
	router := gin.New()
	router.Use(gin.Recovery())

	// Rate limiters (nil-safe: if Cache is nil, middleware is not applied)
	var authLimiter, aiLimiter, apiLimiter, sharePreviewLimiter gin.HandlerFunc
	if deps.Cache != nil {
		authLimiter = middleware.NewRateLimiter(deps.Cache, middleware.RateLimitAuth, deps.Logger)
		aiLimiter = middleware.NewRateLimiter(deps.Cache, middleware.RateLimitAIGeneration, deps.Logger)
		apiLimiter = middleware.NewRateLimiter(deps.Cache, middleware.RateLimitGeneral, deps.Logger)
		sharePreviewLimiter = middleware.NewRateLimiter(deps.Cache, middleware.RateLimitSharePreview, deps.Logger)
	}

	// 健康检查
	router.GET("/health", h.Health)

	// 静态文件服务 - 上传的文件
	router.GET("/uploads/*filepath", ServeUploadedFile)

	// API 路由组
	api := router.Group("/api")
	{
		// 认证路由（无需认证）
		auth := api.Group("/auth")
		if authLimiter != nil {
			auth.Use(authLimiter)
		}
		{
			auth.POST("/register", h.Register)
			auth.POST("/login", h.Login)
			auth.POST("/password/request-reset", h.RequestPasswordReset)
			auth.POST("/password/reset", h.ResetPassword)
			auth.POST("/email/send-verification-code", h.SendEmailVerificationCode)
			auth.POST("/email/verify", h.VerifyEmail)
			auth.POST("/refresh", h.RefreshToken)
			// 手机号验证码登录（未认证；不存在则自动注册）
			auth.POST("/phone/login/send-sms-code", h.PhoneLoginSendSMSCode)
			auth.POST("/phone/login/verify", h.PhoneLoginVerify)
		}

		// 主 API 公开法律文档（无需登录；Grapery 信封 code=1）
		legalPublic := api.Group("/v1/legal")
		{
			legalPublic.GET("/terms-of-service", h.GetLegalTermsOfService)
			legalPublic.GET("/privacy-policy", h.GetLegalPrivacyPolicy)
		}

		// 故事板 feed：游客可读 discover（及 community 等）；带 JWT 时可附加点赞态等。
		// tab=following 且无登录时 handler 侧 userID 为空，service 返回空列表。
		storyboardFeed := api.Group("/v1")
		if apiLimiter != nil {
			storyboardFeed.Use(apiLimiter)
		}
		storyboardFeed.Use(authPkg.OptionalAuthMiddleware())
		storyboardFeed.GET("/storyboards/feed", h.GetStoryboardFeed)

		// 需要认证的路由（使用 /api/v1 前缀）
		authenticated := api.Group("/v1")
		if apiLimiter != nil {
			authenticated.Use(apiLimiter)
		}
		authenticated.Use(authPkg.AuthMiddleware())
		authenticated.Use(h.EnsureActiveUser())
		authenticated.Use(h.RestrictPendingDeletionWrites())

		// AI generation rate-limited sub-group
		aiGen := authenticated.Group("")
		if aiLimiter != nil {
			aiGen.Use(aiLimiter)
		}
		{
			// 公开路由迁移（现在需要认证）
			authenticated.GET("/search", h.Search)
			authenticated.GET("/plaza", h.GetPlaza)
			authenticated.GET("/users/:id", h.GetUserProfile)
			authenticated.GET("/users/:id/followers", h.GetFollowers)
			authenticated.GET("/users/:id/following", h.GetFollowing)
			authenticated.GET("/users/:id/stories", h.GetUserStories)
			authenticated.GET("/users/:id/characters", h.GetUserCharacters)
			authenticated.GET("/users/:id/storyboards", h.GetUserStoryboards)
			authenticated.GET("/dashboard/storyboards", h.GetDashboardStoryboards)
			// REMOVED: /users/:id/drafts - not in StoryCreationAppUI design
			authenticated.GET("/users/:id/liked-stories", h.GetLikedStories)
			authenticated.GET("/users/:id/liked-characters", h.GetLikedCharacters)
			authenticated.GET("/users/:id/liked-storyboards", h.GetLikedStoryboards)
			// REMOVED: /users/:id/draft-storyboards - not in StoryCreationAppUI design
			// REMOVED: /users/:id/activities - not in StoryCreationAppUI design
			authenticated.GET("/users/:id/stats", h.GetUserStats)
			authenticated.GET("/tags/popular", h.GetPopularTags)
			authenticated.GET("/tags/:id/stories", h.GetStoriesByTag)
			authenticated.GET("/stories", h.ListStories)
			authenticated.GET("/stories/:id", h.GetStory)
			authenticated.GET("/stories/:id/tags", h.GetStoryTags)
			authenticated.GET("/stories/:id/stats", h.GetStoryStats)
			authenticated.GET("/storyboards", h.ListStoryboards)
			// REMOVED: Dashboard storyboard feeds - not in StoryCreationAppUI design
			authenticated.GET("/storyboards/:id", h.GetStoryboard)
			authenticated.GET("/storyboards/:id/children", h.GetStoryboardChildren)
			authenticated.GET("/storyboards/:id/tree", h.GetStoryboardTree)

			// 评论相关
			authenticated.GET("/comments", h.ListComments)
			authenticated.GET("/comments/:id", h.GetComment)
			authenticated.GET("/comments/:id/replies", h.GetCommentReplies)
			authenticated.GET("/comments/:id/tree", h.GetCommentTree)

			// 角色相关
			authenticated.GET("/characters", h.ListCharacters)
			authenticated.GET("/characters/:id", h.GetCharacter)
			authenticated.GET("/characters/:id/analytics", h.GetCharacterAnalytics)
			// REMOVED: /characters/:id/posters - not in StoryCreationAppUI design
			authenticated.GET("/characters/:id/storyboards", h.GetCharacterStoryboards)

			// 用户相关
			authenticated.GET("/auth/me", h.CurrentUser)
			authenticated.GET("/auth/account/deletion", h.GetAccountDeletionStatus)
			authenticated.POST("/auth/account/deletion/cancel", h.CancelAccountDeletion)
			authenticated.POST("/auth/account/deletion/send-sms-code", h.SendAccountDeletionSMS)
			authenticated.POST("/auth/account/deletion/verify-sms-code", h.VerifyAccountDeletionSMS)
			authenticated.DELETE("/auth/account", h.RequestAccountDeletion)
			authenticated.POST("/auth/account/phone/send-sms-code", h.SendAccountContactPhoneSMS)
			authenticated.POST("/auth/account/phone/verify-sms-code", h.VerifyAccountContactPhoneSMS)
			authenticated.POST("/auth/account/email/send-verification-code", h.SendAccountContactEmailCode)
			authenticated.POST("/auth/account/email/verify", h.ConfirmAccountContactEmail)
			authenticated.POST("/auth/phone/send-sms-code", h.SendPhoneSMSVerificationCode)
			authenticated.POST("/auth/phone/verify-sms-code", h.VerifyPhoneSMSCode)
			authenticated.GET("/me/creator-analytics", h.GetMyCreatorAnalytics)
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
			aiGen.POST("/stories/:id/render", h.RenderStory)            // AI渲染（同步）- 丰富描述+生成图片
			aiGen.POST("/stories/:id/render-media", h.RenderStoryMedia) // 媒体渲染（异步）- 视频/图片集/动画
			authenticated.GET("/stories/:id/render-status", h.GetRenderTaskStatus)
			authenticated.POST("/stories/:id/publish", h.PublishStory)
			authenticated.POST("/stories/:id/unpublish", h.UnpublishStory)
			authenticated.GET("/stories/:id/contributors", h.GetStoryContributors)
			authenticated.POST("/stories/:id/contributors", h.InviteStoryContributor)
			authenticated.DELETE("/stories/:id/contributors/:userId", h.RemoveStoryContributor)

			// Story Scenes 相关
			authenticated.GET("/stories/:id/scenes", h.ListStoryScenes)
			authenticated.POST("/stories/:id/scenes", h.CreateStoryScene)
			authenticated.POST("/stories/:id/scenes/register-image", h.UploadSceneImage)
			aiGen.POST("/stories/:id/scenes/ai-generate-image", h.GenerateSceneImage)
			authenticated.PUT("/stories/:id/scenes/:sceneId", h.UpdateStoryScene)
			authenticated.DELETE("/stories/:id/scenes/:sceneId", h.DeleteStoryScene)

			// Story Panels 相关
			authenticated.GET("/stories/:id/panels", h.ListStoryPanels)
			authenticated.POST("/stories/:id/panels", h.CreateStoryPanel)
			authenticated.PUT("/stories/:id/panels/:panelId", h.UpdateStoryPanel)
			authenticated.DELETE("/stories/:id/panels/:panelId", h.DeleteStoryPanel)
			authenticated.POST("/stories/:id/panels/reorder", h.ReorderStoryPanels)

			// Story Comments 相关 (Enhanced)
			authenticated.GET("/stories/:id/comments", h.ListStoryComments)
			authenticated.POST("/stories/:id/comments", h.CreateStoryComment)
			authenticated.POST("/comments/:id/replies", h.CreateCommentReply)

			// 故事默认路径相关
			authenticated.POST("/stories/:id/default-path", h.SetDefaultPath)
			authenticated.POST("/stories/:id/default-path/auto", h.CalculateAutoPath)
			authenticated.GET("/stories/:id/default-path", h.GetDefaultPath)

			// Storyboard 相关
			authenticated.POST("/storyboards", h.CreateStoryboard)
			authenticated.PUT("/storyboards/:id", h.UpdateStoryboard)
			authenticated.PUT("/storyboards/:id/scenes/:sceneId", h.UpdateStoryboardPlotScene)
			authenticated.DELETE("/storyboards/:id", h.DeleteStoryboard)
			authenticated.POST("/storyboards/:id/fork", h.ForkStoryboard)
			authenticated.POST("/storyboards/:id/continue", h.ContinueStoryboard) // 平行宇宙续写
			authenticated.POST("/storyboards/:id/like", h.LikeStoryboard)
			authenticated.DELETE("/storyboards/:id/like", h.UnlikeStoryboard)

			// Storyboard Panels 相关
			authenticated.GET("/storyboards/:id/panels", h.ListStoryboardPanels)
			authenticated.POST("/storyboards/:id/panels", h.CreateStoryboardPanel)

			// Storyboard AI Generation 相关 (rate-limited)
			aiGen.POST("/storyboards/:id/generate/content", h.GenerateContent)
			aiGen.POST("/storyboards/:id/generate/structure", h.GenerateStoryboardStructure)
			aiGen.POST("/storyboards/:id/generate/scene-details", h.GenerateSceneDetails)
			aiGen.POST("/storyboards/:id/generate/image", h.GenerateStoryboardImage)
			aiGen.POST("/storyboards/:id/generate/images", h.GenerateAllStoryboardImages)
			aiGen.POST("/storyboards/:id/generate/comic-page", h.GenerateStoryboardComicPage)
			aiGen.POST("/storyboards/:id/generate/comic-pages", h.GenerateAllStoryboardComicPages)
			aiGen.POST("/storyboards/:id/generate/video", h.GenerateStoryboardVideo)
			authenticated.GET("/storyboards/:id/generation-progress", h.GetGenerationProgress)
			authenticated.POST("/storyboards/:id/cancel-generation", h.CancelStoryboardGeneration)
			authenticated.POST("/storyboards/:id/retry-failed-images", h.RetryFailedStoryboardImages)
			authenticated.POST("/storyboards/:id/publish", h.PublishStoryboard)
			authenticated.GET("/storyboards/:id/playlist.m3u8", h.GetStoryboardVideoPlaylist)
			authenticated.GET("/storyboards/:id/scenes/:sceneId/playlist.m3u8", h.GetSceneVideoPlaylist)

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

			// 用户积分相关 (StoryCreationAppUI Design)
			authenticated.GET("/users/:id/points", h.GetUserPoints)

			// 用户屏蔽/举报相关
			authenticated.GET("/users/blocked", h.GetBlockedUsers)
			authenticated.POST("/users/:id/block", h.BlockUser)
			authenticated.DELETE("/users/:id/block", h.UnblockUser)
			authenticated.POST("/users/:id/report", h.ReportUser)
			authenticated.POST("/content/report", h.ReportContent)

			// 标签相关
			authenticated.POST("/stories/:id/tags", h.AddStoryTags)

			// 邀请/推荐系统 (StoryCreationAppUI Design)
			authenticated.GET("/referrals/code", h.GetReferralCode)
			authenticated.GET("/referrals/share", h.GetInviteShareContent)
			authenticated.POST("/share/issue", h.IssueShareLink)
			authenticated.GET("/referrals/stats", h.GetReferralStats)
			authenticated.GET("/referrals", h.GetReferrals)
			authenticated.POST("/referrals/use", h.UseReferralCode)

			// 角色相关
			authenticated.POST("/characters", h.CreateCharacter)
			authenticated.POST("/character-generation-tasks", h.StartCharacterGenerationTask)
			authenticated.GET("/character-generation-tasks", h.ListCharacterGenerationTasks)
			authenticated.GET("/character-generation-tasks/:taskId", h.GetCharacterGenerationTask)
			authenticated.POST("/character-generation-tasks/:taskId/retry", h.RetryCharacterGenerationTask)
			authenticated.POST("/character-generation-tasks/:taskId/dismiss-from-drafts", h.DismissCharacterGenerationTaskFromDrafts)
			authenticated.GET("/stories/:id/fragment-character-suggestions", h.PreviewFragmentCharactersForStory)
			aiGen.POST("/characters/generate", h.GenerateCharacterAttributes) // AI生成角色属性
			authenticated.PUT("/characters/:id", h.UpdateCharacter)
			authenticated.DELETE("/characters/:id", h.DeleteCharacter)
			authenticated.POST("/characters/:id/follow", h.FollowCharacter)
			authenticated.DELETE("/characters/:id/follow", h.UnfollowCharacter)
			// REMOVED: skills routes - not in StoryCreationAppUI design
			aiGen.POST("/characters/:id/generate-avatar", h.GenerateCharacterAvatar)           // AI生成角色头像
			authenticated.PUT("/characters/:id/avatar", h.UpdateCharacterAvatar)               // 更新角色头像
			authenticated.PUT("/characters/:id/use-portrait-as-avatar", h.UsePortraitAsAvatar) // 使用portrait作为头像
			authenticated.GET("/characters/:id/portrait-prompt", h.GetPortraitPrompt)          // 获取形象生成推荐提示词
			aiGen.POST("/characters/:id/generate-portrait", h.GenerateCharacterPortrait)       // AI生成角色完整形象
			aiGen.POST("/characters/:id/generate-three-views", h.GenerateCharacterThreeViews)  // AI 生成/更新三视图
			authenticated.POST("/characters/:id/crop-avatar", h.CropAvatarFromPortrait)        // 从形象图裁剪头像
			// REMOVED: posters routes - not in StoryCreationAppUI design

			// 文件上传相关
			authenticated.POST("/upload/image", h.UploadImage)
			authenticated.POST("/upload/avatar", h.UploadAvatar)
			authenticated.POST("/upload/cover", h.UploadCover)
			authenticated.POST("/upload/video", h.UploadVideo)
			authenticated.POST("/upload/from-url", h.UploadImageFromURL)
			authenticated.POST("/upload/multiple", h.UploadMultiple)
			authenticated.POST("/upload/persist-image-levels", h.PersistImageLevels)
			authenticated.DELETE("/upload", h.DeleteUpload)
			authenticated.GET("/upload/sts-token", h.GetSTSToken)
			authenticated.GET("/upload/image-levels", h.GetImageLevels)

			// REMOVED: Dashboard stats - not in StoryCreationAppUI design

			// 资产管理相关
			authenticated.GET("/assets", h.ListAssets)
			authenticated.GET("/assets/:id", h.GetAsset)
			authenticated.POST("/assets", h.CreateAsset)
			authenticated.PUT("/assets/:id", h.UpdateAsset)
			authenticated.DELETE("/assets/:id", h.DeleteAsset)

			// SSE 实时推送（HTTP Server-Sent Events）
			authenticated.GET("/sse/notifications", h.SSENotificationStream)
			// REMOVED: /sse/activities - not in StoryCreationAppUI design

			// 设备管理（APNs推送）
			authenticated.POST("/devices/register", h.RegisterDevice)
			authenticated.POST("/devices/unregister", h.UnregisterDevice)
			authenticated.POST("/devices/badge", h.UpdateBadge)
			authenticated.POST("/devices/test-push", h.TestPushNotification)

			// AI 生成功能
			aiGen.POST("/ai/generate-story", h.GenerateStory)
			aiGen.POST("/ai/enhance-prompt", h.EnhancePrompt)
			aiGen.POST("/ai/generate-image", h.GenerateImage)
			aiGen.POST("/ai/generate-video", h.GenerateVideo)
			aiGen.POST("/ai/generate-character", h.GenerateCharacter) // iOS兼容端点
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

			// 邀请码管理
			authenticated.POST("/invitation-codes", h.CreateInvitationCode)
			authenticated.GET("/invitation-codes", h.ListInvitationCodes)
			authenticated.GET("/invitation-codes/:id", h.GetInvitationCode)
			authenticated.PUT("/invitation-codes/:id", h.UpdateInvitationCode)
			authenticated.DELETE("/invitation-codes/:id", h.DeleteInvitationCode)

			// Membership 相关
			authenticated.GET("/membership/plans", h.ListMembershipPlans)
			authenticated.GET("/membership/current", h.GetCurrentMembership)
			authenticated.POST("/membership/subscribe", h.SubscribeMembership)
			authenticated.POST("/membership/cancel", h.CancelMembership)
			authenticated.GET("/membership/usage", h.GetMembershipUsage)

			// 互动相关 (Interaction)
			interactionHandler := NewInteractionHandler(deps.InteractionService)
			interactionHandler.RegisterInteractionRoutes(authenticated)

			// 用户设置相关 (User Settings)
			userSettingsHandler := NewUserSettingsHandler(deps.UserSettingsService, deps.GenreCatalogService)
			userSettingsHandler.RegisterUserSettingsRoutes(authenticated)

			// 用户反馈
			if deps.FeedbackService != nil {
				feedbackHandler := NewFeedbackHandler(deps.FeedbackService)
				feedbackHandler.RegisterFeedbackRoutes(authenticated)
			}

			// 碎片相关 (Fragments)
			authenticated.GET("/fragments/:id/assets", h.GetFragmentGenerationAssets)
			aiGen.POST("/fragments/:id/convert-to-story", h.ConvertFragmentToStory)
			aiGen.POST("/fragments/:id/story-prefill-ai", h.ExpandFragmentStoryPrefillAI)
		}

		// 公开接口（无需认证）
		public := api.Group("")
		if apiLimiter != nil {
			public.Use(apiLimiter)
		}
		{
			public.POST("/invitation-codes/validate", h.ValidateInvitationCode) // 验证邀请码（注册前验证）
			// Public Trending (guest-accessible)
			public.GET("/public/stories/trending", h.GetTrendingStoriesPublic)
			public.GET("/public/trending/storyboards", h.GetPublicTrendingStoryboards)
			sharePublic := public.Group("/public/share")
			if sharePreviewLimiter != nil {
				sharePublic.Use(sharePreviewLimiter)
			}
			sharePublic.GET("/preview", h.GetPublicSharePreview)
			if deps.FeedbackService != nil {
				feedbackHandler := NewFeedbackHandler(deps.FeedbackService)
				feedbackHandler.RegisterPublicSupportRoutes(public)
			}
		}

	}

	deps.Logger.Info("router configured successfully")
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
	if !BindJSON(c, &req) {
		return
	}

	resp, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, resp)
}

// Login 用户登录
func (h *Handler) Login(c *gin.Context) {
	var req service.LoginRequest
	if !BindJSON(c, &req) {
		return
	}

	// 提取登录信息
	userAgent := c.Request.UserAgent()
	device, os, browser := utils.ParseUserAgent(userAgent)
	ipAddress := utils.GetClientIP(
		c.Request.RemoteAddr,
		c.GetHeader("X-Forwarded-For"),
		c.GetHeader("X-Real-IP"),
	)

	loginInfo := &service.LoginInfo{
		IPAddress: ipAddress,
		Location:  "", // 可以通过 IP 地址查询地理位置，这里先留空
		Device:    device,
		OS:        os,
		Browser:   browser,
		UserAgent: userAgent,
	}

	resp, err := h.svc.Login(c.Request.Context(), &req, loginInfo)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, resp)
}

// ChangePassword 修改密码
func (h *Handler) ChangePassword(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	// Backward/forward compatible payload:
	// - iOS currently sends old_password/new_password
	// - backend service expects oldPassword/newPassword
	var req struct {
		OldPasswordCamel string `json:"oldPassword"`
		NewPasswordCamel string `json:"newPassword"`
		OldPasswordSnake string `json:"old_password"`
		NewPasswordSnake string `json:"new_password"`
	}
	if !BindJSON(c, &req) {
		return
	}

	oldPassword := req.OldPasswordCamel
	if oldPassword == "" {
		oldPassword = req.OldPasswordSnake
	}
	newPassword := req.NewPasswordCamel
	if newPassword == "" {
		newPassword = req.NewPasswordSnake
	}
	svcReq := &service.ChangePasswordRequest{
		OldPassword: oldPassword,
		NewPassword: newPassword,
	}

	if err := h.svc.ChangePassword(c.Request.Context(), userID, svcReq); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "password changed successfully"})
}

// RequestPasswordReset 请求密码重置
func (h *Handler) RequestPasswordReset(c *gin.Context) {
	var req service.PasswordResetRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.svc.RequestPasswordReset(c.Request.Context(), &req); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "password reset email sent"})
}

// ResetPassword 重置密码
func (h *Handler) ResetPassword(c *gin.Context) {
	// Backward/forward compatible payload:
	// - backend service expects: { token, newPassword }
	// - Android currently sends: { resetToken, password }
	// - iOS currently sends: { token, password }
	var req struct {
		Token       string `json:"token"`
		ResetToken  string `json:"resetToken"`
		NewPassword string `json:"newPassword"`
		Password    string `json:"password"`
	}
	if !BindJSON(c, &req) {
		return
	}

	token := req.Token
	if token == "" {
		token = req.ResetToken
	}
	newPassword := req.NewPassword
	if newPassword == "" {
		newPassword = req.Password
	}
	svcReq := &service.PasswordResetConfirm{
		Token:       token,
		NewPassword: newPassword,
	}

	if err := h.svc.ResetPassword(c.Request.Context(), svcReq); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "password reset successfully"})
}

// SendEmailVerificationCode sends a 6-digit email verification code.
// POST /api/auth/email/send-verification-code
func (h *Handler) SendEmailVerificationCode(c *gin.Context) {
	var req service.EmailVerificationSendRequest
	if !BindJSON(c, &req) {
		return
	}

	ipAddress := utils.GetClientIP(
		c.Request.RemoteAddr,
		c.GetHeader("X-Forwarded-For"),
		c.GetHeader("X-Real-IP"),
	)

	if err := h.svc.SendEmailVerificationCode(c.Request.Context(), &req, ipAddress); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "verification code sent"})
}

// VerifyEmail verifies email via 6-digit code.
// POST /api/auth/email/verify
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req service.EmailVerificationConfirmRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.svc.VerifyEmailByCode(c.Request.Context(), &req); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "email verified"})
}

// RefreshToken 刷新访问令牌
func (h *Handler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refreshToken" binding:"required"`
	}

	if !BindJSON(c, &req) {
		return
	}

	resp, err := h.svc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, resp)
}

// CurrentUser 获取当前用户信息
func (h *Handler) CurrentUser(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	user, err := h.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, user)
}

// ========== 故事默认路径相关端点 ==========

// SetDefaultPathRequest 设置默认路径请求
type SetDefaultPathRequest struct {
	NodeIDs []string `json:"nodeIds" binding:"required"`
}

// SetDefaultPath 设置故事的默认路径
func (h *Handler) SetDefaultPath(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID := c.Param("id")

	var req SetDefaultPathRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.storyboardPathSvc.SetDefaultPath(c.Request.Context(), storyID, userID, req.NodeIDs); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "default path set successfully"})
}

// CalculateAutoPath 自动计算默认路径
func (h *Handler) CalculateAutoPath(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	storyID := c.Param("id")

	path, err := h.storyboardPathSvc.CalculateAutoPath(c.Request.Context(), storyID, userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{
		"nodeIds": path,
		"count":   len(path),
	})
}

// GetDefaultPath 获取故事的默认路径
func (h *Handler) GetDefaultPath(c *gin.Context) {
	storyID := c.Param("id")

	path, err := h.storyboardPathSvc.GetDefaultPath(c.Request.Context(), storyID)
	if err != nil {
		HandleError(c, err)
		return
	}

	h.attachStoryboardIsLikedMany(c, path)
	Success(c, gin.H{
		"path":  path,
		"count": len(path),
	})
}

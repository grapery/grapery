package http

import (
	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// Handler handles HTTP requests
type Handler struct {
	svc                 *service.Service
	aiService           *service.AIService
	writersRoomSvc      *service.WritersRoomService
	storyboardPathSvc   *service.StoryboardPathService
	logger              *zap.Logger
}

// HandlerDependencies 包含所有 handler 依赖的服务
type HandlerDependencies struct {
	Service               *service.Service
	AIService             *service.AIService
	WritersRoomService    *service.WritersRoomService
	StoryboardPathService *service.StoryboardPathService
	InteractionService    service.InteractionService
	UserSettingsService   service.UserSettingsService
	GroupShowcaseService  GroupShowcaseService
	Logger                *zap.Logger
}

// NewHandler creates a new HTTP handler (legacy constructor)
func NewHandler(svc *service.Service, aiService *service.AIService, writersRoomSvc *service.WritersRoomService, logger *zap.Logger) *Handler {
	return &Handler{
		svc:            svc,
		aiService:      aiService,
		writersRoomSvc: writersRoomSvc,
		logger:         logger,
	}
}

// NewHandlerWithDeps creates a new HTTP handler with all dependencies
func NewHandlerWithDeps(deps *HandlerDependencies) *Handler {
	return &Handler{
		svc:                 deps.Service,
		aiService:           deps.AIService,
		writersRoomSvc:      deps.WritersRoomService,
		storyboardPathSvc:   deps.StoryboardPathService,
		logger:              deps.Logger,
	}
}

// SetupRouter configures and returns the Gin router
func SetupRouter(deps *HandlerDependencies) *gin.Engine {
	h := NewHandlerWithDeps(deps)
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
			auth.POST("/email/send-verification-code", h.SendEmailVerificationCode)
			auth.POST("/email/verify", h.VerifyEmail)
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
			authenticated.GET("/users/:id/storyboards", h.GetUserStoryboards)
			authenticated.GET("/users/:id/drafts", h.GetUserDrafts)
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
			// Dashboard storyboard feeds (authenticated)
			authenticated.GET("/dashboard/storyboards", h.GetDashboardStoryboards)
			authenticated.GET("/dashboard/groups/storyboards", h.GetDashboardGroupStoryboards)
			authenticated.GET("/dashboard/characters/storyboards", h.GetDashboardCharacterStoryboards)
			authenticated.GET("/dashboard/trending/storyboards", h.GetTrendingStoryboards)
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
			authenticated.GET("/characters/:id/posters", h.GetCharacterPosters)
			authenticated.GET("/characters/:id/storyboards", h.GetCharacterStoryboards)

			// 群组相关
			authenticated.GET("/groups", h.ListGroups)
			authenticated.GET("/groups/:id", h.GetGroup)
			authenticated.GET("/groups/:id/members", h.GetGroupMembers)
			authenticated.GET("/groups/:id/membership", h.CheckGroupMembership) // 检查成员资格
			authenticated.GET("/groups/:id/activities", h.GetGroupActivities)
			authenticated.GET("/groups/:id/activities/heatmap", h.GetGroupActivityHeatmap)
			authenticated.GET("/groups/:id/stories", h.GetGroupStories) // 获取群组的故事列表
			authenticated.GET("/groups/followed", h.ListFollowedGroups) // 获取用户关注的群组列表
			// 获取全局活动
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

			// Story Scenes 相关
			authenticated.GET("/stories/:id/scenes", h.ListStoryScenes)
			authenticated.POST("/stories/:id/scenes", h.CreateStoryScene)
			authenticated.POST("/stories/:id/scenes/register-image", h.UploadSceneImage)
			authenticated.POST("/stories/:id/scenes/ai-generate-image", h.GenerateSceneImage)
			authenticated.PUT("/stories/:id/scenes/:sceneId", h.UpdateStoryScene)
			authenticated.DELETE("/stories/:id/scenes/:sceneId", h.DeleteStoryScene)

			// 故事默认路径相关
			authenticated.POST("/stories/:id/default-path", h.SetDefaultPath)
			authenticated.POST("/stories/:id/default-path/auto", h.CalculateAutoPath)
			authenticated.GET("/stories/:id/default-path", h.GetDefaultPath)

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
			authenticated.POST("/storyboards/:id/generate/images", h.GenerateAllStoryboardImages)
			authenticated.POST("/storyboards/:id/generate/video", h.GenerateStoryboardVideo)
			authenticated.GET("/storyboards/:id/generation-progress", h.GetGenerationProgress)
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
			authenticated.POST("/characters/:id/generate-avatar", h.GenerateCharacterAvatar)     // AI生成角色头像
			authenticated.PUT("/characters/:id/avatar", h.UpdateCharacterAvatar)                 // 更新角色头像
			authenticated.PUT("/characters/:id/use-portrait-as-avatar", h.UsePortraitAsAvatar)   // 使用portrait作为头像
			authenticated.GET("/characters/:id/portrait-prompt", h.GetPortraitPrompt)            // 获取形象生成推荐提示词
			authenticated.POST("/characters/:id/generate-portrait", h.GenerateCharacterPortrait) // AI生成角色完整形象
			authenticated.POST("/characters/:id/crop-avatar", h.CropAvatarFromPortrait)          // 从形象图裁剪头像
			authenticated.POST("/characters/:id/posters", h.CreateCharacterPoster)
			authenticated.POST("/posters/:id/generate", h.GenerateCharacterPoster) // AI两步生成海报
			authenticated.POST("/posters/:id/publish", h.PublishCharacterPoster)   // 发布海报
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
			authenticated.POST("/groups/:id/members/:userId/role-by-code", h.UpdateMemberRoleByCode) // 使用角色代码更新角色
			authenticated.POST("/groups/:id/leave", h.LeaveGroup)
			authenticated.POST("/groups/:id/follow", h.FollowGroup)                    // 关注群组
			authenticated.DELETE("/groups/:id/follow", h.UnfollowGroup)                // 取消关注群组
			authenticated.GET("/groups/roles", h.ListGroupRoles)                       // 获取所有角色列表
			authenticated.GET("/groups/roles/:code", h.GetGroupRoleByCode)             // 根据代码获取角色
			authenticated.GET("/groups/roles/:code/permissions", h.GetRolePermissions) // 获取角色权限
			authenticated.POST("/groups/roles/initialize", h.InitializeGroupRoles)     // 初始化系统角色

			// 邀请相关
			authenticated.GET("/invitations/pending", h.GetPendingInvitations)
			authenticated.POST("/invitations/:id/accept", h.AcceptInvitation)
			authenticated.POST("/invitations/:id/reject", h.RejectInvitation)

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
			authenticated.GET("/sse/notifications", h.SSENotificationStream)
			authenticated.GET("/sse/activities", h.SSEActivityStream)

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
			authenticated.POST("/ai/generate-character", h.GenerateCharacter) // iOS兼容端点
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

			// 互动相关 (Interaction)
			interactionHandler := NewInteractionHandler(deps.InteractionService)
			interactionHandler.RegisterInteractionRoutes(authenticated)

			// 用户设置相关 (User Settings)
			userSettingsHandler := NewUserSettingsHandler(deps.UserSettingsService)
			userSettingsHandler.RegisterUserSettingsRoutes(authenticated)

			// 小组展示相关 (Group Showcases)
			showcaseHandler := NewGroupShowcaseHandler(deps.GroupShowcaseService)
			authenticated.GET("/groups/:id/showcases", showcaseHandler.GetGroupShowcases)
			authenticated.POST("/groups/:id/showcases", showcaseHandler.AddShowcase)
			authenticated.DELETE("/groups/:id/showcases/:showcaseId", showcaseHandler.RemoveShowcase)
			authenticated.PUT("/groups/:id/showcases/:showcaseId/order", showcaseHandler.UpdateShowcaseOrder)

			// 碎片相关 (Fragments)
			authenticated.POST("/fragments/:id/convert-to-story", h.ConvertFragmentToStory)
		}

		// 公开接口（无需认证）
		public := api.Group("")
		{
			public.POST("/invitation-codes/validate", h.ValidateInvitationCode) // 验证邀请码（注册前验证）
			// Public Trending (guest-accessible)
			public.GET("/public/stories/trending", h.GetTrendingStoriesPublic)
			public.GET("/public/trending/storyboards", h.GetPublicTrendingStoryboards)
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

	Success(c, gin.H{
		"path":  path,
		"count": len(path),
	})
}

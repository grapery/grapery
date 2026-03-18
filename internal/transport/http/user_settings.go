package http

import (
	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// UserSettingsHandler 用户设置处理器
type UserSettingsHandler struct {
	settingsService service.UserSettingsService
}

// NewUserSettingsHandler 创建用户设置处理器
func NewUserSettingsHandler(settingsService service.UserSettingsService) *UserSettingsHandler {
	return &UserSettingsHandler{settingsService: settingsService}
}

// RegisterUserSettingsRoutes 注册用户设置相关路由
func (h *UserSettingsHandler) RegisterUserSettingsRoutes(r *gin.RouterGroup) {
	settings := r.Group("/settings")
	{
		settings.GET("", h.GetSettings)
		settings.PUT("", h.UpdateSettings)
		settings.PUT("/language", h.UpdateLanguage)
		settings.PUT("/theme", h.UpdateTheme)
		settings.PUT("/font-size", h.UpdateFontSize)
		settings.PUT("/privacy", h.UpdatePrivacy)
		settings.PUT("/ai", h.UpdateAISettings)
		settings.PUT("/notifications", h.UpdateNotificationSettings)
	}
}

// GetSettings 获取用户设置
func (h *UserSettingsHandler) GetSettings(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	settings, err := h.settingsService.GetSettings(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, settings)
}

// UpdateSettingsRequest 更新设置请求
type UpdateSettingsRequest struct {
	Language                  string                 `json:"language,omitempty"`
	Theme                     string                 `json:"theme,omitempty"`
	FontSize                  string                 `json:"fontSize,omitempty"`
	DataSaver                 *bool                  `json:"dataSaver,omitempty"`
	ProfileVisibility         string                 `json:"profileVisibility,omitempty"`
	DefaultStoryVisibility    string                 `json:"defaultStoryVisibility,omitempty"`
	DefaultFragmentVisibility string                 `json:"defaultFragmentVisibility,omitempty"`
	AllowFollowFrom           string                 `json:"allowFollowFrom,omitempty"`
	AllowCommentsFrom         string                 `json:"allowCommentsFrom,omitempty"`
	AllowMessagesFrom         string                 `json:"allowMessagesFrom,omitempty"`
	ShowOnlineStatus          *bool                  `json:"showOnlineStatus,omitempty"`
	ShowReadReceipts          *bool                  `json:"showReadReceipts,omitempty"`
	ShowPublicStories         *bool                  `json:"showPublicStories,omitempty"`
	ShowPublicFragments       *bool                  `json:"showPublicFragments,omitempty"`
	ShowPublicBookmarks       *bool                  `json:"showPublicBookmarks,omitempty"`
	AIEnabled                 *bool                  `json:"aiEnabled,omitempty"`
	AIDataSharing             *bool                  `json:"aiDataSharing,omitempty"`
	NotificationSettings      map[string]interface{} `json:"notificationSettings,omitempty"`
}

// UpdateSettings 更新用户设置
func (h *UserSettingsHandler) UpdateSettings(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req UpdateSettingsRequest
	if !BindJSON(c, &req) {
		return
	}

	updates := make(map[string]interface{})
	if req.Language != "" {
		updates["language"] = req.Language
	}
	if req.Theme != "" {
		updates["theme"] = req.Theme
	}
	if req.FontSize != "" {
		updates["fontSize"] = req.FontSize
	}
	if req.DataSaver != nil {
		updates["dataSaver"] = *req.DataSaver
	}
	if req.ProfileVisibility != "" {
		updates["profileVisibility"] = req.ProfileVisibility
	}
	if req.DefaultStoryVisibility != "" {
		updates["defaultStoryVisibility"] = req.DefaultStoryVisibility
	}
	if req.DefaultFragmentVisibility != "" {
		updates["defaultFragmentVisibility"] = req.DefaultFragmentVisibility
	}
	if req.AllowFollowFrom != "" {
		updates["allowFollowFrom"] = req.AllowFollowFrom
	}
	if req.AllowCommentsFrom != "" {
		updates["allowCommentsFrom"] = req.AllowCommentsFrom
	}
	if req.AllowMessagesFrom != "" {
		updates["allowMessagesFrom"] = req.AllowMessagesFrom
	}
	if req.ShowOnlineStatus != nil {
		updates["showOnlineStatus"] = *req.ShowOnlineStatus
	}
	if req.ShowReadReceipts != nil {
		updates["showReadReceipts"] = *req.ShowReadReceipts
	}
	if req.ShowPublicStories != nil {
		updates["showPublicStories"] = *req.ShowPublicStories
	}
	if req.ShowPublicFragments != nil {
		updates["showPublicFragments"] = *req.ShowPublicFragments
	}
	if req.ShowPublicBookmarks != nil {
		updates["showPublicBookmarks"] = *req.ShowPublicBookmarks
	}
	if req.AIEnabled != nil {
		updates["aiEnabled"] = *req.AIEnabled
	}
	if req.AIDataSharing != nil {
		updates["aiDataSharing"] = *req.AIDataSharing
	}
	if req.NotificationSettings != nil {
		updates["notificationSettings"] = req.NotificationSettings
	}

	settings, err := h.settingsService.UpdateSettings(c.Request.Context(), userID, updates)
	if err != nil {
		HandleError(c, err)
		return
	}

	Success(c, settings)
}

// UpdateLanguageRequest 更新语言请求
type UpdateLanguageRequest struct {
	Language string `json:"language" binding:"required"`
}

// UpdateLanguage 更新语言
func (h *UserSettingsHandler) UpdateLanguage(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req UpdateLanguageRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.settingsService.UpdateLanguage(c.Request.Context(), userID, req.Language); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "language updated successfully"})
}

// UpdateThemeRequest 更新主题请求
type UpdateThemeRequest struct {
	Theme string `json:"theme" binding:"required"`
}

// UpdateTheme 更新主题
func (h *UserSettingsHandler) UpdateTheme(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req UpdateThemeRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.settingsService.UpdateTheme(c.Request.Context(), userID, req.Theme); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "theme updated successfully"})
}

// UpdateFontSizeRequest 更新字体大小请求
type UpdateFontSizeRequest struct {
	FontSize string `json:"fontSize" binding:"required"`
}

// UpdateFontSize 更新字体大小
func (h *UserSettingsHandler) UpdateFontSize(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req UpdateFontSizeRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.settingsService.UpdateFontSize(c.Request.Context(), userID, req.FontSize); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "font size updated successfully"})
}

// UpdatePrivacyRequest 更新隐私请求
type UpdatePrivacyRequest struct {
	ProfileVisibility         string `json:"profileVisibility,omitempty"`
	DefaultStoryVisibility    string `json:"defaultStoryVisibility,omitempty"`
	DefaultFragmentVisibility string `json:"defaultFragmentVisibility,omitempty"`
	AllowFollowFrom           string `json:"allowFollowFrom,omitempty"`
	AllowCommentsFrom         string `json:"allowCommentsFrom,omitempty"`
	AllowMessagesFrom         string `json:"allowMessagesFrom,omitempty"`
	ShowPublicStories         *bool  `json:"showPublicStories,omitempty"`
	ShowPublicFragments       *bool  `json:"showPublicFragments,omitempty"`
	ShowPublicBookmarks       *bool  `json:"showPublicBookmarks,omitempty"`
}

// UpdatePrivacy 更新隐私设置
func (h *UserSettingsHandler) UpdatePrivacy(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req UpdatePrivacyRequest
	if !BindJSON(c, &req) {
		return
	}

	privacy := make(map[string]interface{})
	if req.ProfileVisibility != "" {
		privacy["profileVisibility"] = req.ProfileVisibility
	}
	if req.DefaultStoryVisibility != "" {
		privacy["defaultStoryVisibility"] = req.DefaultStoryVisibility
	}
	if req.DefaultFragmentVisibility != "" {
		privacy["defaultFragmentVisibility"] = req.DefaultFragmentVisibility
	}
	if req.AllowFollowFrom != "" {
		privacy["allowFollowFrom"] = req.AllowFollowFrom
	}
	if req.AllowCommentsFrom != "" {
		privacy["allowCommentsFrom"] = req.AllowCommentsFrom
	}
	if req.AllowMessagesFrom != "" {
		privacy["allowMessagesFrom"] = req.AllowMessagesFrom
	}
	if req.ShowPublicStories != nil {
		privacy["showPublicStories"] = *req.ShowPublicStories
	}
	if req.ShowPublicFragments != nil {
		privacy["showPublicFragments"] = *req.ShowPublicFragments
	}
	if req.ShowPublicBookmarks != nil {
		privacy["showPublicBookmarks"] = *req.ShowPublicBookmarks
	}

	if err := h.settingsService.UpdatePrivacy(c.Request.Context(), userID, privacy); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "privacy settings updated successfully"})
}

// UpdateAISettingsRequest 更新AI设置请求
type UpdateAISettingsRequest struct {
	AIEnabled     bool `json:"aiEnabled"`
	AIDataSharing bool `json:"aiDataSharing"`
}

// UpdateAISettings 更新AI设置
func (h *UserSettingsHandler) UpdateAISettings(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var req UpdateAISettingsRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.settingsService.UpdateAISettings(c.Request.Context(), userID, req.AIEnabled, req.AIDataSharing); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "AI settings updated successfully"})
}

// UpdateNotificationSettings 更新通知设置
func (h *UserSettingsHandler) UpdateNotificationSettings(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	var settings map[string]interface{}
	if !BindJSON(c, &settings) {
		return
	}

	if err := h.settingsService.UpdateNotificationSettings(c.Request.Context(), userID, settings); err != nil {
		HandleError(c, err)
		return
	}

	Success(c, gin.H{"message": "notification settings updated successfully"})
}

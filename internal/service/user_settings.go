package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// UserSettingsService 用户设置服务接口
type UserSettingsService interface {
	GetSettings(ctx context.Context, userID string) (*domain.UserSettings, error)
	CreateDefaultSettings(ctx context.Context, userID string) (*domain.UserSettings, error)
	UpdateSettings(ctx context.Context, userID string, updates map[string]interface{}) (*domain.UserSettings, error)
	UpdateLanguage(ctx context.Context, userID string, language string) error
	UpdateTheme(ctx context.Context, userID string, theme string) error
	UpdateFontSize(ctx context.Context, userID string, fontSize string) error
	UpdatePrivacy(ctx context.Context, userID string, privacy map[string]string) error
	UpdateAISettings(ctx context.Context, userID string, aiEnabled, aiDataSharing bool) error
	UpdateNotificationSettings(ctx context.Context, userID string, settings map[string]interface{}) error
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error
}

// userSettingsService 用户设置服务实现
type userSettingsService struct {
	settingsRepo domain.UserSettingsRepository
	logger       *zap.Logger
}

// NewUserSettingsService 创建用户设置服务
func NewUserSettingsService(settingsRepo domain.UserSettingsRepository, logger *zap.Logger) UserSettingsService {
	return &userSettingsService{
		settingsRepo: settingsRepo,
		logger:       logger,
	}
}

// GetSettings 获取用户设置
func (s *userSettingsService) GetSettings(ctx context.Context, userID string) (*domain.UserSettings, error) {
	s.logger.Debug("getting user settings",
		zap.String("userID", userID))

	settings, err := s.settingsRepo.GetUserSettings(userID)
	if err != nil {
		// 如果不存在，创建默认设置
		// 注意：这里我们检查错误类型来判断记录是否存在
		// 实际项目中可能需要更精确的错误判断
		s.logger.Info("user settings not found, creating default",
			zap.String("userID", userID),
			zap.Error(err))
		return s.CreateDefaultSettings(ctx, userID)
	}

	s.logger.Debug("user settings retrieved",
		zap.String("userID", userID))
	return settings, nil
}

// CreateDefaultSettings 创建默认设置
func (s *userSettingsService) CreateDefaultSettings(ctx context.Context, userID string) (*domain.UserSettings, error) {
	s.logger.Info("creating default user settings",
		zap.String("userID", userID))

	defaultSettings := &domain.UserSettings{
		BaseModel: common.BaseModel{
			ID:        utils.GenerateID(),
			CreatedAt: time.Now().Unix(),
			UpdatedAt: time.Now().Unix(),
		},
		UserID:                    userID,
		Language:                  string(domain.LanguageChineseCN),
		Theme:                     string(domain.ThemeSystem),
		FontSize:                  string(domain.FontSizeMedium),
		DataSaver:                 false,
		ProfileVisibility:         string(domain.VisibilityPublic),
		DefaultStoryVisibility:    string(domain.VisibilityPublic),
		DefaultFragmentVisibility: string(domain.VisibilityPublic),
		AllowFollowFrom:           string(domain.AllowFromEveryone),
		AllowCommentsFrom:         string(domain.AllowFromEveryone),
		AllowMessagesFrom:         string(domain.AllowFromFollowersOnly),
		ShowOnlineStatus:          true,
		ShowReadReceipts:          true,
		AIEnabled:                 true,
		AIDataSharing:             true,
		NotificationSettings:      s.getDefaultNotificationSettings(),
	}

	if err := s.settingsRepo.CreateUserSettings(defaultSettings); err != nil {
		s.logger.Error("failed to create default user settings",
			zap.Error(err),
			zap.String("userID", userID))
		return nil, fmt.Errorf("failed to create default settings: %w", err)
	}

	s.logger.Info("default user settings created",
		zap.String("userID", userID))
	return defaultSettings, nil
}

// UpdateSettings 更新设置（通用）
func (s *userSettingsService) UpdateSettings(ctx context.Context, userID string, updates map[string]interface{}) (*domain.UserSettings, error) {
	s.logger.Info("updating user settings",
		zap.String("userID", userID))

	// 获取当前设置
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get settings for update",
			zap.Error(err),
			zap.String("userID", userID))
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	// 应用更新
	if language, ok := updates["language"].(string); ok && language != "" {
		if s.isValidLanguage(language) {
			settings.Language = language
		}
	}
	if theme, ok := updates["theme"].(string); ok && theme != "" {
		if s.isValidTheme(theme) {
			settings.Theme = theme
		}
	}
	if fontSize, ok := updates["fontSize"].(string); ok && fontSize != "" {
		if s.isValidFontSize(fontSize) {
			settings.FontSize = fontSize
		}
	}
	if dataSaver, ok := updates["dataSaver"].(bool); ok {
		settings.DataSaver = dataSaver
	}
	if profileVisibility, ok := updates["profileVisibility"].(string); ok && profileVisibility != "" {
		if s.isValidVisibility(profileVisibility) {
			settings.ProfileVisibility = profileVisibility
		}
	}
	if defaultStoryVisibility, ok := updates["defaultStoryVisibility"].(string); ok && defaultStoryVisibility != "" {
		if s.isValidVisibility(defaultStoryVisibility) {
			settings.DefaultStoryVisibility = defaultStoryVisibility
		}
	}
	if defaultFragmentVisibility, ok := updates["defaultFragmentVisibility"].(string); ok && defaultFragmentVisibility != "" {
		if s.isValidVisibility(defaultFragmentVisibility) {
			settings.DefaultFragmentVisibility = defaultFragmentVisibility
		}
	}
	if allowFollowFrom, ok := updates["allowFollowFrom"].(string); ok && allowFollowFrom != "" {
		if s.isValidAllowFrom(allowFollowFrom) {
			settings.AllowFollowFrom = allowFollowFrom
		}
	}
	if allowCommentsFrom, ok := updates["allowCommentsFrom"].(string); ok && allowCommentsFrom != "" {
		if s.isValidAllowFrom(allowCommentsFrom) {
			settings.AllowCommentsFrom = allowCommentsFrom
		}
	}
	if allowMessagesFrom, ok := updates["allowMessagesFrom"].(string); ok && allowMessagesFrom != "" {
		if s.isValidAllowFrom(allowMessagesFrom) {
			settings.AllowMessagesFrom = allowMessagesFrom
		}
	}
	if showOnlineStatus, ok := updates["showOnlineStatus"].(bool); ok {
		settings.ShowOnlineStatus = showOnlineStatus
	}
	if showReadReceipts, ok := updates["showReadReceipts"].(bool); ok {
		settings.ShowReadReceipts = showReadReceipts
	}
	if aiEnabled, ok := updates["aiEnabled"].(bool); ok {
		settings.AIEnabled = aiEnabled
	}
	if aiDataSharing, ok := updates["aiDataSharing"].(bool); ok {
		settings.AIDataSharing = aiDataSharing
	}

	settings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
		s.logger.Error("failed to update user settings",
			zap.Error(err),
			zap.String("userID", userID))
		return nil, fmt.Errorf("failed to update settings: %w", err)
	}

	s.logger.Info("user settings updated",
		zap.String("userID", userID))
	return settings, nil
}

// UpdateLanguage 更新语言设置
func (s *userSettingsService) UpdateLanguage(ctx context.Context, userID string, language string) error {
	s.logger.Info("updating language",
		zap.String("userID", userID),
		zap.String("language", language))

	if !s.isValidLanguage(language) {
		return fmt.Errorf("invalid language: %s", language)
	}

	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}

	settings.Language = language
	settings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
		s.logger.Error("failed to update language",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to update language: %w", err)
	}

	return nil
}

// UpdateTheme 更新主题设置
func (s *userSettingsService) UpdateTheme(ctx context.Context, userID string, theme string) error {
	s.logger.Info("updating theme",
		zap.String("userID", userID),
		zap.String("theme", theme))

	if !s.isValidTheme(theme) {
		return fmt.Errorf("invalid theme: %s", theme)
	}

	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}

	settings.Theme = theme
	settings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
		s.logger.Error("failed to update theme",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to update theme: %w", err)
	}

	return nil
}

// UpdatePrivacy 更新隐私设置
func (s *userSettingsService) UpdatePrivacy(ctx context.Context, userID string, privacy map[string]string) error {
	s.logger.Info("updating privacy settings",
		zap.String("userID", userID))

	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}

	// 更新隐私设置
	if profileVisibility, ok := privacy["profileVisibility"]; ok && profileVisibility != "" {
		if s.isValidVisibility(profileVisibility) {
			settings.ProfileVisibility = profileVisibility
		}
	}
	if defaultStoryVisibility, ok := privacy["defaultStoryVisibility"]; ok && defaultStoryVisibility != "" {
		if s.isValidVisibility(defaultStoryVisibility) {
			settings.DefaultStoryVisibility = defaultStoryVisibility
		}
	}
	if defaultFragmentVisibility, ok := privacy["defaultFragmentVisibility"]; ok && defaultFragmentVisibility != "" {
		if s.isValidVisibility(defaultFragmentVisibility) {
			settings.DefaultFragmentVisibility = defaultFragmentVisibility
		}
	}
	if allowFollowFrom, ok := privacy["allowFollowFrom"]; ok && allowFollowFrom != "" {
		if s.isValidAllowFrom(allowFollowFrom) {
			settings.AllowFollowFrom = allowFollowFrom
		}
	}
	if allowCommentsFrom, ok := privacy["allowCommentsFrom"]; ok && allowCommentsFrom != "" {
		if s.isValidAllowFrom(allowCommentsFrom) {
			settings.AllowCommentsFrom = allowCommentsFrom
		}
	}
	if allowMessagesFrom, ok := privacy["allowMessagesFrom"]; ok && allowMessagesFrom != "" {
		if s.isValidAllowFrom(allowMessagesFrom) {
			settings.AllowMessagesFrom = allowMessagesFrom
		}
	}

	settings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
		s.logger.Error("failed to update privacy settings",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to update privacy settings: %w", err)
	}

	return nil
}

// UpdateAISettings 更新AI设置
func (s *userSettingsService) UpdateAISettings(ctx context.Context, userID string, aiEnabled, aiDataSharing bool) error {
	s.logger.Info("updating AI settings",
		zap.String("userID", userID),
		zap.Bool("aiEnabled", aiEnabled),
		zap.Bool("aiDataSharing", aiDataSharing))

	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}

	settings.AIEnabled = aiEnabled
	settings.AIDataSharing = aiDataSharing
	settings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
		s.logger.Error("failed to update AI settings",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to update AI settings: %w", err)
	}

	return nil
}

// UpdateNotificationSettings 更新通知设置
func (s *userSettingsService) UpdateNotificationSettings(ctx context.Context, userID string, settings map[string]interface{}) error {
	s.logger.Info("updating notification settings",
		zap.String("userID", userID))

	userSettings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}

	// 将新的通知设置合并到现有设置中
	var existingSettings map[string]interface{}
	if err := json.Unmarshal([]byte(userSettings.NotificationSettings), &existingSettings); err != nil {
		// 如果解析失败，使用默认设置
		existingSettings = s.getDefaultNotificationSettingsMap()
	}

	// 合并设置
	for key, value := range settings {
		existingSettings[key] = value
	}

	// 序列化回 JSON
	jsonBytes, err := json.Marshal(existingSettings)
	if err != nil {
		s.logger.Error("failed to marshal notification settings",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to marshal notification settings: %w", err)
	}

	userSettings.NotificationSettings = string(jsonBytes)
	userSettings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(userSettings); err != nil {
		s.logger.Error("failed to update notification settings",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to update notification settings: %w", err)
	}

	return nil
}

// getDefaultNotificationSettings 获取默认通知设置（JSON字符串）
func (s *userSettingsService) getDefaultNotificationSettings() string {
	settings := s.getDefaultNotificationSettingsMap()
	jsonBytes, _ := json.Marshal(settings)
	return string(jsonBytes)
}

// getDefaultNotificationSettingsMap 获取默认通知设置（map）
func (s *userSettingsService) getDefaultNotificationSettingsMap() map[string]interface{} {
	return map[string]interface{}{
		"push": map[string]bool{
			"enabled":            true,
			"newFollower":        true,
			"newLike":            true,
			"newComment":         true,
			"storyUpdate":        true,
			"directMessage":      true,
			"systemAnnouncement": true,
		},
		"email": map[string]bool{
			"enabled":       true,
			"weeklyDigest":  true,
			"securityAlert": true,
			"marketing":     false,
		},
		"inApp": map[string]interface{}{
			"enabled":      true,
			"showPreview":  true,
			"soundEnabled": true,
		},
	}
}

// 验证辅助函数

func (s *userSettingsService) isValidLanguage(language string) bool {
	switch domain.LanguageType(language) {
	case domain.LanguageSystem, domain.LanguageEnglish, domain.LanguageChineseCN, domain.LanguageJapanese:
		return true
	default:
		return false
	}
}

func (s *userSettingsService) isValidTheme(theme string) bool {
	switch domain.ThemeType(theme) {
	case domain.ThemeLight, domain.ThemeDark, domain.ThemeSystem:
		return true
	default:
		return false
	}
}

func (s *userSettingsService) isValidFontSize(fontSize string) bool {
	switch domain.FontSizeType(fontSize) {
	case domain.FontSizeSmall, domain.FontSizeMedium, domain.FontSizeLarge:
		return true
	default:
		return false
	}
}

func (s *userSettingsService) isValidVisibility(visibility string) bool {
	switch domain.VisibilityType(visibility) {
	case domain.VisibilityPublic, domain.VisibilityFollowersOnly, domain.VisibilityPrivate, domain.VisibilityUnlisted:
		return true
	default:
		return false
	}
}

func (s *userSettingsService) isValidAllowFrom(allowFrom string) bool {
	switch domain.AllowFromType(allowFrom) {
	case domain.AllowFromEveryone, domain.AllowFromFollowersOnly, domain.AllowFromFollowersOfFollowers, domain.AllowFromNoOne:
		return true
	default:
		return false
	}
}


// UpdateFontSize 更新字体大小
func (s *userSettingsService) UpdateFontSize(ctx context.Context, userID string, fontSize string) error {
	s.logger.Info("updating font size",
		zap.String("userID", userID),
		zap.String("fontSize", fontSize))

	if !s.isValidFontSize(fontSize) {
		return fmt.Errorf("invalid font size: %s", fontSize)
	}

	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}

	settings.FontSize = fontSize
	settings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
		s.logger.Error("failed to update font size",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to update font size: %w", err)
	}

	return nil
}

// ChangePassword 修改密码
func (s *userSettingsService) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	s.logger.Info("changing password",
		zap.String("userID", userID))

	// 验证密码长度
	if len(newPassword) < 8 {
		return fmt.Errorf("new password must be at least 8 characters long")
	}

	// 调用 user service 验证当前密码并更新
	// 这个需要在 UserService 中实现，这里先保留接口
	s.logger.Warn("change password requires UserService integration",
		zap.String("userID", userID))

	return fmt.Errorf("change password not fully implemented yet")
}

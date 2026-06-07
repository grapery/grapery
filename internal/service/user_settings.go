package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/recommendation"
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
	UpdatePrivacy(ctx context.Context, userID string, privacy map[string]interface{}) error
	UpdateAISettings(ctx context.Context, userID string, aiEnabled, aiDataSharing bool) error
	UpdateNotificationSettings(ctx context.Context, userID string, settings map[string]interface{}) error
	UpdatePreferredGenres(ctx context.Context, userID string, genres []string) ([]string, error)
	// GetPreferredGenresPreferences 返回当前用户已选体裁与服务端允许的 slug 列表（供设置页展示）。
	GetPreferredGenresPreferences(ctx context.Context, userID string) (preferred []string, allowed []string, err error)
	ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error
}

// userSettingsService 用户设置服务实现
type userSettingsService struct {
	settingsRepo     domain.UserSettingsRepository
	genreCatalogRepo domain.GenreCatalogRepository
	logger           *zap.Logger
	cache            cache.Cache
}

func userSettingsCacheKey(userID string) string {
	return "user_settings:" + userID
}

func (s *userSettingsService) invalidateSettingsCache(ctx context.Context, userID string) {
	if s.cache == nil {
		return
	}
	_ = s.cache.Delete(ctx, userSettingsCacheKey(userID))
}

// NewUserSettingsService 创建用户设置服务
func NewUserSettingsService(settingsRepo domain.UserSettingsRepository, genreCatalogRepo domain.GenreCatalogRepository, logger *zap.Logger, c cache.Cache) UserSettingsService {
	return &userSettingsService{
		settingsRepo:     settingsRepo,
		genreCatalogRepo: genreCatalogRepo,
		logger:           logger,
		cache:            c,
	}
}

// GetSettings 获取用户设置
func (s *userSettingsService) GetSettings(ctx context.Context, userID string) (*domain.UserSettings, error) {
	s.logger.Debug("getting user settings",
		zap.String("userID", userID))
	if s.cache != nil {
		var cached domain.UserSettings
		if err := s.cache.Get(ctx, userSettingsCacheKey(userID), &cached); err == nil {
			return &cached, nil
		}
	}

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

	if s.canonicalizeStoredUserSettings(settings) {
		if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
			s.logger.Warn("failed to persist canonical user settings",
				zap.Error(err),
				zap.String("userID", userID))
		}
	}
	if s.cache != nil {
		_ = s.cache.Set(ctx, userSettingsCacheKey(userID), settings, 10*time.Minute)
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
		ShowPublicStories:         true,
		ShowPublicFragments:       true,
		ShowPublicBookmarks:       true,
		AIEnabled:                 true,
		AIDataSharing:             true,
		NotificationSettings:      s.getDefaultNotificationSettings(),
		PreferredGenres:           []string{},
		TeenProtectionEnabled:     false,
	}

	if err := s.settingsRepo.CreateUserSettings(defaultSettings); err != nil {
		s.logger.Error("failed to create default user settings",
			zap.Error(err),
			zap.String("userID", userID))
		return nil, fmt.Errorf("failed to create default settings: %w", err)
	}

	s.logger.Info("default user settings created",
		zap.String("userID", userID))
	if s.cache != nil {
		_ = s.cache.Set(ctx, userSettingsCacheKey(userID), defaultSettings, 10*time.Minute)
	}
	return defaultSettings, nil
}

// UpdateSettings 更新设置（通用）
func (s *userSettingsService) UpdateSettings(ctx context.Context, userID string, updates map[string]interface{}) (*domain.UserSettings, error) {
	preferredGenresChanged := false
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
		language = s.canonicalLanguage(language)
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
		allowFollowFrom = s.canonicalAllowFrom(allowFollowFrom)
		if s.isValidAllowFrom(allowFollowFrom) {
			settings.AllowFollowFrom = allowFollowFrom
		}
	}
	if allowCommentsFrom, ok := updates["allowCommentsFrom"].(string); ok && allowCommentsFrom != "" {
		allowCommentsFrom = s.canonicalAllowFrom(allowCommentsFrom)
		if s.isValidAllowFrom(allowCommentsFrom) {
			settings.AllowCommentsFrom = allowCommentsFrom
		}
	}
	if allowMessagesFrom, ok := updates["allowMessagesFrom"].(string); ok && allowMessagesFrom != "" {
		allowMessagesFrom = s.canonicalAllowFrom(allowMessagesFrom)
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
	if showPublicStories, ok := updates["showPublicStories"].(bool); ok {
		settings.ShowPublicStories = showPublicStories
	}
	if showPublicFragments, ok := updates["showPublicFragments"].(bool); ok {
		settings.ShowPublicFragments = showPublicFragments
	}
	if showPublicBookmarks, ok := updates["showPublicBookmarks"].(bool); ok {
		settings.ShowPublicBookmarks = showPublicBookmarks
	}
	if aiEnabled, ok := updates["aiEnabled"].(bool); ok {
		settings.AIEnabled = aiEnabled
	}
	if aiDataSharing, ok := updates["aiDataSharing"].(bool); ok {
		settings.AIDataSharing = aiDataSharing
	}
	if teen, ok := updates["teenProtectionEnabled"].(bool); ok {
		settings.TeenProtectionEnabled = teen
	}
	if raw, ok := updates["notificationSettings"]; ok {
		if patch, ok := raw.(map[string]interface{}); ok {
			if err := s.applyNotificationPatch(settings, patch); err != nil {
				return nil, fmt.Errorf("failed to merge notification settings: %w", err)
			}
		}
	}
	if rawGenres, ok := updates["preferredGenres"]; ok {
		if genres, ok := parseStringSliceFlexible(rawGenres); ok {
			settings.PreferredGenres = s.sanitizePreferredGenres(genres)
			preferredGenresChanged = true
		}
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
	s.invalidateSettingsCache(ctx, userID)
	if preferredGenresChanged {
		recommendation.InvalidateAllForUser(ctx, s.cache, userID)
	}
	return settings, nil
}

func (s *userSettingsService) UpdatePreferredGenres(ctx context.Context, userID string, genres []string) ([]string, error) {
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return nil, err
	}
	sanitized := s.sanitizePreferredGenres(genres)
	settings.PreferredGenres = sanitized
	settings.UpdatedAt = time.Now().Unix()
	if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
		return nil, fmt.Errorf("failed to update preferred genres: %w", err)
	}
	s.invalidateSettingsCache(ctx, userID)
	recommendation.InvalidateAllForUser(ctx, s.cache, userID)
	return sanitized, nil
}

func (s *userSettingsService) GetPreferredGenresPreferences(ctx context.Context, userID string) ([]string, []string, error) {
	if userID == "" {
		return nil, nil, fmt.Errorf("user id required")
	}
	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	allowed := s.allowedGenreSlugsOrdered()
	preferred := s.sanitizePreferredGenres(settings.PreferredGenres)
	return preferred, allowed, nil
}

// UpdateLanguage 更新语言设置
func (s *userSettingsService) UpdateLanguage(ctx context.Context, userID string, language string) error {
	s.logger.Info("updating language",
		zap.String("userID", userID),
		zap.String("language", language))

	language = s.canonicalLanguage(language)
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
	s.invalidateSettingsCache(ctx, userID)

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
	s.invalidateSettingsCache(ctx, userID)

	return nil
}

// UpdatePrivacy 更新隐私设置
func (s *userSettingsService) UpdatePrivacy(ctx context.Context, userID string, privacy map[string]interface{}) error {
	s.logger.Info("updating privacy settings",
		zap.String("userID", userID))

	settings, err := s.GetSettings(ctx, userID)
	if err != nil {
		return err
	}

	// 更新隐私设置
	if profileVisibility, ok := privacy["profileVisibility"].(string); ok && profileVisibility != "" {
		if s.isValidVisibility(profileVisibility) {
			settings.ProfileVisibility = profileVisibility
		}
	}
	if defaultStoryVisibility, ok := privacy["defaultStoryVisibility"].(string); ok && defaultStoryVisibility != "" {
		if s.isValidVisibility(defaultStoryVisibility) {
			settings.DefaultStoryVisibility = defaultStoryVisibility
		}
	}
	if defaultFragmentVisibility, ok := privacy["defaultFragmentVisibility"].(string); ok && defaultFragmentVisibility != "" {
		if s.isValidVisibility(defaultFragmentVisibility) {
			settings.DefaultFragmentVisibility = defaultFragmentVisibility
		}
	}
	if allowFollowFrom, ok := privacy["allowFollowFrom"].(string); ok && allowFollowFrom != "" {
		allowFollowFrom = s.canonicalAllowFrom(allowFollowFrom)
		if s.isValidAllowFrom(allowFollowFrom) {
			settings.AllowFollowFrom = allowFollowFrom
		}
	}
	if allowCommentsFrom, ok := privacy["allowCommentsFrom"].(string); ok && allowCommentsFrom != "" {
		allowCommentsFrom = s.canonicalAllowFrom(allowCommentsFrom)
		if s.isValidAllowFrom(allowCommentsFrom) {
			settings.AllowCommentsFrom = allowCommentsFrom
		}
	}
	if allowMessagesFrom, ok := privacy["allowMessagesFrom"].(string); ok && allowMessagesFrom != "" {
		allowMessagesFrom = s.canonicalAllowFrom(allowMessagesFrom)
		if s.isValidAllowFrom(allowMessagesFrom) {
			settings.AllowMessagesFrom = allowMessagesFrom
		}
	}
	if showPublicStories, ok := privacy["showPublicStories"].(bool); ok {
		settings.ShowPublicStories = showPublicStories
	}
	if showPublicFragments, ok := privacy["showPublicFragments"].(bool); ok {
		settings.ShowPublicFragments = showPublicFragments
	}
	if showPublicBookmarks, ok := privacy["showPublicBookmarks"].(bool); ok {
		settings.ShowPublicBookmarks = showPublicBookmarks
	}

	settings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(settings); err != nil {
		s.logger.Error("failed to update privacy settings",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to update privacy settings: %w", err)
	}
	s.invalidateSettingsCache(ctx, userID)

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
	s.invalidateSettingsCache(ctx, userID)

	return nil
}

// mergeNotificationMaps deep-merges src into dst (nested maps for push/email/inApp).
func mergeNotificationMaps(dst, src map[string]interface{}) {
	for k, v := range src {
		srcMap, smOk := v.(map[string]interface{})
		if !smOk {
			if sm, ok := v.(map[string]bool); ok {
				srcMap = make(map[string]interface{}, len(sm))
				for kk, vv := range sm {
					srcMap[kk] = vv
				}
				smOk = true
			}
		}
		if smOk {
			if dstVal, ok := dst[k]; ok {
				if dstMap, ok := dstVal.(map[string]interface{}); ok {
					mergeNotificationMaps(dstMap, srcMap)
					continue
				}
			}
			// replace missing or non-map destination with a copy of src branch
			dst[k] = deepCopyMap(srcMap)
			continue
		}
		dst[k] = v
	}
}

func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		if vm, ok := v.(map[string]interface{}); ok {
			out[k] = deepCopyMap(vm)
		} else {
			out[k] = v
		}
	}
	return out
}

func (s *userSettingsService) applyNotificationPatch(dest *domain.UserSettings, patch map[string]interface{}) error {
	var existing map[string]interface{}
	if err := json.Unmarshal([]byte(dest.NotificationSettings), &existing); err != nil || existing == nil {
		existing = s.getDefaultNotificationSettingsMap()
	}
	mergeNotificationMaps(existing, patch)
	jsonBytes, err := json.Marshal(existing)
	if err != nil {
		return fmt.Errorf("failed to marshal notification settings: %w", err)
	}
	dest.NotificationSettings = string(jsonBytes)
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

	if err := s.applyNotificationPatch(userSettings, settings); err != nil {
		s.logger.Error("failed to merge notification settings",
			zap.Error(err),
			zap.String("userID", userID))
		return err
	}

	userSettings.UpdatedAt = time.Now().Unix()

	if err := s.settingsRepo.UpdateUserSettings(userSettings); err != nil {
		s.logger.Error("failed to update notification settings",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to update notification settings: %w", err)
	}
	s.invalidateSettingsCache(ctx, userID)

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
		"push": map[string]interface{}{
			"enabled":            true,
			"newFollower":        true,
			"newLike":            true,
			"newComment":         true,
			"storyUpdate":        true,
			"directMessage":      true,
			"systemAnnouncement": true,
			"marketing":          false,
		},
		"email": map[string]interface{}{
			"enabled":        true,
			"weeklyDigest":   true,
			"securityAlert":  true,
			"marketing":      false,
			"productUpdates": true,
		},
		"inApp": map[string]interface{}{
			"enabled":          true,
			"showPreview":      true,
			"soundEnabled":     true,
			"vibrationEnabled": true,
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
	s.invalidateSettingsCache(ctx, userID)

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

func (s *userSettingsService) allowedGenreSlugSet() map[string]struct{} {
	if s.genreCatalogRepo == nil {
		return domain.AllowedPreferredGenreSet()
	}
	slugs, err := s.genreCatalogRepo.AllSlugs()
	if err != nil {
		s.logger.Warn("genre catalog slugs unavailable, falling back to static allowlist",
			zap.Error(err))
		return domain.AllowedPreferredGenreSet()
	}
	if len(slugs) == 0 {
		return domain.AllowedPreferredGenreSet()
	}
	m := make(map[string]struct{}, len(slugs))
	for _, g := range slugs {
		k := strings.ToLower(strings.TrimSpace(g))
		if k != "" {
			m[k] = struct{}{}
		}
	}
	return m
}

func (s *userSettingsService) allowedGenreSlugsOrdered() []string {
	if s.genreCatalogRepo == nil {
		out := make([]string, len(domain.AllowedPreferredGenreSlugs))
		copy(out, domain.AllowedPreferredGenreSlugs)
		return out
	}
	slugs, err := s.genreCatalogRepo.AllSlugs()
	if err != nil || len(slugs) == 0 {
		out := make([]string, len(domain.AllowedPreferredGenreSlugs))
		copy(out, domain.AllowedPreferredGenreSlugs)
		return out
	}
	return slugs
}

func (s *userSettingsService) sanitizePreferredGenres(genres []string) []string {
	if len(genres) == 0 {
		return []string{}
	}
	allowed := s.allowedGenreSlugSet()
	out := make([]string, 0, len(genres))
	seen := map[string]struct{}{}
	for _, g := range genres {
		genre := strings.ToLower(strings.TrimSpace(g))
		if genre == "" {
			continue
		}
		if _, ok := allowed[genre]; !ok {
			continue
		}
		if _, ok := seen[genre]; ok {
			continue
		}
		seen[genre] = struct{}{}
		out = append(out, genre)
		if len(out) >= 12 {
			break
		}
	}
	return out
}

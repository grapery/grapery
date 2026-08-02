package pay

import (
	"context"
	"time"

	"github.com/grapery/grapery/models/vippay"
	"github.com/sirupsen/logrus"
)

// UsageLimitStrategy 用量限制策略接口
type UsageLimitStrategy interface {
	// 检查用户是否可以执行特定操作
	CanPerformAction(ctx context.Context, userID int64, action ActionType) (*ActionPermission, error)

	// 获取用户当前限制状态
	GetUserLimitStatus(ctx context.Context, userID int64) (*UserLimitStatus, error)

	// 获取功能限制信息
	GetFeatureLimits(ctx context.Context, userID int64) (map[string]*FeatureLimit, error)

	// 应用限制策略
	ApplyLimitStrategy(ctx context.Context, userID int64, action ActionType) (*LimitResult, error)
}

// ActionType 操作类型
type ActionType int

const (
	ActionTypeChat           ActionType = iota + 1 // 聊天对话
	ActionTypeImageGen                             // 图片生成
	ActionTypeVideoGen                             // 视频生成
	ActionTypeStoryGen                             // 故事生成
	ActionTypeRoleGen                              // 角色生成
	ActionTypeContextGen                           // 上下文生成
	ActionTypeFileUpload                           // 文件上传
	ActionTypeExport                               // 导出功能
	ActionTypeAdvancedSearch                       // 高级搜索
	ActionTypeBatchProcess                         // 批量处理
)

// UserLimitStatus 用户限制状态
type UserLimitStatus struct {
	UserID             int64                    `json:"user_id"`
	SubscriptionStatus string                   `json:"subscription_status"` // 订阅状态
	IsVip              bool                     `json:"is_vip"`              // 是否为VIP
	VipLevel           string                   `json:"vip_level"`           // VIP等级
	FeatureLimits      map[string]*FeatureLimit `json:"feature_limits"`      // 功能限制
	OverallUsage       *UsageLimitResult        `json:"overall_usage"`       // 总体用量
	LastResetTime      *time.Time               `json:"last_reset_time"`     // 上次重置时间
	NextResetTime      *time.Time               `json:"next_reset_time"`     // 下次重置时间
	WarningThreshold   float64                  `json:"warning_threshold"`   // 警告阈值
	IsWarning          bool                     `json:"is_warning"`          // 是否达到警告
	IsLimited          bool                     `json:"is_limited"`          // 是否被限制
	LimitedFeatures    []string                 `json:"limited_features"`    // 被限制的功能
	AvailableFeatures  []string                 `json:"available_features"`  // 可用功能
}

// FeatureLimit 功能限制
type FeatureLimit struct {
	FeatureName     string     `json:"feature_name"`     // 功能名称
	UsageType       string     `json:"usage_type"`       // 用量类型
	CurrentUsage    int64      `json:"current_usage"`    // 当前用量
	UsageLimit      int64      `json:"usage_limit"`      // 用量限制
	RemainingUsage  int64      `json:"remaining_usage"`  // 剩余用量
	UsagePercentage float64    `json:"usage_percentage"` // 用量百分比
	IsLimited       bool       `json:"is_limited"`       // 是否被限制
	IsWarning       bool       `json:"is_warning"`       // 是否警告
	ResetTime       *time.Time `json:"reset_time"`       // 重置时间
	Priority        int        `json:"priority"`         // 优先级（1-10，数字越大优先级越高）
}

// LimitResult 限制结果
type LimitResult struct {
	CanPerform         bool                     `json:"can_perform"`         // 是否可以执行
	Reason             string                   `json:"reason"`              // 原因
	ErrorCode          string                   `json:"error_code"`          // 错误代码
	ErrorMessage       string                   `json:"error_message"`       // 错误信息
	UpgradeSuggestion  string                   `json:"upgrade_suggestion"`  // 升级建议
	AlternativeActions []string                 `json:"alternative_actions"` // 替代操作
	WaitTime           *time.Duration           `json:"wait_time"`           // 等待时间
	FeatureLimits      map[string]*FeatureLimit `json:"feature_limits"`      // 相关功能限制
}

// UsageLimitStrategyImpl 用量限制策略实现
type UsageLimitStrategyImpl struct {
	tokenUsageService TokenUsageService
	logger            *logrus.Logger
}

// NewUsageLimitStrategy 创建用量限制策略
func NewUsageLimitStrategy(tokenUsageService TokenUsageService, logger *logrus.Logger) UsageLimitStrategy {
	return &UsageLimitStrategyImpl{
		tokenUsageService: tokenUsageService,
		logger:            logger,
	}
}

// CanPerformAction 检查用户是否可以执行特定操作
func (s *UsageLimitStrategyImpl) CanPerformAction(ctx context.Context, userID int64, action ActionType) (*ActionPermission, error) {
	// 获取用户订阅状态
	subscription, err := vippay.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		// 没有活跃订阅，使用默认限制
		return s.getDefaultActionPermission(action), nil
	}

	// 根据操作类型获取对应的用量类型
	usageType := s.getUsageTypeFromAction(action)

	// 检查用量限制
	permission, err := s.tokenUsageService.CanPerformAction(ctx, userID, usageType)
	if err != nil {
		return nil, err
	}

	// 根据订阅等级调整权限
	return s.adjustPermissionBySubscription(permission, subscription, action), nil
}

// GetUserLimitStatus 获取用户当前限制状态
func (s *UsageLimitStrategyImpl) GetUserLimitStatus(ctx context.Context, userID int64) (*UserLimitStatus, error) {
	// 获取用户订阅状态
	subscription, err := vippay.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		// 没有活跃订阅
		return s.getDefaultUserLimitStatus(userID), nil
	}

	// 获取功能限制
	featureLimits, err := s.GetFeatureLimits(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 计算总体用量
	overallUsage, err := s.calculateOverallUsage(ctx, userID)
	if err != nil {
		return nil, err
	}

	// 确定VIP等级
	vipLevel := s.determineVipLevel(subscription)

	// 检查是否有限制
	isLimited, limitedFeatures := s.checkLimitedFeatures(featureLimits)
	isWarning := s.checkWarningStatus(featureLimits)

	// 计算重置时间
	nextResetTime := s.calculateNextResetTime()

	return &UserLimitStatus{
		UserID:             userID,
		SubscriptionStatus: s.getSubscriptionStatusString(subscription.Status),
		IsVip:              subscription.Status == vippay.UserSubscriptionStatusActive,
		VipLevel:           vipLevel,
		FeatureLimits:      featureLimits,
		OverallUsage:       overallUsage,
		NextResetTime:      &nextResetTime,
		WarningThreshold:   80.0,
		IsWarning:          isWarning,
		IsLimited:          isLimited,
		LimitedFeatures:    limitedFeatures,
		AvailableFeatures:  s.getAvailableFeatures(featureLimits),
	}, nil
}

// GetFeatureLimits 获取功能限制信息
func (s *UsageLimitStrategyImpl) GetFeatureLimits(ctx context.Context, userID int64) (map[string]*FeatureLimit, error) {
	// 获取用户订阅状态
	_, err := vippay.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		// 没有活跃订阅，返回默认限制
		return s.getDefaultFeatureLimits(), nil
	}

	featureLimits := make(map[string]*FeatureLimit)

	// 定义所有功能类型
	actions := []ActionType{
		ActionTypeChat,
		ActionTypeImageGen,
		ActionTypeVideoGen,
		ActionTypeStoryGen,
		ActionTypeRoleGen,
		ActionTypeContextGen,
		ActionTypeFileUpload,
		ActionTypeExport,
		ActionTypeAdvancedSearch,
		ActionTypeBatchProcess,
	}

	for _, action := range actions {
		usageType := s.getUsageTypeFromAction(action)
		limitResult, err := s.tokenUsageService.CheckUserUsageLimit(ctx, userID, usageType)
		if err != nil {
			s.logger.WithError(err).Error("Failed to check usage limit")
			continue
		}

		featureLimit := &FeatureLimit{
			FeatureName:     s.getActionName(action),
			UsageType:       s.getUsageTypeString(usageType),
			CurrentUsage:    limitResult.CurrentUsage,
			UsageLimit:      limitResult.UsageLimit,
			RemainingUsage:  limitResult.RemainingUsage,
			UsagePercentage: limitResult.UsagePercentage,
			IsLimited:       limitResult.IsExceeded,
			IsWarning:       limitResult.IsWarning,
			ResetTime:       limitResult.ResetTime,
			Priority:        s.getActionPriority(action),
		}

		featureLimits[s.getActionName(action)] = featureLimit
	}

	return featureLimits, nil
}

// ApplyLimitStrategy 应用限制策略
func (s *UsageLimitStrategyImpl) ApplyLimitStrategy(ctx context.Context, userID int64, action ActionType) (*LimitResult, error) {
	// 检查是否可以执行操作
	permission, err := s.CanPerformAction(ctx, userID, action)
	if err != nil {
		return nil, err
	}

	if permission.CanPerform {
		return &LimitResult{
			CanPerform: true,
			Reason:     "action_allowed",
		}, nil
	}

	// 应用限制策略
	return s.applyRestrictionStrategy(ctx, userID, action, permission), nil
}

// 辅助方法

// getUsageTypeFromAction 从操作类型获取用量类型
func (s *UsageLimitStrategyImpl) getUsageTypeFromAction(action ActionType) vippay.TokenUsageType {
	switch action {
	case ActionTypeChat:
		return vippay.TokenUsageTypeChat
	case ActionTypeImageGen:
		return vippay.TokenUsageTypeImageGen
	case ActionTypeVideoGen:
		return vippay.TokenUsageTypeVideoGen
	case ActionTypeStoryGen:
		return vippay.TokenUsageTypeStoryGen
	case ActionTypeRoleGen:
		return vippay.TokenUsageTypeRoleGen
	case ActionTypeContextGen:
		return vippay.TokenUsageTypeContextGen
	default:
		return vippay.TokenUsageTypeOther
	}
}

// getActionName 获取操作名称
func (s *UsageLimitStrategyImpl) getActionName(action ActionType) string {
	switch action {
	case ActionTypeChat:
		return "chat"
	case ActionTypeImageGen:
		return "image_generation"
	case ActionTypeVideoGen:
		return "video_generation"
	case ActionTypeStoryGen:
		return "story_generation"
	case ActionTypeRoleGen:
		return "role_generation"
	case ActionTypeContextGen:
		return "context_generation"
	case ActionTypeFileUpload:
		return "file_upload"
	case ActionTypeExport:
		return "export"
	case ActionTypeAdvancedSearch:
		return "advanced_search"
	case ActionTypeBatchProcess:
		return "batch_process"
	default:
		return "unknown"
	}
}

// getActionPriority 获取操作优先级
func (s *UsageLimitStrategyImpl) getActionPriority(action ActionType) int {
	switch action {
	case ActionTypeChat:
		return 10 // 最高优先级
	case ActionTypeImageGen:
		return 7
	case ActionTypeVideoGen:
		return 5
	case ActionTypeStoryGen:
		return 8
	case ActionTypeRoleGen:
		return 6
	case ActionTypeContextGen:
		return 7
	case ActionTypeFileUpload:
		return 4
	case ActionTypeExport:
		return 3
	case ActionTypeAdvancedSearch:
		return 2
	case ActionTypeBatchProcess:
		return 1 // 最低优先级
	default:
		return 1
	}
}

// getUsageTypeString 获取用量类型字符串
func (s *UsageLimitStrategyImpl) getUsageTypeString(usageType vippay.TokenUsageType) string {
	switch usageType {
	case vippay.TokenUsageTypeChat:
		return "chat"
	case vippay.TokenUsageTypeImageGen:
		return "image_gen"
	case vippay.TokenUsageTypeVideoGen:
		return "video_gen"
	case vippay.TokenUsageTypeStoryGen:
		return "story_gen"
	case vippay.TokenUsageTypeRoleGen:
		return "role_gen"
	case vippay.TokenUsageTypeContextGen:
		return "context_gen"
	case vippay.TokenUsageTypeOther:
		return "other"
	default:
		return "unknown"
	}
}

// getDefaultActionPermission 获取默认操作权限
func (s *UsageLimitStrategyImpl) getDefaultActionPermission(action ActionType) *ActionPermission {
	// 免费用户的默认限制
	switch action {
	case ActionTypeChat:
		return &ActionPermission{
			CanPerform:        true,
			Reason:            "free_user_chat_allowed",
			UpgradeSuggestion: "升级到Premium套餐以获得更多聊天额度",
		}
	case ActionTypeImageGen:
		return &ActionPermission{
			CanPerform:        false,
			Reason:            "free_user_image_gen_limited",
			ErrorCode:         "FREE_USER_LIMIT",
			ErrorMessage:      "免费用户无法使用图片生成功能",
			UpgradeSuggestion: "升级到Premium套餐以使用图片生成功能",
		}
	case ActionTypeVideoGen:
		return &ActionPermission{
			CanPerform:        false,
			Reason:            "free_user_video_gen_limited",
			ErrorCode:         "FREE_USER_LIMIT",
			ErrorMessage:      "免费用户无法使用视频生成功能",
			UpgradeSuggestion: "升级到Pro套餐以使用视频生成功能",
		}
	default:
		return &ActionPermission{
			CanPerform:        false,
			Reason:            "free_user_feature_limited",
			ErrorCode:         "FREE_USER_LIMIT",
			ErrorMessage:      "免费用户无法使用此功能",
			UpgradeSuggestion: "升级到Premium或Pro套餐以使用此功能",
		}
	}
}

// adjustPermissionBySubscription 根据订阅调整权限
func (s *UsageLimitStrategyImpl) adjustPermissionBySubscription(permission *ActionPermission, subscription *vippay.UserSubscription, action ActionType) *ActionPermission {
	// 如果已经是VIP用户，即使用量超限也保持会员身份
	if subscription.Status == vippay.UserSubscriptionStatusActive {
		// VIP用户超限时的特殊处理
		if !permission.CanPerform {
			permission.UpgradeSuggestion = s.getVipUpgradeSuggestion(subscription, action)
		}
	}

	return permission
}

// getVipUpgradeSuggestion 获取VIP用户升级建议
func (s *UsageLimitStrategyImpl) getVipUpgradeSuggestion(subscription *vippay.UserSubscription, action ActionType) string {
	// 根据当前订阅等级和操作类型提供升级建议
	vipLevel := s.determineVipLevel(subscription)

	switch vipLevel {
	case "basic":
		if action == ActionTypeVideoGen {
			return "升级到Pro套餐以获得更多视频生成额度"
		}
		return "升级到Premium套餐以获得更多使用额度"
	case "premium":
		return "升级到Pro套餐以获得更多使用额度"
	case "pro":
		return "联系客服了解企业版解决方案"
	default:
		return "升级到更高等级套餐以获得更多使用额度"
	}
}

// determineVipLevel 确定VIP等级
func (s *UsageLimitStrategyImpl) determineVipLevel(subscription *vippay.UserSubscription) string {
	// 根据订阅的配额限制确定VIP等级
	if subscription.QuotaLimit >= 2000000*12 { // Pro年费
		return "pro"
	} else if subscription.QuotaLimit >= 50000000*12 { // Premium年费
		return "premium"
	} else if subscription.QuotaLimit >= 1000000*12 { // Basic年费
		return "basic"
	}
	return "free"
}

// getSubscriptionStatusString 获取订阅状态字符串
func (s *UsageLimitStrategyImpl) getSubscriptionStatusString(status vippay.UserSubscriptionStatus) string {
	switch status {
	case vippay.UserSubscriptionStatusActive:
		return "active"
	case vippay.UserSubscriptionStatusExpired:
		return "expired"
	case vippay.UserSubscriptionStatusCanceled:
		return "canceled"
	case vippay.UserSubscriptionStatusPending:
		return "pending"
	case vippay.UserSubscriptionStatusFailed:
		return "failed"
	case vippay.UserSubscriptionStatusTrial:
		return "trial"
	case vippay.UserSubscriptionStatusPaused:
		return "paused"
	default:
		return "unknown"
	}
}

// calculateOverallUsage 计算总体用量
func (s *UsageLimitStrategyImpl) calculateOverallUsage(ctx context.Context, userID int64) (*UsageLimitResult, error) {
	// 获取所有类型的用量统计
	stats, err := s.tokenUsageService.GetUserUsageStats(ctx, userID, vippay.TokenUsagePeriodMonthly)
	if err != nil {
		return nil, err
	}

	totalTokens := stats["total_tokens"].(int64)

	// 这里需要根据用户订阅计算总体限制
	// 简化实现，实际应该根据订阅类型计算
	overallLimit := int64(1000000) // 默认限制

	return &UsageLimitResult{
		CurrentUsage:     totalTokens,
		UsageLimit:       overallLimit,
		RemainingUsage:   overallLimit - totalTokens,
		UsagePercentage:  float64(totalTokens) / float64(overallLimit) * 100,
		Period:           "monthly",
		WarningThreshold: 80.0,
		IsWarning:        float64(totalTokens)/float64(overallLimit)*100 >= 80.0,
		IsExceeded:       totalTokens >= overallLimit,
		IsAllowed:        totalTokens < overallLimit,
	}, nil
}

// checkLimitedFeatures 检查被限制的功能
func (s *UsageLimitStrategyImpl) checkLimitedFeatures(featureLimits map[string]*FeatureLimit) (bool, []string) {
	var limitedFeatures []string
	isLimited := false

	for featureName, limit := range featureLimits {
		if limit.IsLimited {
			limitedFeatures = append(limitedFeatures, featureName)
			isLimited = true
		}
	}

	return isLimited, limitedFeatures
}

// checkWarningStatus 检查警告状态
func (s *UsageLimitStrategyImpl) checkWarningStatus(featureLimits map[string]*FeatureLimit) bool {
	for _, limit := range featureLimits {
		if limit.IsWarning {
			return true
		}
	}
	return false
}

// getAvailableFeatures 获取可用功能
func (s *UsageLimitStrategyImpl) getAvailableFeatures(featureLimits map[string]*FeatureLimit) []string {
	var availableFeatures []string

	for featureName, limit := range featureLimits {
		if !limit.IsLimited {
			availableFeatures = append(availableFeatures, featureName)
		}
	}

	return availableFeatures
}

// calculateNextResetTime 计算下次重置时间
func (s *UsageLimitStrategyImpl) calculateNextResetTime() time.Time {
	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)
	return time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location())
}

// getDefaultUserLimitStatus 获取默认用户限制状态
func (s *UsageLimitStrategyImpl) getDefaultUserLimitStatus(userID int64) *UserLimitStatus {
	nextResetTime := s.calculateNextResetTime()

	return &UserLimitStatus{
		UserID:             userID,
		SubscriptionStatus: "free",
		IsVip:              false,
		VipLevel:           "free",
		FeatureLimits:      s.getDefaultFeatureLimits(),
		OverallUsage: &UsageLimitResult{
			CurrentUsage:     0,
			UsageLimit:       1000,
			RemainingUsage:   1000,
			UsagePercentage:  0,
			Period:           "monthly",
			WarningThreshold: 80.0,
			IsWarning:        false,
			IsExceeded:       false,
			IsAllowed:        true,
		},
		NextResetTime:     &nextResetTime,
		WarningThreshold:  80.0,
		IsWarning:         false,
		IsLimited:         false,
		LimitedFeatures:   []string{},
		AvailableFeatures: []string{"chat"},
	}
}

// getDefaultFeatureLimits 获取默认功能限制
func (s *UsageLimitStrategyImpl) getDefaultFeatureLimits() map[string]*FeatureLimit {
	nextResetTime := s.calculateNextResetTime()

	return map[string]*FeatureLimit{
		"chat": {
			FeatureName:     "chat",
			UsageType:       "chat",
			CurrentUsage:    0,
			UsageLimit:      1000,
			RemainingUsage:  1000,
			UsagePercentage: 0,
			IsLimited:       false,
			IsWarning:       false,
			ResetTime:       &nextResetTime,
			Priority:        10,
		},
		"image_generation": {
			FeatureName:     "image_generation",
			UsageType:       "image_gen",
			CurrentUsage:    0,
			UsageLimit:      0,
			RemainingUsage:  0,
			UsagePercentage: 0,
			IsLimited:       true,
			IsWarning:       false,
			ResetTime:       &nextResetTime,
			Priority:        7,
		},
	}
}

// applyRestrictionStrategy 应用限制策略
func (s *UsageLimitStrategyImpl) applyRestrictionStrategy(ctx context.Context, userID int64, action ActionType, permission *ActionPermission) *LimitResult {
	// 根据操作类型和用户状态应用不同的限制策略
	switch action {
	case ActionTypeChat:
		// 聊天功能：即使超限也允许基本对话，但限制高级功能
		return &LimitResult{
			CanPerform:         false,
			Reason:             "chat_limit_exceeded",
			ErrorCode:          "CHAT_LIMIT_EXCEEDED",
			ErrorMessage:       "聊天额度已用完，请等待下月重置或升级套餐",
			UpgradeSuggestion:  permission.UpgradeSuggestion,
			AlternativeActions: []string{"等待下月重置", "升级套餐", "使用基础对话功能"},
		}
	case ActionTypeImageGen:
		// 图片生成：完全限制
		return &LimitResult{
			CanPerform:         false,
			Reason:             "image_gen_limit_exceeded",
			ErrorCode:          "IMAGE_GEN_LIMIT_EXCEEDED",
			ErrorMessage:       "图片生成额度已用完",
			UpgradeSuggestion:  permission.UpgradeSuggestion,
			AlternativeActions: []string{"等待下月重置", "升级套餐"},
		}
	case ActionTypeVideoGen:
		// 视频生成：完全限制
		return &LimitResult{
			CanPerform:         false,
			Reason:             "video_gen_limit_exceeded",
			ErrorCode:          "VIDEO_GEN_LIMIT_EXCEEDED",
			ErrorMessage:       "视频生成额度已用完",
			UpgradeSuggestion:  permission.UpgradeSuggestion,
			AlternativeActions: []string{"等待下月重置", "升级到Pro套餐"},
		}
	default:
		// 其他功能：完全限制
		return &LimitResult{
			CanPerform:         false,
			Reason:             "feature_limit_exceeded",
			ErrorCode:          "FEATURE_LIMIT_EXCEEDED",
			ErrorMessage:       "功能使用额度已用完",
			UpgradeSuggestion:  permission.UpgradeSuggestion,
			AlternativeActions: []string{"等待下月重置", "升级套餐"},
		}
	}
}

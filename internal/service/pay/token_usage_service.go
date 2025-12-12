package pay

import (
	"context"
	"fmt"
	"time"

	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/sirupsen/logrus"
)

// TokenUsageService Token用量管理服务接口
type TokenUsageService interface {
	// 记录Token使用
	RecordTokenUsage(ctx context.Context, userID int64, usageType paymodels.TokenUsageType, inputTokens, outputTokens int64, modelName, featureName string) error

	// 记录失败请求
	RecordFailedRequest(ctx context.Context, userID int64, usageType paymodels.TokenUsageType) error

	// 检查用户用量限制
	CheckUserUsageLimit(ctx context.Context, userID int64, usageType paymodels.TokenUsageType) (*UsageLimitResult, error)

	// 获取用户用量统计
	GetUserUsageStats(ctx context.Context, userID int64, period paymodels.TokenUsagePeriod) (map[string]interface{}, error)

	// 获取用户各类型用量
	GetUserUsageByType(ctx context.Context, userID int64, period paymodels.TokenUsagePeriod) (map[paymodels.TokenUsageType]map[string]interface{}, error)

	// 获取用户当前周期用量
	GetUserCurrentUsage(ctx context.Context, userID int64, usageType paymodels.TokenUsageType, period paymodels.TokenUsagePeriod) (*paymodels.TokenUsage, error)

	// 检查是否可以执行操作
	CanPerformAction(ctx context.Context, userID int64, usageType paymodels.TokenUsageType) (*ActionPermission, error)
}

// UsageLimitResult 用量限制检查结果
type UsageLimitResult struct {
	IsAllowed        bool       `json:"is_allowed"`        // 是否允许使用
	CurrentUsage     int64      `json:"current_usage"`     // 当前用量
	UsageLimit       int64      `json:"usage_limit"`       // 用量限制
	RemainingUsage   int64      `json:"remaining_usage"`   // 剩余用量
	UsagePercentage  float64    `json:"usage_percentage"`  // 用量百分比
	Period           string     `json:"period"`            // 统计周期
	ResetTime        *time.Time `json:"reset_time"`        // 重置时间
	WarningThreshold float64    `json:"warning_threshold"` // 警告阈值
	IsWarning        bool       `json:"is_warning"`        // 是否达到警告阈值
	IsExceeded       bool       `json:"is_exceeded"`       // 是否超出限制
}

// ActionPermission 操作权限
type ActionPermission struct {
	CanPerform        bool   `json:"can_perform"`        // 是否可以执行
	Reason            string `json:"reason"`             // 原因
	ErrorCode         string `json:"error_code"`         // 错误代码
	ErrorMessage      string `json:"error_message"`      // 错误信息
	UpgradeSuggestion string `json:"upgrade_suggestion"` // 升级建议
}

// TokenUsageServiceImpl Token用量管理服务实现
type TokenUsageServiceImpl struct {
	logger *logrus.Logger
}

// NewTokenUsageService 创建Token用量管理服务
func NewTokenUsageService(logger *logrus.Logger) TokenUsageService {
	return &TokenUsageServiceImpl{
		logger: logger,
	}
}

// RecordTokenUsage 记录Token使用
func (s *TokenUsageServiceImpl) RecordTokenUsage(ctx context.Context, userID int64, usageType paymodels.TokenUsageType, inputTokens, outputTokens int64, modelName, featureName string) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"usage_type":    usageType,
		"input_tokens":  inputTokens,
		"output_tokens": outputTokens,
		"model_name":    modelName,
		"feature_name":  featureName,
	}).Info("Recording token usage")

	// 记录日用量
	err := paymodels.IncrementTokenUsage(ctx, userID, usageType, paymodels.TokenUsagePeriodDaily, inputTokens, outputTokens, modelName, featureName)
	if err != nil {
		s.logger.WithError(err).Error("Failed to record daily token usage")
		return err
	}

	// 记录月用量
	err = paymodels.IncrementTokenUsage(ctx, userID, usageType, paymodels.TokenUsagePeriodMonthly, inputTokens, outputTokens, modelName, featureName)
	if err != nil {
		s.logger.WithError(err).Error("Failed to record monthly token usage")
		return err
	}

	// 记录年用量
	err = paymodels.IncrementTokenUsage(ctx, userID, usageType, paymodels.TokenUsagePeriodYearly, inputTokens, outputTokens, modelName, featureName)
	if err != nil {
		s.logger.WithError(err).Error("Failed to record yearly token usage")
		return err
	}

	return nil
}

// RecordFailedRequest 记录失败请求
func (s *TokenUsageServiceImpl) RecordFailedRequest(ctx context.Context, userID int64, usageType paymodels.TokenUsageType) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"usage_type": usageType,
	}).Info("Recording failed request")

	// 记录日失败请求
	err := paymodels.IncrementFailedRequest(ctx, userID, usageType, paymodels.TokenUsagePeriodDaily)
	if err != nil {
		s.logger.WithError(err).Error("Failed to record daily failed request")
		return err
	}

	// 记录月失败请求
	err = paymodels.IncrementFailedRequest(ctx, userID, usageType, paymodels.TokenUsagePeriodMonthly)
	if err != nil {
		s.logger.WithError(err).Error("Failed to record monthly failed request")
		return err
	}

	// 记录年失败请求
	err = paymodels.IncrementFailedRequest(ctx, userID, usageType, paymodels.TokenUsagePeriodYearly)
	if err != nil {
		s.logger.WithError(err).Error("Failed to record yearly failed request")
		return err
	}

	return nil
}

// CheckUserUsageLimit 检查用户用量限制
func (s *TokenUsageServiceImpl) CheckUserUsageLimit(ctx context.Context, userID int64, usageType paymodels.TokenUsageType) (*UsageLimitResult, error) {
	// 获取用户当前活跃订阅
	subscription, err := paymodels.GetUserActiveSubscriptionByUserID(ctx, userID)
	if err != nil {
		// 如果没有活跃订阅，使用默认限制
		return s.getDefaultUsageLimit(usageType), nil
	}

	// 获取用户当前月用量
	currentUsage, err := s.GetUserCurrentUsage(ctx, userID, usageType, paymodels.TokenUsagePeriodMonthly)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get current usage")
		return nil, err
	}

	// 根据订阅类型和用量类型获取限制
	usageLimit := s.getUsageLimitBySubscriptionAndType(subscription, usageType)

	result := &UsageLimitResult{
		CurrentUsage:     currentUsage.TotalTokens,
		UsageLimit:       usageLimit,
		RemainingUsage:   usageLimit - currentUsage.TotalTokens,
		Period:           "monthly",
		WarningThreshold: 80.0, // 80%警告阈值
	}

	// 计算用量百分比
	if usageLimit > 0 {
		result.UsagePercentage = float64(currentUsage.TotalTokens) / float64(usageLimit) * 100
	}

	// 检查是否超出限制
	result.IsExceeded = currentUsage.TotalTokens >= usageLimit
	result.IsAllowed = !result.IsExceeded

	// 检查是否达到警告阈值
	result.IsWarning = result.UsagePercentage >= result.WarningThreshold

	// 计算重置时间（下个月1号）
	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)
	resetTime := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location())
	result.ResetTime = &resetTime

	return result, nil
}

// GetUserUsageStats 获取用户用量统计
func (s *TokenUsageServiceImpl) GetUserUsageStats(ctx context.Context, userID int64, period paymodels.TokenUsagePeriod) (map[string]interface{}, error) {
	return paymodels.GetUserTokenUsageStats(ctx, userID, period)
}

// GetUserUsageByType 获取用户各类型用量
func (s *TokenUsageServiceImpl) GetUserUsageByType(ctx context.Context, userID int64, period paymodels.TokenUsagePeriod) (map[paymodels.TokenUsageType]map[string]interface{}, error) {
	return paymodels.GetUserTokenUsageByType(ctx, userID, period)
}

// GetUserCurrentUsage 获取用户当前周期用量
func (s *TokenUsageServiceImpl) GetUserCurrentUsage(ctx context.Context, userID int64, usageType paymodels.TokenUsageType, period paymodels.TokenUsagePeriod) (*paymodels.TokenUsage, error) {
	usage, err := paymodels.GetUserCurrentPeriodUsage(ctx, userID, usageType, period)
	if err != nil {
		// 如果没有记录，返回空用量
		return &paymodels.TokenUsage{
			UserID:       userID,
			UsageType:    usageType,
			Period:       period,
			InputTokens:  0,
			OutputTokens: 0,
			TotalTokens:  0,
			RequestCount: 0,
			SuccessCount: 0,
			FailedCount:  0,
		}, nil
	}
	return usage, nil
}

// CanPerformAction 检查是否可以执行操作
func (s *TokenUsageServiceImpl) CanPerformAction(ctx context.Context, userID int64, usageType paymodels.TokenUsageType) (*ActionPermission, error) {
	// 检查用量限制
	limitResult, err := s.CheckUserUsageLimit(ctx, userID, usageType)
	if err != nil {
		return &ActionPermission{
			CanPerform:   false,
			Reason:       "usage_check_failed",
			ErrorCode:    "USAGE_CHECK_ERROR",
			ErrorMessage: "Failed to check usage limit",
		}, err
	}

	if !limitResult.IsAllowed {
		return &ActionPermission{
			CanPerform:        false,
			Reason:            "usage_limit_exceeded",
			ErrorCode:         "USAGE_LIMIT_EXCEEDED",
			ErrorMessage:      fmt.Sprintf("Usage limit exceeded. Current: %d, Limit: %d", limitResult.CurrentUsage, limitResult.UsageLimit),
			UpgradeSuggestion: s.getUpgradeSuggestion(usageType),
		}, nil
	}

	return &ActionPermission{
		CanPerform: true,
		Reason:     "usage_within_limit",
	}, nil
}

// getUsageLimitBySubscriptionAndType 根据订阅和用量类型获取限制
func (s *TokenUsageServiceImpl) getUsageLimitBySubscriptionAndType(subscription *paymodels.UserSubscription, usageType paymodels.TokenUsageType) int64 {
	// 基础限制来自订阅的QuotaLimit
	baseLimit := int64(subscription.QuotaLimit)

	// 根据用量类型调整限制
	switch usageType {
	case paymodels.TokenUsageTypeChat:
		// 聊天对话使用基础限制
		return baseLimit
	case paymodels.TokenUsageTypeImageGen:
		// 图片生成使用基础限制的20%
		return baseLimit / 5
	case paymodels.TokenUsageTypeVideoGen:
		// 视频生成使用基础限制的10%
		return baseLimit / 10
	case paymodels.TokenUsageTypeStoryGen:
		// 故事生成使用基础限制的30%
		return int64(float64(baseLimit) * 0.3)
	case paymodels.TokenUsageTypeRoleGen:
		// 角色生成使用基础限制的15%
		return int64(float64(baseLimit) * 0.15)
	case paymodels.TokenUsageTypeContextGen:
		// 上下文生成使用基础限制的25%
		return int64(float64(baseLimit) * 0.25)
	default:
		// 其他类型使用基础限制
		return baseLimit
	}
}

// getDefaultUsageLimit 获取默认用量限制（无订阅用户）
func (s *TokenUsageServiceImpl) getDefaultUsageLimit(usageType paymodels.TokenUsageType) *UsageLimitResult {
	var defaultLimit int64

	switch usageType {
	case paymodels.TokenUsageTypeChat:
		defaultLimit = 1000 // 免费用户每月1000 tokens
	case paymodels.TokenUsageTypeImageGen:
		defaultLimit = 50 // 免费用户每月50次图片生成
	case paymodels.TokenUsageTypeVideoGen:
		defaultLimit = 10 // 免费用户每月10次视频生成
	case paymodels.TokenUsageTypeStoryGen:
		defaultLimit = 100 // 免费用户每月100次故事生成
	case paymodels.TokenUsageTypeRoleGen:
		defaultLimit = 20 // 免费用户每月20次角色生成
	case paymodels.TokenUsageTypeContextGen:
		defaultLimit = 50 // 免费用户每月50次上下文生成
	default:
		defaultLimit = 100 // 其他类型默认100次
	}

	now := time.Now()
	nextMonth := now.AddDate(0, 1, 0)
	resetTime := time.Date(nextMonth.Year(), nextMonth.Month(), 1, 0, 0, 0, 0, now.Location())

	return &UsageLimitResult{
		CurrentUsage:     0,
		UsageLimit:       defaultLimit,
		RemainingUsage:   defaultLimit,
		UsagePercentage:  0,
		Period:           "monthly",
		ResetTime:        &resetTime,
		WarningThreshold: 80.0,
		IsWarning:        false,
		IsExceeded:       false,
		IsAllowed:        true,
	}
}

// getUpgradeSuggestion 获取升级建议
func (s *TokenUsageServiceImpl) getUpgradeSuggestion(usageType paymodels.TokenUsageType) string {
	switch usageType {
	case paymodels.TokenUsageTypeChat:
		return "升级到Premium或Pro套餐以获得更多聊天额度"
	case paymodels.TokenUsageTypeImageGen:
		return "升级到Premium或Pro套餐以获得更多图片生成额度"
	case paymodels.TokenUsageTypeVideoGen:
		return "升级到Pro套餐以获得更多视频生成额度"
	case paymodels.TokenUsageTypeStoryGen:
		return "升级到Premium或Pro套餐以获得更多故事生成额度"
	case paymodels.TokenUsageTypeRoleGen:
		return "升级到Premium或Pro套餐以获得更多角色生成额度"
	case paymodels.TokenUsageTypeContextGen:
		return "升级到Premium或Pro套餐以获得更多上下文生成额度"
	default:
		return "升级到Premium或Pro套餐以获得更多使用额度"
	}
}

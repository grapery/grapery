package pay

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// TokenUsageType Token使用类型
type TokenUsageType int

const (
	TokenUsageTypeChat       TokenUsageType = iota + 1 // 聊天对话
	TokenUsageTypeImageGen                             // 图片生成
	TokenUsageTypeVideoGen                             // 视频生成
	TokenUsageTypeStoryGen                             // 故事生成
	TokenUsageTypeRoleGen                              // 角色生成
	TokenUsageTypeContextGen                           // 上下文生成
	TokenUsageTypeOther                                // 其他用途
)

// TokenUsagePeriod Token用量统计周期
type TokenUsagePeriod int

const (
	TokenUsagePeriodDaily   TokenUsagePeriod = iota + 1 // 日统计
	TokenUsagePeriodWeekly                              // 周统计
	TokenUsagePeriodMonthly                             // 月统计
	TokenUsagePeriodYearly                              // 年统计
)

// TokenUsage 用户Token用量记录模型
type TokenUsage struct {
	IDBase
	UserID         int64            `gorm:"column:user_id;not null;index" json:"user_id"`                       // 用户ID
	SubscriptionID *uint            `gorm:"column:subscription_id;index" json:"subscription_id"`                // 关联的订阅ID（可选）
	UsageType      TokenUsageType   `gorm:"column:usage_type;not null;index" json:"usage_type"`                 // 使用类型
	Period         TokenUsagePeriod `gorm:"column:period;not null;index" json:"period"`                         // 统计周期
	PeriodStart    time.Time        `gorm:"column:period_start;not null;index" json:"period_start"`             // 周期开始时间
	PeriodEnd      time.Time        `gorm:"column:period_end;not null;index" json:"period_end"`                 // 周期结束时间
	InputTokens    int64            `gorm:"column:input_tokens;default:0" json:"input_tokens"`                  // 输入Token数量
	OutputTokens   int64            `gorm:"column:output_tokens;default:0" json:"output_tokens"`                // 输出Token数量
	TotalTokens    int64            `gorm:"column:total_tokens;default:0" json:"total_tokens"`                  // 总Token数量
	RequestCount   int              `gorm:"column:request_count;default:0" json:"request_count"`                // 请求次数
	SuccessCount   int              `gorm:"column:success_count;default:0" json:"success_count"`                // 成功次数
	FailedCount    int              `gorm:"column:failed_count;default:0" json:"failed_count"`                  // 失败次数
	CostAmount     float64          `gorm:"column:cost_amount;type:decimal(10,4);default:0" json:"cost_amount"` // 成本金额
	Currency       string           `gorm:"column:currency;size:10;default:'USD'" json:"currency"`              // 货币类型
	ModelName      string           `gorm:"column:model_name;size:100" json:"model_name"`                       // 使用的模型名称
	FeatureName    string           `gorm:"column:feature_name;size:100" json:"feature_name"`                   // 功能名称
	Metadata       string           `gorm:"column:metadata;type:text" json:"metadata"`                          // 元数据（JSON）
	LastUsedAt     *time.Time       `gorm:"column:last_used_at;index" json:"last_used_at"`                      // 最后使用时间
	IsActive       bool             `gorm:"column:is_active;default:true" json:"is_active"`                     // 是否活跃
}

func (tu TokenUsage) TableName() string {
	return "token_usage"
}

// GetMetadata 获取元数据
func (tu *TokenUsage) GetMetadata() (map[string]interface{}, error) {
	if tu.Metadata == "" {
		return map[string]interface{}{}, nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(tu.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
func (tu *TokenUsage) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	tu.Metadata = string(data)
	return nil
}

// GetSuccessRate 获取成功率
func (tu *TokenUsage) GetSuccessRate() float64 {
	if tu.RequestCount == 0 {
		return 0
	}
	return float64(tu.SuccessCount) / float64(tu.RequestCount) * 100
}

// GetAverageTokensPerRequest 获取平均每次请求的Token数
func (tu *TokenUsage) GetAverageTokensPerRequest() float64 {
	if tu.RequestCount == 0 {
		return 0
	}
	return float64(tu.TotalTokens) / float64(tu.RequestCount)
}

// IsWithinPeriod 检查是否在周期内
func (tu *TokenUsage) IsWithinPeriod() bool {
	now := time.Now()
	return now.After(tu.PeriodStart) && now.Before(tu.PeriodEnd)
}

// CreateTokenUsage 创建Token用量记录
func CreateTokenUsage(ctx context.Context, usage *TokenUsage) error {
	return DataBase().WithContext(ctx).Create(usage).Error
}

// GetTokenUsage 获取Token用量记录
func GetTokenUsage(ctx context.Context, id uint) (*TokenUsage, error) {
	var usage TokenUsage
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&usage).Error
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

// GetUserTokenUsageByPeriod 获取用户指定周期的Token用量
func GetUserTokenUsageByPeriod(ctx context.Context, userID int64, usageType TokenUsageType, period TokenUsagePeriod, periodStart, periodEnd time.Time) (*TokenUsage, error) {
	var usage TokenUsage
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND usage_type = ? AND period = ? AND period_start = ? AND period_end = ?",
			userID, usageType, period, periodStart, periodEnd).
		First(&usage).Error
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

// GetUserCurrentPeriodUsage 获取用户当前周期的Token用量
func GetUserCurrentPeriodUsage(ctx context.Context, userID int64, usageType TokenUsageType, period TokenUsagePeriod) (*TokenUsage, error) {
	now := time.Now()
	var periodStart, periodEnd time.Time

	switch period {
	case TokenUsagePeriodDaily:
		periodStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 1)
	case TokenUsagePeriodWeekly:
		// 获取本周一
		weekday := int(now.Weekday())
		if weekday == 0 { // 周日
			weekday = 7
		}
		periodStart = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 7)
	case TokenUsagePeriodMonthly:
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 1, 0)
	case TokenUsagePeriodYearly:
		periodStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(1, 0, 0)
	}

	return GetUserTokenUsageByPeriod(ctx, userID, usageType, period, periodStart, periodEnd)
}

// GetUserTokenUsageHistory 获取用户Token用量历史
func GetUserTokenUsageHistory(ctx context.Context, userID int64, usageType TokenUsageType, period TokenUsagePeriod, offset, limit int) ([]*TokenUsage, error) {
	var usages []*TokenUsage
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND usage_type = ? AND period = ?", userID, usageType, period).
		Order("period_start DESC").
		Offset(offset).
		Limit(limit).
		Find(&usages).Error
	if err != nil {
		return nil, err
	}
	return usages, nil
}

// UpdateTokenUsage 更新Token用量
func UpdateTokenUsage(ctx context.Context, usage *TokenUsage) error {
	return DataBase().WithContext(ctx).Save(usage).Error
}

// IncrementTokenUsage 增加Token用量
func IncrementTokenUsage(ctx context.Context, userID int64, usageType TokenUsageType, period TokenUsagePeriod, inputTokens, outputTokens int64, modelName, featureName string) error {
	now := time.Now()
	var periodStart, periodEnd time.Time

	// 计算周期时间范围
	switch period {
	case TokenUsagePeriodDaily:
		periodStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 1)
	case TokenUsagePeriodWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		periodStart = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 7)
	case TokenUsagePeriodMonthly:
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 1, 0)
	case TokenUsagePeriodYearly:
		periodStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(1, 0, 0)
	}

	totalTokens := inputTokens + outputTokens

	// 使用 ON DUPLICATE KEY UPDATE 或 UPSERT 逻辑
	var usage TokenUsage
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND usage_type = ? AND period = ? AND period_start = ? AND period_end = ?",
			userID, usageType, period, periodStart, periodEnd).
		First(&usage).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		usage = TokenUsage{
			UserID:       userID,
			UsageType:    usageType,
			Period:       period,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalTokens:  totalTokens,
			RequestCount: 1,
			SuccessCount: 1,
			ModelName:    modelName,
			FeatureName:  featureName,
			LastUsedAt:   &now,
			IsActive:     true,
		}
		return DataBase().WithContext(ctx).Create(&usage).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	return DataBase().WithContext(ctx).
		Model(&usage).
		Updates(map[string]interface{}{
			"input_tokens":  gorm.Expr("input_tokens + ?", inputTokens),
			"output_tokens": gorm.Expr("output_tokens + ?", outputTokens),
			"total_tokens":  gorm.Expr("total_tokens + ?", totalTokens),
			"request_count": gorm.Expr("request_count + 1"),
			"success_count": gorm.Expr("success_count + 1"),
			"last_used_at":  &now,
		}).Error
}

// IncrementFailedRequest 增加失败请求计数
func IncrementFailedRequest(ctx context.Context, userID int64, usageType TokenUsageType, period TokenUsagePeriod) error {
	now := time.Now()
	var periodStart, periodEnd time.Time

	switch period {
	case TokenUsagePeriodDaily:
		periodStart = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 1)
	case TokenUsagePeriodWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		periodStart = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 0, 7)
	case TokenUsagePeriodMonthly:
		periodStart = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(0, 1, 0)
	case TokenUsagePeriodYearly:
		periodStart = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		periodEnd = periodStart.AddDate(1, 0, 0)
	}

	var usage TokenUsage
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND usage_type = ? AND period = ? AND period_start = ? AND period_end = ?",
			userID, usageType, period, periodStart, periodEnd).
		First(&usage).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		usage = TokenUsage{
			UserID:       userID,
			UsageType:    usageType,
			Period:       period,
			PeriodStart:  periodStart,
			PeriodEnd:    periodEnd,
			RequestCount: 1,
			FailedCount:  1,
			LastUsedAt:   &now,
			IsActive:     true,
		}
		return DataBase().WithContext(ctx).Create(&usage).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	return DataBase().WithContext(ctx).
		Model(&usage).
		Updates(map[string]interface{}{
			"request_count": gorm.Expr("request_count + 1"),
			"failed_count":  gorm.Expr("failed_count + 1"),
			"last_used_at":  &now,
		}).Error
}

// GetUserTokenUsageStats 获取用户Token用量统计
func GetUserTokenUsageStats(ctx context.Context, userID int64, period TokenUsagePeriod) (map[string]interface{}, error) {
	var stats struct {
		TotalInputTokens  int64   `json:"total_input_tokens"`
		TotalOutputTokens int64   `json:"total_output_tokens"`
		TotalTokens       int64   `json:"total_tokens"`
		TotalRequests     int     `json:"total_requests"`
		TotalSuccess      int     `json:"total_success"`
		TotalFailed       int     `json:"total_failed"`
		TotalCost         float64 `json:"total_cost"`
		SuccessRate       float64 `json:"success_rate"`
	}

	err := DataBase().WithContext(ctx).
		Model(&TokenUsage{}).
		Where("user_id = ? AND period = ? AND is_active = ?", userID, period, true).
		Select(`
			SUM(input_tokens) as total_input_tokens,
			SUM(output_tokens) as total_output_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(request_count) as total_requests,
			SUM(success_count) as total_success,
			SUM(failed_count) as total_failed,
			SUM(cost_amount) as total_cost
		`).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	// 计算成功率
	if stats.TotalRequests > 0 {
		stats.SuccessRate = float64(stats.TotalSuccess) / float64(stats.TotalRequests) * 100
	}

	return map[string]interface{}{
		"total_input_tokens":  stats.TotalInputTokens,
		"total_output_tokens": stats.TotalOutputTokens,
		"total_tokens":        stats.TotalTokens,
		"total_requests":      stats.TotalRequests,
		"total_success":       stats.TotalSuccess,
		"total_failed":        stats.TotalFailed,
		"total_cost":          stats.TotalCost,
		"success_rate":        stats.SuccessRate,
	}, nil
}

// GetUserTokenUsageByType 按类型获取用户Token用量
func GetUserTokenUsageByType(ctx context.Context, userID int64, period TokenUsagePeriod) (map[TokenUsageType]map[string]interface{}, error) {
	var usages []struct {
		UsageType         TokenUsageType `json:"usage_type"`
		TotalInputTokens  int64          `json:"total_input_tokens"`
		TotalOutputTokens int64          `json:"total_output_tokens"`
		TotalTokens       int64          `json:"total_tokens"`
		TotalRequests     int            `json:"total_requests"`
		TotalSuccess      int            `json:"total_success"`
		TotalFailed       int            `json:"total_failed"`
		TotalCost         float64        `json:"total_cost"`
	}

	err := DataBase().WithContext(ctx).
		Model(&TokenUsage{}).
		Where("user_id = ? AND period = ? AND is_active = ?", userID, period, true).
		Select(`
			usage_type,
			SUM(input_tokens) as total_input_tokens,
			SUM(output_tokens) as total_output_tokens,
			SUM(total_tokens) as total_tokens,
			SUM(request_count) as total_requests,
			SUM(success_count) as total_success,
			SUM(failed_count) as total_failed,
			SUM(cost_amount) as total_cost
		`).
		Group("usage_type").
		Scan(&usages).Error

	if err != nil {
		return nil, err
	}

	result := make(map[TokenUsageType]map[string]interface{})
	for _, usage := range usages {
		successRate := 0.0
		if usage.TotalRequests > 0 {
			successRate = float64(usage.TotalSuccess) / float64(usage.TotalRequests) * 100
		}

		result[usage.UsageType] = map[string]interface{}{
			"total_input_tokens":  usage.TotalInputTokens,
			"total_output_tokens": usage.TotalOutputTokens,
			"total_tokens":        usage.TotalTokens,
			"total_requests":      usage.TotalRequests,
			"total_success":       usage.TotalSuccess,
			"total_failed":        usage.TotalFailed,
			"total_cost":          usage.TotalCost,
			"success_rate":        successRate,
		}
	}

	return result, nil
}

// CleanupExpiredTokenUsage 清理过期的Token用量记录
func CleanupExpiredTokenUsage(ctx context.Context, beforeDate time.Time) error {
	return DataBase().WithContext(ctx).
		Where("period_end < ? AND is_active = ?", beforeDate, false).
		Delete(&TokenUsage{}).Error
}

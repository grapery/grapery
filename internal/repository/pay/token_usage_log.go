package pay

import (
	"context"
	"encoding/json"
	"time"
)

// EntityType 业务实体类型
type EntityType string

const (
	EntityTypeStory      EntityType = "story"
	EntityTypeStoryboard EntityType = "storyboard"
	EntityTypeCharacter  EntityType = "character"
	EntityTypePoster     EntityType = "poster"
	EntityTypeStoryScene EntityType = "story_scene"
	EntityTypeAITask     EntityType = "ai_task"
)

// OperationType 操作类型
type OperationType string

const (
	OperationTypeCreate        OperationType = "create"
	OperationTypeUpdate        OperationType = "update"
	OperationTypeGenerateImage OperationType = "generate_image"
	OperationTypeGenerateVideo OperationType = "generate_video"
	OperationTypeGenerateText  OperationType = "generate_text"
	OperationTypeEnhancePrompt OperationType = "enhance_prompt"
)

// TokenUsageLog Token用量日志模型（详细记录）
type TokenUsageLog struct {
	IDBase
	UserID        int64          `gorm:"column:user_id;not null;index:idx_user_id" json:"user_id"`                                      // 用户ID
	EntityType    EntityType     `gorm:"column:entity_type;size:50;not null;index:idx_entity;index:idx_user_entity" json:"entity_type"` // 业务实体类型
	EntityID      string         `gorm:"column:entity_id;size:36;not null;index:idx_entity;index:idx_user_entity" json:"entity_id"`     // 业务实体ID
	OperationType OperationType  `gorm:"column:operation_type;size:50;not null" json:"operation_type"`                                  // 操作类型
	UsageType     TokenUsageType `gorm:"column:usage_type;not null" json:"usage_type"`                                                  // Token使用类型

	// Token 使用详情
	InputTokens  int `gorm:"column:input_tokens;default:0" json:"input_tokens"`   // 输入Token数
	OutputTokens int `gorm:"column:output_tokens;default:0" json:"output_tokens"` // 输出Token数
	TotalTokens  int `gorm:"column:total_tokens;default:0" json:"total_tokens"`   // 总Token数

	// 模型和功能信息
	ModelName   string `gorm:"column:model_name;size:100" json:"model_name"`     // 使用的模型名称
	Provider    string `gorm:"column:provider;size:50" json:"provider"`          // 提供商: gemini, hailuo等
	FeatureName string `gorm:"column:feature_name;size:100" json:"feature_name"` // 功能名称

	// 关联信息
	TaskID  string `gorm:"column:task_id;size:36;index:idx_task_id" json:"task_id"`    // 关联的AI任务ID
	StoryID string `gorm:"column:story_id;size:36;index:idx_story_id" json:"story_id"` // 关联的故事ID（如果适用）

	// 成本和计费
	CostAmount float64 `gorm:"column:cost_amount;type:decimal(10,4);default:0" json:"cost_amount"`  // 成本金额
	Currency   string  `gorm:"column:currency;size:10;default:'USD'" json:"currency"`               // 货币类型
	IsBilled   bool    `gorm:"column:is_billed;default:false;index:idx_is_billed" json:"is_billed"` // 是否已计费
	BillingID  string  `gorm:"column:billing_id;size:36" json:"billing_id"`                         // 计费记录ID

	// 元数据
	Metadata string `gorm:"column:metadata;type:json" json:"metadata"` // 扩展元数据（JSON格式）

	// 时间戳
	CreatedAt time.Time `gorm:"column:created_at;not null;index:idx_created_at" json:"created_at"` // 创建时间
}

func (tul TokenUsageLog) TableName() string {
	return "token_usage_log"
}

// GetMetadata 获取元数据
func (tul *TokenUsageLog) GetMetadata() (map[string]interface{}, error) {
	if tul.Metadata == "" {
		return map[string]interface{}{}, nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(tul.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
func (tul *TokenUsageLog) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	tul.Metadata = string(data)
	return nil
}

// CreateTokenUsageLog 创建Token用量日志记录
func CreateTokenUsageLog(ctx context.Context, log *TokenUsageLog) error {
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return DataBase().WithContext(ctx).Create(log).Error
}

// GetTokenUsageLogsByEntity 按业务实体查询日志
func GetTokenUsageLogsByEntity(ctx context.Context, entityType EntityType, entityID string, limit, offset int) ([]*TokenUsageLog, error) {
	var logs []*TokenUsageLog
	err := DataBase().WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// GetTokenUsageLogsByUser 按用户查询日志
func GetTokenUsageLogsByUser(ctx context.Context, userID int64, limit, offset int) ([]*TokenUsageLog, error) {
	var logs []*TokenUsageLog
	err := DataBase().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// GetTokenUsageLogsByTimeRange 按时间范围查询日志
func GetTokenUsageLogsByTimeRange(ctx context.Context, userID int64, startTime, endTime time.Time, limit, offset int) ([]*TokenUsageLog, error) {
	var logs []*TokenUsageLog
	query := DataBase().WithContext(ctx).
		Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startTime, endTime).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

// GetTokenUsageLogsByQuery 多条件查询日志
func GetTokenUsageLogsByQuery(ctx context.Context, userID int64, entityType *EntityType, entityID *string, startTime, endTime *time.Time, limit, offset int) ([]*TokenUsageLog, int64, error) {
	var logs []*TokenUsageLog
	var total int64

	query := DataBase().WithContext(ctx).Model(&TokenUsageLog{}).Where("user_id = ?", userID)

	if entityType != nil {
		query = query.Where("entity_type = ?", *entityType)
	}
	if entityID != nil {
		query = query.Where("entity_id = ?", *entityID)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at < ?", *endTime)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error

	return logs, total, err
}

// GetTokenUsageSummary 汇总统计
func GetTokenUsageSummary(ctx context.Context, userID int64, entityType *EntityType, entityID *string, startTime, endTime *time.Time) (map[string]interface{}, error) {
	var summary struct {
		TotalInputTokens  int     `json:"total_input_tokens"`
		TotalOutputTokens int     `json:"total_output_tokens"`
		TotalTokens       int     `json:"total_tokens"`
		TotalCost         float64 `json:"total_cost"`
		RecordCount       int64   `json:"record_count"`
	}

	query := DataBase().WithContext(ctx).Model(&TokenUsageLog{}).
		Where("user_id = ?", userID)

	if entityType != nil {
		query = query.Where("entity_type = ?", *entityType)
	}
	if entityID != nil {
		query = query.Where("entity_id = ?", *entityID)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at < ?", *endTime)
	}

	err := query.Select(`
		SUM(input_tokens) as total_input_tokens,
		SUM(output_tokens) as total_output_tokens,
		SUM(total_tokens) as total_tokens,
		SUM(cost_amount) as total_cost,
		COUNT(*) as record_count
	`).Scan(&summary).Error

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_input_tokens":  summary.TotalInputTokens,
		"total_output_tokens": summary.TotalOutputTokens,
		"total_tokens":        summary.TotalTokens,
		"total_cost":          summary.TotalCost,
		"record_count":        summary.RecordCount,
	}, nil
}

// GetTokenUsageSummaryByEntityType 按实体类型汇总
func GetTokenUsageSummaryByEntityType(ctx context.Context, userID int64, startTime, endTime *time.Time) (map[EntityType]map[string]interface{}, error) {
	var summaries []struct {
		EntityType        EntityType `json:"entity_type"`
		TotalInputTokens  int        `json:"total_input_tokens"`
		TotalOutputTokens int        `json:"total_output_tokens"`
		TotalTokens       int        `json:"total_tokens"`
		TotalCost         float64    `json:"total_cost"`
		RecordCount       int64      `json:"record_count"`
	}

	query := DataBase().WithContext(ctx).Model(&TokenUsageLog{}).
		Where("user_id = ?", userID)

	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at < ?", *endTime)
	}

	err := query.Select(`
		entity_type,
		SUM(input_tokens) as total_input_tokens,
		SUM(output_tokens) as total_output_tokens,
		SUM(total_tokens) as total_tokens,
		SUM(cost_amount) as total_cost,
		COUNT(*) as record_count
	`).Group("entity_type").Scan(&summaries).Error

	if err != nil {
		return nil, err
	}

	result := make(map[EntityType]map[string]interface{})
	for _, s := range summaries {
		result[s.EntityType] = map[string]interface{}{
			"total_input_tokens":  s.TotalInputTokens,
			"total_output_tokens": s.TotalOutputTokens,
			"total_tokens":        s.TotalTokens,
			"total_cost":          s.TotalCost,
			"record_count":        s.RecordCount,
		}
	}

	return result, nil
}

// MarkLogsAsBilled 标记日志为已计费
func MarkLogsAsBilled(ctx context.Context, logIDs []uint, billingID string) error {
	return DataBase().WithContext(ctx).
		Model(&TokenUsageLog{}).
		Where("id IN ?", logIDs).
		Updates(map[string]interface{}{
			"is_billed":  true,
			"billing_id": billingID,
		}).Error
}

// GetUnbilledLogs 获取未计费的日志
func GetUnbilledLogs(ctx context.Context, userID int64, startTime, endTime *time.Time) ([]*TokenUsageLog, error) {
	var logs []*TokenUsageLog
	query := DataBase().WithContext(ctx).
		Where("user_id = ? AND is_billed = ?", userID, false)

	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at < ?", *endTime)
	}

	err := query.Order("created_at ASC").Find(&logs).Error
	return logs, err
}

// CalculateUnbilledAmount 计算未计费金额
func CalculateUnbilledAmount(ctx context.Context, userID int64, startTime, endTime *time.Time) (float64, error) {
	var total float64
	query := DataBase().WithContext(ctx).Model(&TokenUsageLog{}).
		Where("user_id = ? AND is_billed = ?", userID, false)

	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at < ?", *endTime)
	}

	err := query.Select("COALESCE(SUM(cost_amount), 0)").Scan(&total).Error
	return total, err
}

// ExportTokenUsageLogs 导出日志数据（用于CSV/JSON导出）
func ExportTokenUsageLogs(ctx context.Context, userID int64, entityType *EntityType, entityID *string, startTime, endTime *time.Time) ([]*TokenUsageLog, error) {
	var logs []*TokenUsageLog
	query := DataBase().WithContext(ctx).
		Where("user_id = ?", userID)

	if entityType != nil {
		query = query.Where("entity_type = ?", *entityType)
	}
	if entityID != nil {
		query = query.Where("entity_id = ?", *entityID)
	}
	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at < ?", *endTime)
	}

	err := query.Order("created_at ASC").Find(&logs).Error
	return logs, err
}

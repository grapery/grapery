package pay

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/sirupsen/logrus"
)

// TokenUsageLogService Token用量日志服务接口
type TokenUsageLogService interface {
	// RecordUsageLog 记录用量日志
	RecordUsageLog(ctx context.Context, log *paymodels.TokenUsageLog) error

	// QueryLogs 查询日志（支持多条件）
	QueryLogs(ctx context.Context, userID int64, entityType *paymodels.EntityType, entityID *string, startTime, endTime *time.Time, page, limit int) ([]*paymodels.TokenUsageLog, int64, error)

	// GetSummary 获取汇总统计
	GetSummary(ctx context.Context, userID int64, entityType *paymodels.EntityType, entityID *string, startTime, endTime *time.Time) (map[string]interface{}, error)

	// GetSummaryByEntityType 按实体类型汇总
	GetSummaryByEntityType(ctx context.Context, userID int64, startTime, endTime *time.Time) (map[paymodels.EntityType]map[string]interface{}, error)

	// ExportLogs 导出日志（CSV/JSON）
	ExportLogs(ctx context.Context, userID int64, entityType *paymodels.EntityType, entityID *string, startTime, endTime *time.Time, format string) ([]byte, string, error)

	// GetLogsByEntity 按业务实体查询日志
	GetLogsByEntity(ctx context.Context, entityType paymodels.EntityType, entityID string, limit, offset int) ([]*paymodels.TokenUsageLog, error)

	// CalculateBilling 计算计费金额
	CalculateBilling(ctx context.Context, userID int64, startTime, endTime *time.Time) (float64, error)

	// MarkAsBilled 标记为已计费
	MarkAsBilled(ctx context.Context, logIDs []uint, billingID string) error

	// GetUnbilledLogs 获取未计费日志
	GetUnbilledLogs(ctx context.Context, userID int64, startTime, endTime *time.Time) ([]*paymodels.TokenUsageLog, error)
}

// TokenUsageLogServiceImpl Token用量日志服务实现
type TokenUsageLogServiceImpl struct {
	logger *logrus.Logger
}

// NewTokenUsageLogService 创建Token用量日志服务
func NewTokenUsageLogService(logger *logrus.Logger) TokenUsageLogService {
	return &TokenUsageLogServiceImpl{
		logger: logger,
	}
}

// RecordUsageLog 记录用量日志
func (s *TokenUsageLogServiceImpl) RecordUsageLog(ctx context.Context, log *paymodels.TokenUsageLog) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":      log.UserID,
		"entity_type":  log.EntityType,
		"entity_id":    log.EntityID,
		"operation":    log.OperationType,
		"total_tokens": log.TotalTokens,
		"task_id":      log.TaskID,
	}).Debug("Recording token usage log")

	err := paymodels.CreateTokenUsageLog(ctx, log)
	if err != nil {
		s.logger.WithError(err).Error("Failed to record token usage log")
		return err
	}

	return nil
}

// QueryLogs 查询日志（支持多条件）
func (s *TokenUsageLogServiceImpl) QueryLogs(ctx context.Context, userID int64, entityType *paymodels.EntityType, entityID *string, startTime, endTime *time.Time, page, limit int) ([]*paymodels.TokenUsageLog, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	logs, total, err := paymodels.GetTokenUsageLogsByQuery(ctx, userID, entityType, entityID, startTime, endTime, limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to query token usage logs")
		return nil, 0, err
	}

	return logs, total, nil
}

// GetSummary 获取汇总统计
func (s *TokenUsageLogServiceImpl) GetSummary(ctx context.Context, userID int64, entityType *paymodels.EntityType, entityID *string, startTime, endTime *time.Time) (map[string]interface{}, error) {
	summary, err := paymodels.GetTokenUsageSummary(ctx, userID, entityType, entityID, startTime, endTime)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get token usage summary")
		return nil, err
	}
	return summary, nil
}

// GetSummaryByEntityType 按实体类型汇总
func (s *TokenUsageLogServiceImpl) GetSummaryByEntityType(ctx context.Context, userID int64, startTime, endTime *time.Time) (map[paymodels.EntityType]map[string]interface{}, error) {
	summary, err := paymodels.GetTokenUsageSummaryByEntityType(ctx, userID, startTime, endTime)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get token usage summary by entity type")
		return nil, err
	}
	return summary, nil
}

// ExportLogs 导出日志（CSV/JSON）
func (s *TokenUsageLogServiceImpl) ExportLogs(ctx context.Context, userID int64, entityType *paymodels.EntityType, entityID *string, startTime, endTime *time.Time, format string) ([]byte, string, error) {
	logs, err := paymodels.ExportTokenUsageLogs(ctx, userID, entityType, entityID, startTime, endTime)
	if err != nil {
		s.logger.WithError(err).Error("Failed to export token usage logs")
		return nil, "", err
	}

	switch strings.ToLower(format) {
	case "csv":
		return s.exportToCSV(logs)
	case "json":
		return s.exportToJSON(logs)
	default:
		return s.exportToJSON(logs)
	}
}

// exportToCSV 导出为CSV格式
func (s *TokenUsageLogServiceImpl) exportToCSV(logs []*paymodels.TokenUsageLog) ([]byte, string, error) {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	// 写入表头
	headers := []string{
		"ID", "User ID", "Entity Type", "Entity ID", "Operation Type", "Usage Type",
		"Input Tokens", "Output Tokens", "Total Tokens",
		"Model Name", "Provider", "Feature Name",
		"Task ID", "Story ID",
		"Cost Amount", "Currency", "Is Billed", "Billing ID",
		"Created At",
	}
	if err := writer.Write(headers); err != nil {
		return nil, "", err
	}

	// 写入数据
	for _, log := range logs {
		record := []string{
			fmt.Sprintf("%d", log.ID),
			fmt.Sprintf("%d", log.UserID),
			string(log.EntityType),
			log.EntityID,
			string(log.OperationType),
			fmt.Sprintf("%d", log.UsageType),
			fmt.Sprintf("%d", log.InputTokens),
			fmt.Sprintf("%d", log.OutputTokens),
			fmt.Sprintf("%d", log.TotalTokens),
			log.ModelName,
			log.Provider,
			log.FeatureName,
			log.TaskID,
			log.StoryID,
			fmt.Sprintf("%.4f", log.CostAmount),
			log.Currency,
			fmt.Sprintf("%t", log.IsBilled),
			log.BillingID,
			log.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return nil, "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", err
	}

	return []byte(buf.String()), "text/csv", nil
}

// exportToJSON 导出为JSON格式
func (s *TokenUsageLogServiceImpl) exportToJSON(logs []*paymodels.TokenUsageLog) ([]byte, string, error) {
	data, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return nil, "", err
	}
	return data, "application/json", nil
}

// GetLogsByEntity 按业务实体查询日志
func (s *TokenUsageLogServiceImpl) GetLogsByEntity(ctx context.Context, entityType paymodels.EntityType, entityID string, limit, offset int) ([]*paymodels.TokenUsageLog, error) {
	logs, err := paymodels.GetTokenUsageLogsByEntity(ctx, entityType, entityID, limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get token usage logs by entity")
		return nil, err
	}
	return logs, nil
}

// CalculateBilling 计算计费金额
func (s *TokenUsageLogServiceImpl) CalculateBilling(ctx context.Context, userID int64, startTime, endTime *time.Time) (float64, error) {
	amount, err := paymodels.CalculateUnbilledAmount(ctx, userID, startTime, endTime)
	if err != nil {
		s.logger.WithError(err).Error("Failed to calculate billing amount")
		return 0, err
	}
	return amount, nil
}

// MarkAsBilled 标记为已计费
func (s *TokenUsageLogServiceImpl) MarkAsBilled(ctx context.Context, logIDs []uint, billingID string) error {
	err := paymodels.MarkLogsAsBilled(ctx, logIDs, billingID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to mark logs as billed")
		return err
	}
	return nil
}

// GetUnbilledLogs 获取未计费日志
func (s *TokenUsageLogServiceImpl) GetUnbilledLogs(ctx context.Context, userID int64, startTime, endTime *time.Time) ([]*paymodels.TokenUsageLog, error) {
	logs, err := paymodels.GetUnbilledLogs(ctx, userID, startTime, endTime)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get unbilled logs")
		return nil, err
	}
	return logs, nil
}

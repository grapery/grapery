package pay

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	paypkg "github.com/grapestree/fgrapery/grapery/internal/service/pay"
	"github.com/sirupsen/logrus"
)

// TokenUsageLogHandler Token用量日志处理器
type TokenUsageLogHandler struct {
	logService paypkg.TokenUsageLogService
	logger     *logrus.Logger
}

// NewTokenUsageLogHandler 创建Token用量日志处理器
func NewTokenUsageLogHandler(logService paypkg.TokenUsageLogService, logger *logrus.Logger) *TokenUsageLogHandler {
	return &TokenUsageLogHandler{
		logService: logService,
		logger:     logger,
	}
}

// GetLogs 查询日志列表
// GET /api/vippay/usage/logs?entity_type=storyboard&entity_id=xxx&start_time=xxx&end_time=xxx&page=1&limit=20
func (h *TokenUsageLogHandler) GetLogs(c *gin.Context) {
	userID, ok := RequireUserIDInt64(c)
	if !ok {
		return
	}

	// 解析查询参数
	var entityType *paymodels.EntityType
	if et := c.Query("entity_type"); et != "" {
		etValue := paymodels.EntityType(et)
		entityType = &etValue
	}

	var entityID *string
	if eid := c.Query("entity_id"); eid != "" {
		entityID = &eid
	}

	var startTime *time.Time
	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = &t
		}
	}

	var endTime *time.Time
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = &t
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// 查询日志
	logs, total, err := h.logService.QueryLogs(c.Request.Context(), userID, entityType, entityID, startTime, endTime, page, limit)
	if err != nil {
		h.logger.WithError(err).Error("Failed to query token usage logs")
		HandleError(c, err)
		return
	}

	SuccessLegacy(c, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetSummary 获取汇总统计
// GET /api/vippay/usage/logs/summary?entity_type=storyboard&entity_id=xxx&start_time=xxx&end_time=xxx
func (h *TokenUsageLogHandler) GetSummary(c *gin.Context) {
	userID, ok := RequireUserIDInt64(c)
	if !ok {
		return
	}

	// 解析查询参数
	var entityType *paymodels.EntityType
	if et := c.Query("entity_type"); et != "" {
		etValue := paymodels.EntityType(et)
		entityType = &etValue
	}

	var entityID *string
	if eid := c.Query("entity_id"); eid != "" {
		entityID = &eid
	}

	var startTime *time.Time
	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = &t
		}
	}

	var endTime *time.Time
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = &t
		}
	}

	// 获取汇总
	summary, err := h.logService.GetSummary(c.Request.Context(), userID, entityType, entityID, startTime, endTime)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get token usage summary")
		HandleError(c, err)
		return
	}

	SuccessLegacy(c, summary)
}

// GetSummaryByEntityType 按实体类型汇总
// GET /api/vippay/usage/logs/summary/by-type?start_time=xxx&end_time=xxx
func (h *TokenUsageLogHandler) GetSummaryByEntityType(c *gin.Context) {
	userID, ok := RequireUserIDInt64(c)
	if !ok {
		return
	}

	var startTime *time.Time
	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = &t
		}
	}

	var endTime *time.Time
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = &t
		}
	}

	// 获取汇总
	summary, err := h.logService.GetSummaryByEntityType(c.Request.Context(), userID, startTime, endTime)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get token usage summary by entity type")
		HandleError(c, err)
		return
	}

	// 转换 EntityType 为字符串键
	result := make(map[string]interface{})
	for et, stats := range summary {
		result[string(et)] = stats
	}

	SuccessLegacy(c, result)
}

// ExportLogs 导出日志
// GET /api/vippay/usage/logs/export?format=csv&entity_type=storyboard&entity_id=xxx&start_time=xxx&end_time=xxx
func (h *TokenUsageLogHandler) ExportLogs(c *gin.Context) {
	userID, ok := RequireUserIDInt64(c)
	if !ok {
		return
	}

	// 解析查询参数
	format := c.DefaultQuery("format", "json")

	var entityType *paymodels.EntityType
	if et := c.Query("entity_type"); et != "" {
		etValue := paymodels.EntityType(et)
		entityType = &etValue
	}

	var entityID *string
	if eid := c.Query("entity_id"); eid != "" {
		entityID = &eid
	}

	var startTime *time.Time
	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = &t
		}
	}

	var endTime *time.Time
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = &t
		}
	}

	// 导出日志
	data, contentType, err := h.logService.ExportLogs(c.Request.Context(), userID, entityType, entityID, startTime, endTime, format)
	if err != nil {
		h.logger.WithError(err).Error("Failed to export token usage logs")
		HandleError(c, err)
		return
	}

	// 生成文件名
	filename := "token_usage_logs"
	if format == "csv" {
		filename += ".csv"
	} else {
		filename += ".json"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, contentType, data)
}

// GetLogsByEntity 按业务实体查询日志
// GET /api/vippay/usage/logs/by-entity/:entity_type/:entity_id?limit=20&offset=0
func (h *TokenUsageLogHandler) GetLogsByEntity(c *gin.Context) {
	entityTypeStr, ok := RequireParam(c, "entity_type")
	if !ok {
		return
	}
	entityID, ok := RequireParam(c, "entity_id")
	if !ok {
		return
	}

	entityType := paymodels.EntityType(entityTypeStr)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	// 查询日志
	logs, err := h.logService.GetLogsByEntity(c.Request.Context(), entityType, entityID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get token usage logs by entity")
		HandleError(c, err)
		return
	}

	SuccessLegacy(c, logs)
}

// GetBilling 获取计费汇总
// GET /api/vippay/usage/logs/billing?start_time=xxx&end_time=xxx
func (h *TokenUsageLogHandler) GetBilling(c *gin.Context) {
	userID, ok := RequireUserIDInt64(c)
	if !ok {
		return
	}

	var startTime *time.Time
	if st := c.Query("start_time"); st != "" {
		if t, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = &t
		}
	}

	var endTime *time.Time
	if et := c.Query("end_time"); et != "" {
		if t, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = &t
		}
	}

	// 获取未计费日志
	unbilledLogs, err := h.logService.GetUnbilledLogs(c.Request.Context(), userID, startTime, endTime)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get unbilled logs")
		HandleError(c, err)
		return
	}

	// 计算未计费金额
	unbilledAmount, err := h.logService.CalculateBilling(c.Request.Context(), userID, startTime, endTime)
	if err != nil {
		h.logger.WithError(err).Error("Failed to calculate billing amount")
		HandleError(c, err)
		return
	}

	SuccessLegacy(c, gin.H{
		"unbilled_logs":   unbilledLogs,
		"unbilled_amount": unbilledAmount,
		"log_count":       len(unbilledLogs),
	})
}

// MarkAsBilled 标记为已计费
// POST /api/vippay/usage/logs/mark-billed
func (h *TokenUsageLogHandler) MarkAsBilled(c *gin.Context) {
	var req struct {
		LogIDs    []uint `json:"log_ids" binding:"required"`
		BillingID string `json:"billing_id" binding:"required"`
	}

	if !BindJSON(c, &req) {
		return
	}

	err := h.logService.MarkAsBilled(c.Request.Context(), req.LogIDs, req.BillingID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to mark logs as billed")
		HandleError(c, err)
		return
	}

	SuccessLegacy(c, nil)
}

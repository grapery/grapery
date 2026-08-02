package llmchathandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/grapery/grapery/pkg/cloud/coze"
	llmchatpkg "github.com/grapery/grapery/pkg/llmchat"
	llmchatservice "github.com/grapery/grapery/service/llmchat"
	"github.com/grapery/grapery/service/llmchat/middleware"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/compliance"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 统一API响应结构体
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// NotificationListResponse defines the payload returned for notifications queries.
type NotificationListResponse struct {
	Notifications []NotificationItem `json:"notifications"`
	TotalCount    int64              `json:"total_count"`
	HasMore       bool               `json:"has_more"`
	NextCursor    *int64             `json:"next_cursor,omitempty"`
}

// NotificationItem maps service notification DTOs to handler response.
type NotificationItem struct {
	ID                  int64  `json:"id"`
	Type                string `json:"type"`
	Title               string `json:"title"`
	Content             string `json:"content"`
	IsRead              bool   `json:"is_read"`
	RelatedUserID       *int64 `json:"related_user_id,omitempty"`
	RelatedStoryID      *int64 `json:"related_story_id,omitempty"`
	RelatedStoryBoardID *int64 `json:"related_storyboard_id,omitempty"`
	RelatedCommentID    *int64 `json:"related_comment_id,omitempty"`
	ExtraData           gin.H  `json:"extra_data,omitempty"`
	CreatedAt           int64  `json:"created_at"`
	UpdatedAt           int64  `json:"updated_at"`
}

var (
	fetchNotificationsFunc       = llmchatservice.GetRecentNotifications
	markAllNotificationsReadFunc = llmchatservice.MarkAllNotificationsRead
	markNotificationReadFunc     = llmchatservice.MarkNotificationRead
)

// SSEPayload 用于SSE推送的结构体，包含event和data字段，便于前端解析
type SSEPayload struct {
	Event     string `json:"event"`
	Data      string `json:"data"`
	MessageId string `json:"message_id"`
}

// 注册自定义验证器
func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 注册字符串不为空验证器
		v.RegisterValidation("notempty", validateNotEmpty)
	}
}

// validateNotEmpty 验证字符串不为空
func validateNotEmpty(fl validator.FieldLevel) bool {
	if str, ok := fl.Field().Interface().(string); ok {
		return strings.TrimSpace(str) != ""
	}
	return false
}

// RegisterLLMChatRoutes 注册llmchat相关路由，使用鉴权和限流中间件
func RegisterLLMChatRoutes(r *gin.Engine) {
	api := r.Group("/api/llmchat")
	api.Use(middleware.AuthMiddleware())
	api.Use(middleware.RateLimitMiddleware())
	{
		api.GET("/session", GetSessionHandler)
		api.GET("/session/list", GetSessionListHandler)
		api.POST("/session", CreateSessionHandler)
		api.POST("/role/session", CreateRoleSessionHandler)
		api.POST("/session/:id/messages", SessionMessageHandler)
		api.POST("/role/session/:id/clear", ClearSessionHandler)
		api.POST("/message", SendMessageHandler)
		api.POST("/message/:id/retry", RetryMessageHandler)
		api.POST("/message/:id/feedback", FeedbackMessageHandler)
		api.POST("/message/:id/interrupt", InterruptMessageHandler)
		api.GET("/health", HealthHandler)
		api.POST("/bot/:id/publish", PublishBotHandler)
		api.POST("/bot/create", CreateBotHandler)
		api.PUT("/bot/:id/update", UpdateBotHandler)
		api.DELETE("/bot/:id/delete", DeleteBotHandler)
		api.GET("/bot/:id/detail", GetBotDetailHandler)
		api.PUT("/bot/:id/detail", UpdateBotDetailHandler)

		notificationGroup := api.Group("/notifications")
		{
			notificationGroup.GET("", GetSystemNotificationsHandler)
			notificationGroup.POST("/read_all", ReadAllSystemNotificationsHandler)
			notificationGroup.POST(":id/read", ReadSingleNotificationHandler)
		}
	}
}

func CreateBotHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[CreateBotHandler] handler入口")
	var req struct {
		UserID     int64  `json:"user_id" binding:"required"`
		RoleId     int64  `json:"role_id" binding:"required"`
		Name       string `json:"name" binding:"required,notempty"`
		IconFileID string `json:"icon_file_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[CreateBotHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[CreateBotHandler] 参数绑定成功", zap.Int64("userID", req.UserID), zap.Int64("roleId", req.RoleId), zap.String("name", req.Name), zap.String("iconFileID", req.IconFileID))
	res, err := llmchatservice.CreatBotService(c.Request.Context(), req.UserID, req.RoleId, req.Name, req.IconFileID)
	if err != nil {
		log.Error("[CreateBotHandler] CreatBotService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[CreateBotHandler] CreatBotService成功", zap.String("botId", res))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: map[string]string{"bot_id": res}})
}

func PublishBotHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[PublishBotHandler] handler入口")
	var req struct {
		UserID int64  `json:"user_id" binding:"required"`
		RoleId int64  `json:"role_id" binding:"required"`
		BotId  string `json:"bot_id" binding:"required,notempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[PublishBotHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[PublishBotHandler] 参数绑定成功", zap.Int64("userID", req.UserID), zap.Int64("roleId", req.RoleId), zap.String("botId", req.BotId))
	res, err := llmchatservice.PublishBotService(c.Request.Context(), req.UserID, req.RoleId, req.BotId)
	if err != nil {
		log.Error("[PublishBotHandler] PublishBotService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[PublishBotHandler] PublishBotService成功", zap.String("botId", res))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: map[string]string{"bot_id": res}})
}

func UpdateBotHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[UpdateBotHandler] handler入口")
	var req struct {
		UserID int64  `json:"user_id" binding:"required"`
		RoleId int64  `json:"role_id" binding:"required"`
		BotId  string `json:"bot_id" binding:"required,notempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[UpdateBotHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[UpdateBotHandler] 参数绑定成功", zap.Int64("userID", req.UserID), zap.Int64("roleId", req.RoleId), zap.String("botId", req.BotId))
	res, err := llmchatservice.UpdateBotService(c.Request.Context(), req.UserID, req.RoleId, req.BotId)
	if err != nil {
		log.Error("[UpdateBotHandler] UpdateBotService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[UpdateBotHandler] UpdateBotService成功", zap.String("botId", res))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: map[string]string{"bot_id": res}})
	return
}

func DeleteBotHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[DeleteBotHandler] handler入口")
	var req struct {
		UserID int64  `json:"user_id" binding:"required"`
		RoleId int64  `json:"role_id" binding:"required"`
		BotId  string `json:"bot_id" binding:"required,notempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[DeleteBotHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[DeleteBotHandler] 参数绑定成功", zap.Int64("userID", req.UserID), zap.Int64("roleId", req.RoleId), zap.String("botId", req.BotId))
	err := llmchatservice.DeleteBotService(c.Request.Context(), req.UserID, req.RoleId, req.BotId)
	if err != nil {
		log.Error("[DeleteBotHandler] DeleteBotService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[DeleteBotHandler] DeleteBotService成功")
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: struct{}{}})
}

func GetBotDetailHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[GetBotDetailHandler] handler入口")
	var req struct {
		UserID int64  `json:"user_id" binding:"required"`
		RoleId int64  `json:"role_id" binding:"required"`
		BotId  string `json:"bot_id" binding:"required,notempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[GetBotDetailHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[GetBotDetailHandler] 参数绑定成功", zap.Int64("userID", req.UserID), zap.Int64("roleId", req.RoleId), zap.String("botId", req.BotId))
	bot, err := llmchatservice.GetBotService(c.Request.Context(), req.UserID, req.RoleId, req.BotId)
	if err != nil {
		log.Error("[GetBotDetailHandler] GetBotDetailService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[GetBotDetailHandler] GetBotDetailService成功", zap.Any("bot", bot))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: bot})
}

func UpdateBotDetailHandler(c *gin.Context) {

}

func HealthHandler(c *gin.Context) {
	// 健康检查接口，返回200 OK
	c.JSON(http.StatusOK, APIResponse{
		Code:    http.StatusOK,
		Message: "OK",
		Data:    struct{}{},
	})
}

// GetSystemNotificationsHandler returns latest notifications for the authenticated user.
func GetSystemNotificationsHandler(c *gin.Context) {
	log := log.Log()
	userID := c.GetInt64(utils.UserIdKey)
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	log.Info("[GetSystemNotificationsHandler] 请求入口", zap.Int64("userID", userID), zap.Int("limit", limit))
	notifications, hasMore, total, err := fetchNotificationsFunc(c.Request.Context(), userID, limit)
	if err != nil {
		log.Error("[GetSystemNotificationsHandler] 服务调用失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	respItems := make([]NotificationItem, 0, len(notifications))
	for _, n := range notifications {
		item := NotificationItem{
			ID:                  n.ID,
			Type:                string(n.Type),
			Title:               n.Title,
			Content:             n.Content,
			IsRead:              n.IsRead,
			RelatedUserID:       n.RelatedUserID,
			RelatedStoryID:      n.RelatedStoryID,
			RelatedStoryBoardID: n.RelatedStoryBoardID,
			RelatedCommentID:    n.RelatedCommentID,
			CreatedAt:           n.CreatedAt,
			UpdatedAt:           n.UpdatedAt,
		}
		if n.ExtraData != nil {
			item.ExtraData = gin.H{}
			for k, v := range n.ExtraData {
				item.ExtraData[k] = v
			}
		}
		respItems = append(respItems, item)
	}
	response := NotificationListResponse{
		Notifications: respItems,
		TotalCount:    total,
		HasMore:       hasMore,
	}
	if len(respItems) > 0 {
		lastCreatedAt := respItems[len(respItems)-1].CreatedAt
		response.NextCursor = &lastCreatedAt
	}
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: response})
}

// ReadAllSystemNotificationsHandler marks all notifications as read for current user.
func ReadAllSystemNotificationsHandler(c *gin.Context) {
	log := log.Log()
	userID := c.GetInt64(utils.UserIdKey)
	log.Info("[ReadAllSystemNotificationsHandler] 请求入口", zap.Int64("userID", userID))
	if err := markAllNotificationsReadFunc(c.Request.Context(), userID); err != nil {
		log.Error("[ReadAllSystemNotificationsHandler] 服务调用失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: struct{}{}})
}

// ReadSingleNotificationHandler marks an individual notification as read.
func ReadSingleNotificationHandler(c *gin.Context) {
	log := log.Log()
	userID := c.GetInt64(utils.UserIdKey)
	notificationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || notificationID <= 0 {
		log.Warn("[ReadSingleNotificationHandler] 无效的通知ID", zap.String("id", c.Param("id")))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: "invalid notification id", Data: struct{}{}})
		return
	}
	log.Info("[ReadSingleNotificationHandler] 请求入口", zap.Int64("userID", userID), zap.Int64("notificationID", notificationID))
	if err := markNotificationReadFunc(c.Request.Context(), userID, notificationID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Warn("[ReadSingleNotificationHandler] 通知不存在或不属于该用户", zap.Int64("notificationID", notificationID), zap.Int64("userID", userID))
			c.JSON(http.StatusNotFound, APIResponse{Code: http.StatusNotFound, Message: "notification not found", Data: struct{}{}})
			return
		}
		log.Error("[ReadSingleNotificationHandler] 服务调用失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: struct{}{}})
}

// CreateSessionHandler 创建会话
func CreateSessionHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[CreateSessionHandler] handler入口")
	var req struct {
		UserID int64  `json:"user_id" binding:"required"`
		Name   string `json:"name" binding:"required,notempty"`
		RoleId string `json:"role_id" binding:"required,notempty"`
		BotId  string `json:"bot_id" binding:"required,notempty"`
	}
	req.UserID = c.GetInt64(utils.UserIdKey)
	log.Info("[CreateSessionHandler] 解析参数", zap.Int64("userID", req.UserID))
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[CreateSessionHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	roleId, err := strconv.ParseInt(req.RoleId, 10, 64)
	if err != nil {
		log.Error("[CreateSessionHandler] 解析roleId失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	botId, err := llmchatservice.GetBotService(c.Request.Context(), req.UserID, roleId, req.BotId)
	if err != nil {
		log.Error("[CreateSessionHandler] GetBotService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	if req.RoleId != botId {
		req.BotId = botId
	} else {
		req.BotId = botId
	}
	log.Info("[CreateSessionHandler] 参数绑定成功", zap.String("name", req.Name), zap.String("role_id", req.RoleId), zap.String("bot_id", req.BotId))
	res, err := llmchatservice.CreateSessionService(c.Request.Context(), req.UserID, req.Name, req.RoleId, botId)
	if err != nil {
		log.Error("[CreateSessionHandler] CreateSessionService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[CreateSessionHandler] 创建会话成功", zap.Any("session", res))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: res})
}

func GetSessionHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[GetSessionHandler] handler入口")
	var req struct {
		UserID int64 `form:"user_id" binding:"required"`
		RoleID int64 `form:"role_id" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Error("[GetSessionHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[GetSessionHandler] 参数绑定成功", zap.Int64("user_id", req.UserID), zap.Int64("role_id", req.RoleID))
	session, err := llmchatservice.GetSessionService(c.Request.Context(), req.UserID, req.RoleID)
	if err != nil {
		log.Error("[GetSessionHandler] GetSessionService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[GetSessionHandler] 获取session成功", zap.Any("session", session))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: session})
}

func GetSessionListHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[GetSessionListHandler] handler入口")

	// 1. 获取参数 user_id, page, page_size
	var req struct {
		UserID   int64 `form:"user_id" binding:"required"`
		Page     int   `form:"page"`
		PageSize int   `form:"page_size"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		log.Error("[GetSessionListHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	log.Info("[GetSessionListHandler] 参数绑定成功", zap.Int64("user_id", req.UserID), zap.Int("page", req.Page), zap.Int("page_size", req.PageSize))

	// 2. 获取全部会话（已按时间倒序）
	sessions, err := llmchatservice.GetUserSessionListService(c.Request.Context(), req.UserID)
	if err != nil {
		log.Error("[GetSessionListHandler] 获取会话列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}

	// 3. 分页处理
	total := len(sessions)
	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	pagedSessions := sessions[start:end]
	hasMore := end < total

	// 4. 返回
	c.JSON(http.StatusOK, APIResponse{
		Code:    http.StatusOK,
		Message: "success",
		Data: gin.H{
			"sessions": pagedSessions,
			"has_more": hasMore,
			"total":    total,
		},
	})
}

// CreateRoleSessionHandler 创建角色会话
func CreateRoleSessionHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[CreateRoleSessionHandler] handler入口")
	var req struct {
		UserID int64  `json:"user_id" binding:"required"`
		Name   string `json:"name" binding:"required,notempty"`
		RoleId string `json:"role_id" binding:"required,notempty"`
		BotId  string `json:"bot_id" binding:"required,notempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[CreateRoleSessionHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[CreateRoleSessionHandler] 参数绑定成功", zap.String("role_id", req.BotId), zap.Int64("user_id", req.UserID), zap.String("name", req.Name), zap.String("bot_id", req.BotId))
	res, err := llmchatservice.CreateSessionService(c.Request.Context(), req.UserID, req.Name, req.RoleId, req.BotId)
	if err != nil {
		log.Error("[CreateRoleSessionHandler] CreateSessionService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[CreateRoleSessionHandler] 创建角色会话成功", zap.Any("session", res))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: res})
}

func ClearSessionHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[ClearSessionHandler] handler入口")
	sessionID := c.Param("id")
	log.Info("[ClearSessionHandler] 获取sessionID", zap.String("sessionID", sessionID))
	err := llmchatservice.ClearSessionService(c.Request.Context(), sessionID)
	if err != nil {
		log.Error("[ClearSessionHandler] ClearSessionService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[ClearSessionHandler] 清理会话成功", zap.String("sessionID", sessionID))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: struct{}{}})
}

// SendMessageHandler 发送消息，支持SSE
func SendMessageHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[SendMessageHandler] handler入口")
	var req struct {
		SessionID string `json:"session_id" binding:"required,notempty"`
		Content   string `json:"content" binding:"required,notempty"`
		ImageURL  string `json:"image_url"`
	}
	userID := c.GetInt64(utils.UserIdKey)
	log.Info("[SendMessageHandler] 解析参数", zap.Int64("userID", userID))
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[SendMessageHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	isPass, err := compliance.TextCompliance(req.Content)
	if err != nil {
		log.Error("[SendMessageHandler] 简介合规检测失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: "合规检测失败", Data: struct{}{}})
		return
	}
	if !isPass {
		log.Error("[SendMessageHandler] 简介合规检测失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: "简介不合规", Data: struct{}{}})
		return
	}

	session, err := llmchatservice.GetSessionBySessionID(c.Request.Context(), req.SessionID)
	if err != nil {
		log.Error("[SendMessageHandler] GetSessionService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	if session == nil {
		log.Error("[SendMessageHandler] 会话不存在", zap.String("session_id", req.SessionID))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: "session not found", Data: struct{}{}})
		return
	}
	log.Info("[SendMessageHandler] 参数绑定成功", zap.String("session_id", req.SessionID), zap.String("content", req.Content))
	assists := make(map[string]string)
	if req.ImageURL != "" {
		assists["image_url"] = req.ImageURL
	}
	res, err := llmchatservice.SendMessageService(c.Request.Context(), req.SessionID, userID, req.Content, assists)
	if err != nil {
		log.Error("[SendMessageHandler] SendMessageService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	msg, _ := res.(*llmchatpkg.LLMChatMessage)
	log.Info("[SendMessageHandler] SendMessageService成功", zap.String("session_id", req.SessionID), zap.String("message_id", msg.MessageId), zap.Int64("userID", userID))
	// 设置SSE响应头，必须在最前面
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()
	ctx := c.Request.Context()
	streamChan := make(chan string, 100)
	answerMap := make(map[string][]coze.AnswerOrFollowUp)
	log.Info("[SendMessageHandler] 启动流式SendMessageStream", zap.String("session_id", req.SessionID), zap.String("message_id", msg.MessageId), zap.Int64("userID", userID), zap.String("content", req.Content))
	go func() {
		err = llmchatpkg.GetLLMChatEngine().SendMessageStream(ctx, req.SessionID, session.BotId, msg.MessageId, userID, req.Content, streamChan, answerMap)
		if err != nil {
			log.Error("[SendMessageHandler] SendMessageStream失败", zap.Error(err))
			c.Writer.Write([]byte("event: error\ndata: " + err.Error() + "\n\n"))
			c.Writer.Flush()
			return
		}
		log.Info("[SendMessageHandler] SendMessageStream成功")
	}()
	log.Info("[SendMessageHandler] 流式通道启动成功，开始推送SSE数据")
	for {
		chunk, ok := <-streamChan
		if !ok {
			log.Info("[SendMessageHandler] streamChan已关闭，结束SSE推送")
			// 推送结束事件，event为done，data为[DONE]
			payload := SSEPayload{
				MessageId: msg.MessageId,
				Event:     "done",
				Data:      "[DONE]",
			}
			b, _ := json.Marshal(payload)
			c.Writer.Write(b) // SSE事件块分隔
			c.Writer.Write([]byte("\n\n"))
			c.Writer.Flush()
			break
		}
		log.Info("[SendMessageHandler] 推送SSE chunk", zap.String("chunk", chunk))
		// 普通流式内容，event为conversation.message.delta
		payload := SSEPayload{
			Event: "conversation.message.delta",
			Data:  chunk,
		}
		b, _ := json.Marshal(payload)
		c.Writer.Write(b) // SSE事件块分隔
		c.Writer.Write([]byte("\n\n"))
		c.Writer.Flush()
	}
	log.Info("[SendMessageHandler] SSE流结束，推送event: done")
	// c.Writer.Write([]byte("event: done\ndata: [DONE]\n\n"))
	// c.Writer.Flush()
	// log.Info("[SendMessageHandler] SSE流式响应完成，answerMap收集内容", zap.Any("answerMap", answerMap))
	// // 统一结构体返回
	// c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: res})
}

// RetryMessageHandler 重试消息，支持SSE
func RetryMessageHandler(c *gin.Context) {
	log := log.Log()
	log.Info("[RetryMessageHandler] handler入口")
	var req struct {
		SessionID string `json:"session_id" binding:"required,notempty"`
		MessageID int64  `json:"message_id" binding:"required"`
		Msg       string `json:"msg" binding:"required,notempty"`
	}
	userID := c.GetInt64(utils.UserIdKey)
	log.Info("[RetryMessageHandler] 解析参数", zap.Int64("userID", userID))
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error("[RetryMessageHandler] 参数绑定失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	log.Info("[RetryMessageHandler] 参数绑定成功", zap.String("session_id", req.SessionID), zap.Int64("message_id", req.MessageID), zap.String("msg", req.Msg))
	session, err := llmchatservice.GetSessionBySessionID(c.Request.Context(), req.SessionID)
	if err != nil {
		log.Error("[SendMessageHandler] GetSessionService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	if session == nil {
		log.Error("[SendMessageHandler] 会话不存在", zap.String("session_id", req.SessionID))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: "session not found", Data: struct{}{}})
		return
	}
	// 复制消息
	newMsg, err := llmchatservice.CopyMessageService(c.Request.Context(), req.MessageID, userID)
	if err != nil {
		log.Error("[RetryMessageHandler] CopyMessageService失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	isPass, err := compliance.TextCompliance(newMsg.Content)
	if err != nil {
		log.Error("[RetryMessageHandler] 简介合规检测失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: "合规检测失败", Data: struct{}{}})
		return
	}
	if !isPass {
		log.Error("[RetryMessageHandler] 简介合规检测失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: "简介不合规", Data: struct{}{}})
		return
	}
	log.Info("[RetryMessageHandler] CopyMessageService成功", zap.String("new_message_id", newMsg.MessageId))

	// 设置SSE响应头
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()
	ctx := c.Request.Context()
	streamChan := make(chan string, 100)
	answerMap := make(map[string][]coze.AnswerOrFollowUp)
	log.Info("[RetryMessageHandler] 启动流式SendMessageStream", zap.String("session_id", req.SessionID), zap.String("message_id", newMsg.MessageId), zap.Int64("userID", userID), zap.String("msg", req.Msg))
	go func() {
		err = llmchatpkg.GetLLMChatEngine().SendMessageStream(ctx, req.SessionID, session.BotId, newMsg.MessageId, userID, req.Msg, streamChan, answerMap)
		if err != nil {
			log.Error("[RetryMessageHandler] SendMessageStream失败", zap.Error(err))
			c.Writer.Write([]byte("event: error\ndata: " + err.Error() + "\n\n"))
			c.Writer.Flush()
			return
		}
		log.Info("[RetryMessageHandler] SendMessageStream成功")
	}()
	log.Info("[RetryMessageHandler] 流式通道启动成功，开始推送SSE数据")
	for {
		chunk, ok := <-streamChan
		if !ok {
			log.Info("[RetryMessageHandler] streamChan已关闭，结束SSE推送")
			payload := SSEPayload{
				Event: "done",
				Data:  "[DONE]",
			}
			b, _ := json.Marshal(payload)
			c.Writer.Write(b)
			c.Writer.Write([]byte("\n\n"))
			c.Writer.Flush()
			break
		}
		log.Info("[RetryMessageHandler] 推送SSE chunk", zap.String("chunk", chunk))
		payload := SSEPayload{
			Event: "conversation.message.delta",
			Data:  chunk,
		}
		b, _ := json.Marshal(payload)
		c.Writer.Write(b)
		c.Writer.Write([]byte("\n\n"))
		c.Writer.Flush()
	}
	log.Info("[RetryMessageHandler] SSE流结束，推送event: done")
	c.Writer.Write([]byte("event: done\ndata: [DONE]\n\n"))
	c.Writer.Flush()
	log.Info("[RetryMessageHandler] SSE流式响应完成，answerMap收集内容", zap.Any("answerMap", answerMap))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: newMsg})
}

// FeedbackMessageHandler 消息反馈
func FeedbackMessageHandler(c *gin.Context) {
	msgID := c.Param("id")
	var req struct {
		Type   int   `json:"type"` // like/dislike/complaint
		UserID int64 `json:"user_id" binding:"required"`
	}
	// 日志：收到反馈请求
	log.Log().Info("[FeedbackMessageHandler] 收到反馈请求", zap.String("msgID", msgID))
	if err := c.ShouldBindJSON(&req); err != nil {
		// 日志：参数绑定失败
		log.Log().Warn("[FeedbackMessageHandler] 参数绑定失败", zap.Error(err), zap.String("msgID", msgID))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: err.Error(), Data: struct{}{}})
		return
	}
	// 日志：参数绑定成功
	log.Log().Info("[FeedbackMessageHandler] 参数绑定成功", zap.String("msgID", msgID), zap.Int64("userID", req.UserID), zap.Int("type", req.Type))
	res, err := llmchatservice.FeedbackMessageService(c.Request.Context(), msgID, req.UserID, req.Type)
	if err != nil {
		// 日志：服务处理失败
		log.Log().Error("[FeedbackMessageHandler] 服务处理失败", zap.Error(err), zap.String("msgID", msgID), zap.Int64("userID", req.UserID), zap.Int("type", req.Type))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	// 日志：反馈成功
	log.Log().Info("[FeedbackMessageHandler] 反馈成功", zap.String("msgID", msgID), zap.Int64("userID", req.UserID), zap.Int("type", req.Type))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: res})
}

// InterruptMessageHandler 中断消息
func InterruptMessageHandler(c *gin.Context) {
	msgID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	// 日志：收到中断请求
	log.Log().Info("[InterruptMessageHandler] 收到中断请求", zap.String("msgID", c.Param("id")))
	if err != nil {
		// 日志：参数解析失败
		log.Log().Warn("[InterruptMessageHandler] 参数解析失败", zap.Error(err), zap.String("msgID", c.Param("id")))
		c.JSON(http.StatusBadRequest, APIResponse{Code: http.StatusBadRequest, Message: "invalid message id", Data: struct{}{}})
		return
	}
	// 日志：参数解析成功
	log.Log().Info("[InterruptMessageHandler] 参数解析成功", zap.Int64("msgID", msgID))
	err = llmchatservice.InterruptMessageService(c.Request.Context(), msgID)
	if err != nil {
		// 日志：服务处理失败
		log.Log().Error("[InterruptMessageHandler] 服务处理失败", zap.Error(err), zap.Int64("msgID", msgID))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	// 日志：中断成功
	log.Log().Info("[InterruptMessageHandler] 中断成功", zap.Int64("msgID", msgID))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "interrupted", Data: struct{}{}})
}

func SessionMessageHandler(c *gin.Context) {
	sessionID := c.Param("id")
	params := c.Request.URL.Query()
	page, _ := strconv.Atoi(params.Get("page"))
	pageSize, _ := strconv.Atoi(params.Get("page_size"))
	messageID := params.Get("message_id")
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 10
	}
	// 日志：收到会话消息请求
	log.Log().Info("[SessionMessageHandler] 收到会话消息请求", zap.String("sessionID", sessionID), zap.Int("page", page), zap.Int("pageSize", pageSize), zap.String("messageID", messageID))
	var msgs []*llmchatpkg.LLMChatMessage
	var hasMore bool
	var err error
	if messageID != "" {
		msgs, hasMore, err = llmchatservice.SessionMessagePageByMessageID(c.Request.Context(), sessionID, messageID, pageSize)
		// 日志：按messageID分页
		log.Log().Info("[SessionMessageHandler] 按messageID分页", zap.String("sessionID", sessionID), zap.String("messageID", messageID), zap.Int("pageSize", pageSize))
	} else {
		msgs, hasMore, err = llmchatservice.SessionMessageService(c.Request.Context(), sessionID, page, pageSize)
		// 日志：按页码分页
		log.Log().Info("[SessionMessageHandler] 按页码分页", zap.String("sessionID", sessionID), zap.Int("page", page), zap.Int("pageSize", pageSize))
	}
	if err != nil {
		// 日志：服务处理失败
		log.Log().Error("[SessionMessageHandler] 服务处理失败", zap.Error(err), zap.String("sessionID", sessionID))
		c.JSON(http.StatusInternalServerError, APIResponse{Code: http.StatusInternalServerError, Message: err.Error(), Data: struct{}{}})
		return
	}
	// 日志：查询成功
	log.Log().Info("[SessionMessageHandler] 查询成功", zap.String("sessionID", sessionID), zap.Int("msgCount", len(msgs)), zap.Bool("hasMore", hasMore))
	c.JSON(http.StatusOK, APIResponse{Code: http.StatusOK, Message: "success", Data: gin.H{"msgs": msgs, "has_more": hasMore}})
}

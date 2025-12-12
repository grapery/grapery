package http

import (
	"github.com/gin-gonic/gin"
	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

// RegisterDevice 注册设备（用于接收 APNs 推送）
// POST /api/devices/register
func (h *Handler) RegisterDevice(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		DeviceToken string `json:"deviceToken" binding:"required"`
		Platform    string `json:"platform" binding:"required"` // ios, macos, ipados
		AppVersion  string `json:"appVersion"`
		DeviceModel string `json:"deviceModel"`
		OSVersion   string `json:"osVersion"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// 验证平台
	validPlatforms := map[string]bool{
		"ios":     true,
		"macos":   true,
		"ipados":  true,
		"watchos": true,
		"tvos":    true,
	}
	if !validPlatforms[req.Platform] {
		InvalidParams(c, "invalid platform")
		return
	}

	if err := h.svc.RegisterDeviceToken(c.Request.Context(), userID, req.DeviceToken, req.Platform); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"message": "device registered successfully",
		"userId":  userID,
	})
}

// UnregisterDevice 注销设备
// POST /api/devices/unregister
func (h *Handler) UnregisterDevice(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		DeviceToken string `json:"deviceToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	if err := h.svc.UnregisterDeviceToken(c.Request.Context(), userID, req.DeviceToken); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"message": "device unregistered successfully",
	})
}

// UpdateBadge 更新应用徽章数
// POST /api/devices/badge
func (h *Handler) UpdateBadge(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		Count int `json:"count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	if err := h.svc.UpdateDeviceTokenBadge(c.Request.Context(), userID, req.Count); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{
		"message": "badge updated successfully",
	})
}

// TestPushNotification 测试推送通知（开发用）
// POST /api/devices/test-push
func (h *Handler) TestPushNotification(c *gin.Context) {
	userID := authPkg.GetUserID(c)
	if userID == "" {
		Unauthorized(c, "not authenticated")
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	// 创建测试通知
	notification := &struct {
		ID      string
		UserID  string
		Type    string
		Title   string
		Content string
		Link    string
	}{
		ID:      "test",
		UserID:  userID,
		Type:    "system",
		Title:   req.Title,
		Content: req.Message,
		Link:    "",
	}

	// 发送推送（这里需要实现实际的推送逻辑）
	// err := h.svc.SendNotificationToAPNs(c.Request.Context(), userID, notification)

	Success(c, gin.H{
		"message":      "test notification sent",
		"notification": notification,
	})
}

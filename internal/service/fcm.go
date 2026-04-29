package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"go.uber.org/zap"
	"golang.org/x/oauth2/google"
)

// FCMService Firebase Cloud Messaging Service
type FCMService struct {
	projectID       string
	credentialsJSON []byte
	httpClient      *http.Client
	accessToken     string
	tokenExpiry     time.Time
	tokenMutex      sync.RWMutex
	enabled         bool
	logger          *zap.Logger
}

// FCMConfig FCM 配置
type FCMConfig struct {
	ProjectID       string
	CredentialsJSON []byte // Firebase 服务账号 JSON 密钥
}

// NewFCMService 创建 FCM 服务实例
func NewFCMService(config *FCMConfig, logger *zap.Logger) *FCMService {
	svc := &FCMService{
		projectID:       config.ProjectID,
		credentialsJSON: config.CredentialsJSON,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled: false,
		logger:  logger,
	}

	// 验证配置
	if config.ProjectID != "" && len(config.CredentialsJSON) > 0 {
		svc.enabled = true
		logger.Info("FCM service initialized",
			zap.String("projectId", config.ProjectID))
	} else {
		logger.Warn("FCM service not configured, push notifications disabled")
	}

	return svc
}

// IsEnabled 检查 FCM 服务是否启用
func (f *FCMService) IsEnabled() bool {
	return f.enabled
}

// FCMMessage FCM 消息结构
type FCMMessage struct {
	Message struct {
		Token        string            `json:"token,omitempty"`
		Topic        string            `json:"topic,omitempty"`
		Condition    string            `json:"condition,omitempty"`
		Notification *FCMNotification  `json:"notification,omitempty"`
		Data         map[string]string `json:"data,omitempty"`
		Android      *FCMAndroidConfig `json:"android,omitempty"`
		Webpush      *FCMWebpushConfig `json:"webpush,omitempty"`
	} `json:"message"`
}

// FCMNotification FCM 通知内容
type FCMNotification struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body,omitempty"`
	Image string `json:"image,omitempty"`
}

// FCMAndroidConfig Android 特定配置
type FCMAndroidConfig struct {
	Priority     string                  `json:"priority,omitempty"` // normal, high
	TTL          string                  `json:"ttl,omitempty"`      // e.g., "86400s"
	CollapseKey  string                  `json:"collapse_key,omitempty"`
	Notification *FCMAndroidNotification `json:"notification,omitempty"`
}

// FCMAndroidNotification Android 通知配置
type FCMAndroidNotification struct {
	Title                string `json:"title,omitempty"`
	Body                 string `json:"body,omitempty"`
	Icon                 string `json:"icon,omitempty"`
	Color                string `json:"color,omitempty"`
	Sound                string `json:"sound,omitempty"`
	Tag                  string `json:"tag,omitempty"`
	ClickAction          string `json:"click_action,omitempty"`
	ChannelID            string `json:"channel_id,omitempty"`
	DefaultSound         bool   `json:"default_sound,omitempty"`
	NotificationPriority string `json:"notification_priority,omitempty"` // PRIORITY_MIN, PRIORITY_LOW, PRIORITY_DEFAULT, PRIORITY_HIGH, PRIORITY_MAX
}

// FCMWebpushConfig Webpush 配置
type FCMWebpushConfig struct {
	Headers      map[string]string `json:"headers,omitempty"`
	Notification map[string]string `json:"notification,omitempty"`
}

// FCMResponse FCM 响应
type FCMResponse struct {
	Name  string `json:"name,omitempty"`
	Error *struct {
		Code    int    `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
		Status  string `json:"status,omitempty"`
	} `json:"error,omitempty"`
}

// getAccessToken 获取 OAuth2 访问令牌
func (f *FCMService) getAccessToken(ctx context.Context) (string, error) {
	f.tokenMutex.RLock()
	if f.accessToken != "" && time.Now().Before(f.tokenExpiry.Add(-5*time.Minute)) {
		token := f.accessToken
		f.tokenMutex.RUnlock()
		return token, nil
	}
	f.tokenMutex.RUnlock()

	f.tokenMutex.Lock()
	defer f.tokenMutex.Unlock()

	// 双重检查
	if f.accessToken != "" && time.Now().Before(f.tokenExpiry.Add(-5*time.Minute)) {
		return f.accessToken, nil
	}

	// 使用服务账号获取新令牌
	credentials, err := google.CredentialsFromJSON(ctx, f.credentialsJSON,
		"https://www.googleapis.com/auth/firebase.messaging")
	if err != nil {
		return "", fmt.Errorf("failed to parse credentials: %w", err)
	}

	token, err := credentials.TokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	f.accessToken = token.AccessToken
	f.tokenExpiry = token.Expiry

	return f.accessToken, nil
}

// SendToDevice 发送通知到单个设备
func (f *FCMService) SendToDevice(ctx context.Context, deviceToken string, payload *domain.PushNotificationPayload) (*domain.PushNotificationResult, error) {
	if !f.enabled {
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     false,
			Error:       "FCM service not enabled",
		}, nil
	}

	// 构建 FCM 消息
	msg := &FCMMessage{}
	msg.Message.Token = deviceToken
	msg.Message.Notification = &FCMNotification{
		Title: payload.Title,
		Body:  payload.Body,
	}
	msg.Message.Data = payload.Data

	// Android 特定配置
	msg.Message.Android = &FCMAndroidConfig{
		Priority: "high",
		Notification: &FCMAndroidNotification{
			Sound:        payload.Sound,
			ChannelID:    payload.Category,
			DefaultSound: payload.Sound == "" || payload.Sound == "default",
		},
	}

	return f.sendMessage(ctx, msg)
}

// SendToTopic 发送通知到主题
func (f *FCMService) SendToTopic(ctx context.Context, topic string, payload *domain.PushNotificationPayload) (*domain.PushNotificationResult, error) {
	if !f.enabled {
		return &domain.PushNotificationResult{
			Success: false,
			Error:   "FCM service not enabled",
		}, nil
	}

	msg := &FCMMessage{}
	msg.Message.Topic = topic
	msg.Message.Notification = &FCMNotification{
		Title: payload.Title,
		Body:  payload.Body,
	}
	msg.Message.Data = payload.Data

	return f.sendMessage(ctx, msg)
}

// sendMessage 发送 FCM 消息
func (f *FCMService) sendMessage(ctx context.Context, msg *FCMMessage) (*domain.PushNotificationResult, error) {
	startTime := time.Now()

	accessToken, err := f.getAccessToken(ctx)
	if err != nil {
		f.logger.Error("failed to get FCM access token", zap.Error(err))
		// 记录通知发送错误
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordNotificationError("push", "fcm", "network")
		}
		return &domain.PushNotificationResult{
			DeviceToken: msg.Message.Token,
			Success:     false,
			Error:       err.Error(),
		}, err
	}

	// 构建请求
	url := fmt.Sprintf("https://fcm.googleapis.com/v1/projects/%s/messages:send", f.projectID)
	body, err := json.Marshal(msg)
	if err != nil {
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordNotificationError("push", "fcm", "payload_too_large")
		}
		return nil, fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		f.logger.Error("failed to send FCM request", zap.Error(err))
		// 记录通知发送失败
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordNotificationSent("push", "fcm", "failed", time.Since(startTime))
			metrics.RecordNotificationError("push", "fcm", "network")
		}
		return &domain.PushNotificationResult{
			DeviceToken: msg.Message.Token,
			Success:     false,
			Error:       err.Error(),
		}, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var fcmResp FCMResponse
	if err := json.Unmarshal(respBody, &fcmResp); err != nil {
		f.logger.Error("failed to parse FCM response", zap.Error(err), zap.String("body", string(respBody)))
	}

	if resp.StatusCode != http.StatusOK {
		errMsg := string(respBody)
		if fcmResp.Error != nil {
			errMsg = fcmResp.Error.Message
		}
		f.logger.Error("FCM request failed",
			zap.Int("status", resp.StatusCode),
			zap.String("error", errMsg))
		// 记录通知发送失败
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordNotificationSent("push", "fcm", "failed", time.Since(startTime))
			// 根据错误类型分类记录
			errorType := "unknown"
			if fcmResp.Error != nil {
				switch fcmResp.Error.Status {
				case "UNREGISTERED", "INVALID_ARGUMENT":
					errorType = "invalid_token"
				case "QUOTA_EXCEEDED":
					errorType = "rate_limit"
				}
			}
			metrics.RecordNotificationError("push", "fcm", errorType)
		}
		return &domain.PushNotificationResult{
			DeviceToken: msg.Message.Token,
			Success:     false,
			Error:       errMsg,
		}, fmt.Errorf("FCM request failed: %s", errMsg)
	}

	f.logger.Debug("FCM message sent",
		zap.String("messageId", fcmResp.Name),
		zap.String("token", msg.Message.Token))

	// 记录通知发送成功
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.RecordNotificationSent("push", "fcm", "success", time.Since(startTime))
	}

	return &domain.PushNotificationResult{
		DeviceToken: msg.Message.Token,
		Success:     true,
		MessageID:   fcmResp.Name,
	}, nil
}

// SendBatch 批量发送通知
func (f *FCMService) SendBatch(ctx context.Context, deviceTokens []string, payload *domain.PushNotificationPayload) []*domain.PushNotificationResult {
	results := make([]*domain.PushNotificationResult, len(deviceTokens))

	// 并发发送，但限制并发数
	sem := make(chan struct{}, 10) // 最多 10 个并发
	var wg sync.WaitGroup

	for i, token := range deviceTokens {
		wg.Add(1)
		go func(idx int, deviceToken string) {
			defer wg.Done()
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			result, _ := f.SendToDevice(ctx, deviceToken, payload)
			results[idx] = result
		}(i, token)
	}

	wg.Wait()
	return results
}

// ========== Service 结构体扩展 ==========

var fcmService *FCMService

// SetFCMService 设置 FCM 服务
func (s *Service) SetFCMService(fcm *FCMService) {
	fcmService = fcm
}

// fcm 获取 FCM 服务实例
func (s *Service) fcm() *FCMService {
	return fcmService
}

// SendNotificationToFCM 发送通知到 Android 设备
func (s *Service) SendNotificationToFCM(ctx context.Context, userID string, notification *domain.Notification) error {
	if s.fcm() == nil || !s.fcm().IsEnabled() {
		s.logger.Debug("FCM not enabled, skipping push notification")
		return nil
	}

	// 获取用户的 Android 设备
	devices, err := s.repo.GetUserDevicesByPlatform(ctx, userID, domain.PlatformAndroid)
	if err != nil {
		s.logger.Error("failed to get user Android devices", zap.Error(err))
		return err
	}

	if len(devices) == 0 {
		return nil
	}

	badge, uerr := s.repo.UnreadNotificationCount(ctx, userID)
	if uerr != nil {
		s.logger.Debug("unread count for FCM badge unavailable", zap.Error(uerr))
		badge = 0
	}
	data := pushNotificationDataMap(notification)
	if badge > 0 {
		data["badge"] = fmt.Sprintf("%d", badge)
	}

	payload := &domain.PushNotificationPayload{
		Title:    notification.Title,
		Body:     notificationPushBody(notification),
		Sound:    "default",
		Category: notification.Type,
		Badge:    badge,
		Data:     data,
	}

	// 发送到每个设备
	for _, device := range devices {
		if !device.IsActive {
			continue
		}
		result, _ := s.fcm().SendToDevice(ctx, device.DeviceToken, payload)
		if result != nil && !result.Success {
			s.logger.Warn("FCM notification failed",
				zap.String("deviceId", device.ID),
				zap.String("error", result.Error))

			// 如果是无效 token，标记设备为非活跃
			if result.Error == "UNREGISTERED" || result.Error == "INVALID_ARGUMENT" {
				_ = s.repo.DeactivateUserDevice(ctx, device.DeviceToken)
			}
		}
	}

	return nil
}

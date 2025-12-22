package service

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// APNsService Apple Push Notification Service
type APNsService struct {
	bundleID       string
	teamID         string
	keyID          string
	privateKey     *ecdsa.PrivateKey
	httpClient     *http.Client
	authToken      string
	tokenExpiry    time.Time
	tokenMutex     sync.RWMutex
	useSandbox     bool
	enabled        bool
	logger         *zap.Logger
}

// APNsConfig APNs 配置
type APNsConfig struct {
	BundleID   string // App Bundle ID
	TeamID     string // Apple Developer Team ID
	KeyID      string // APNs Auth Key ID
	PrivateKey string // APNs Auth Key (.p8 file content)
	UseSandbox bool   // 是否使用 Sandbox 环境
}

// NewAPNsService 创建 APNs 服务实例
func NewAPNsService(config *APNsConfig, logger *zap.Logger) *APNsService {
	svc := &APNsService{
		bundleID:   config.BundleID,
		teamID:     config.TeamID,
		keyID:      config.KeyID,
		useSandbox: config.UseSandbox,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled: false,
		logger:  logger,
	}

	// 解析私钥
	if config.PrivateKey != "" {
		key, err := parseAPNsPrivateKey(config.PrivateKey)
		if err != nil {
			logger.Error("failed to parse APNs private key", zap.Error(err))
		} else {
			svc.privateKey = key
		}
	}

	// 验证配置
	if config.BundleID != "" && config.TeamID != "" && config.KeyID != "" && svc.privateKey != nil {
		svc.enabled = true
		logger.Info("APNs service initialized",
			zap.String("bundleId", config.BundleID),
			zap.Bool("sandbox", config.UseSandbox))
	} else {
		logger.Warn("APNs service not fully configured, push notifications disabled")
	}

	return svc
}

// parseAPNsPrivateKey 解析 APNs 私钥
func parseAPNsPrivateKey(pemData string) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not an ECDSA private key")
	}

	return ecdsaKey, nil
}

// IsEnabled 检查 APNs 服务是否启用
func (a *APNsService) IsEnabled() bool {
	return a.enabled
}

// APNsPayload Apple Push Notification 载荷
type APNsPayload struct {
	Aps APNsAps `json:"aps"`
	// 自定义数据
	NotificationID string `json:"notificationId,omitempty"`
	Type           string `json:"type,omitempty"`
	TargetID       string `json:"targetId,omitempty"`
	Link           string `json:"link,omitempty"`
}

// APNsAps APNs标准载荷
type APNsAps struct {
	Alert            interface{} `json:"alert,omitempty"`
	Badge            *int        `json:"badge,omitempty"`
	Sound            string      `json:"sound,omitempty"`
	ContentAvailable int         `json:"content-available,omitempty"`
	Category         string      `json:"category,omitempty"`
	ThreadID         string      `json:"thread-id,omitempty"`
	MutableContent   int         `json:"mutable-content,omitempty"`
}

// APNsAlert 通知提醒内容
type APNsAlert struct {
	Title        string   `json:"title,omitempty"`
	Subtitle     string   `json:"subtitle,omitempty"`
	Body         string   `json:"body,omitempty"`
	LaunchImage  string   `json:"launch-image,omitempty"`
	TitleLocKey  string   `json:"title-loc-key,omitempty"`
	TitleLocArgs []string `json:"title-loc-args,omitempty"`
	LocKey       string   `json:"loc-key,omitempty"`
	LocArgs      []string `json:"loc-args,omitempty"`
}

// getAuthToken 获取 APNs JWT 认证令牌
func (a *APNsService) getAuthToken() (string, error) {
	a.tokenMutex.RLock()
	if a.authToken != "" && time.Now().Before(a.tokenExpiry.Add(-5*time.Minute)) {
		token := a.authToken
		a.tokenMutex.RUnlock()
		return token, nil
	}
	a.tokenMutex.RUnlock()

	a.tokenMutex.Lock()
	defer a.tokenMutex.Unlock()

	// 双重检查
	if a.authToken != "" && time.Now().Before(a.tokenExpiry.Add(-5*time.Minute)) {
		return a.authToken, nil
	}

	// 生成新的 JWT
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": a.teamID,
		"iat": now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token.Header["kid"] = a.keyID

	signedToken, err := token.SignedString(a.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	a.authToken = signedToken
	a.tokenExpiry = now.Add(50 * time.Minute) // Apple 建议每 60 分钟刷新一次

	return a.authToken, nil
}

// getAPNsURL 获取 APNs 服务器 URL
func (a *APNsService) getAPNsURL(deviceToken string) string {
	if a.useSandbox {
		return fmt.Sprintf("https://api.sandbox.push.apple.com/3/device/%s", deviceToken)
	}
	return fmt.Sprintf("https://api.push.apple.com/3/device/%s", deviceToken)
}

// SendToDevice 发送通知到单个设备
func (a *APNsService) SendToDevice(ctx context.Context, deviceToken string, payload *domain.PushNotificationPayload) (*domain.PushNotificationResult, error) {
	if !a.enabled {
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     false,
			Error:       "APNs service not enabled",
		}, nil
	}

	// 构建 APNs 载荷
	apnsPayload := &APNsPayload{
		Aps: APNsAps{
			Alert: APNsAlert{
				Title: payload.Title,
				Body:  payload.Body,
			},
			Sound:          payload.Sound,
			Category:       payload.Category,
			ThreadID:       payload.ThreadID,
			MutableContent: 1,
		},
	}

	if payload.Badge > 0 {
		apnsPayload.Aps.Badge = &payload.Badge
	}

	// 添加自定义数据
	if payload.Data != nil {
		if v, ok := payload.Data["notificationId"]; ok {
			apnsPayload.NotificationID = v
		}
		if v, ok := payload.Data["type"]; ok {
			apnsPayload.Type = v
		}
		if v, ok := payload.Data["link"]; ok {
			apnsPayload.Link = v
		}
	}

	return a.sendNotification(ctx, deviceToken, apnsPayload)
}

// sendNotification 发送 APNs 通知
func (a *APNsService) sendNotification(ctx context.Context, deviceToken string, payload *APNsPayload) (*domain.PushNotificationResult, error) {
	authToken, err := a.getAuthToken()
	if err != nil {
		a.logger.Error("failed to get APNs auth token", zap.Error(err))
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     false,
			Error:       err.Error(),
		}, err
	}

	// 序列化载荷
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 创建请求
	url := a.getAPNsURL(deviceToken)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apns-topic", a.bundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("apns-expiration", "0")

	// 发送请求
	resp, err := a.httpClient.Do(req)
	if err != nil {
		a.logger.Error("failed to send APNs request", zap.Error(err))
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     false,
			Error:       err.Error(),
		}, err
	}
	defer resp.Body.Close()

	// 获取 apns-id
	apnsID := resp.Header.Get("apns-id")

	if resp.StatusCode == http.StatusOK {
		a.logger.Debug("APNs notification sent",
			zap.String("apnsId", apnsID),
			zap.String("token", deviceToken[:min(16, len(deviceToken))]+"..."))
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     true,
			MessageID:   apnsID,
		}, nil
	}

	// 解析错误响应
	respBody, _ := io.ReadAll(resp.Body)
	var errorResp struct {
		Reason    string `json:"reason"`
		Timestamp int64  `json:"timestamp,omitempty"`
	}
	_ = json.Unmarshal(respBody, &errorResp)

	a.logger.Error("APNs request failed",
		zap.Int("status", resp.StatusCode),
		zap.String("reason", errorResp.Reason))

	return &domain.PushNotificationResult{
		DeviceToken: deviceToken,
		Success:     false,
		Error:       errorResp.Reason,
	}, fmt.Errorf("APNs request failed: %s", errorResp.Reason)
}

// SendBatch 批量发送通知
func (a *APNsService) SendBatch(ctx context.Context, deviceTokens []string, payload *domain.PushNotificationPayload) []*domain.PushNotificationResult {
	results := make([]*domain.PushNotificationResult, len(deviceTokens))

	// 并发发送，但限制并发数
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	for i, token := range deviceTokens {
		wg.Add(1)
		go func(idx int, deviceToken string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, _ := a.SendToDevice(ctx, deviceToken, payload)
			results[idx] = result
		}(i, token)
	}

	wg.Wait()
	return results
}

// SendSilentNotification 发送静默通知（后台更新）
func (a *APNsService) SendSilentNotification(ctx context.Context, deviceToken string, data map[string]string) (*domain.PushNotificationResult, error) {
	if !a.enabled {
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     false,
			Error:       "APNs service not enabled",
		}, nil
	}

	payload := &APNsPayload{
		Aps: APNsAps{
			ContentAvailable: 1,
		},
	}

	if data != nil {
		if v, ok := data["notificationId"]; ok {
			payload.NotificationID = v
		}
		if v, ok := data["type"]; ok {
			payload.Type = v
		}
	}

	return a.sendSilentNotification(ctx, deviceToken, payload)
}

// sendSilentNotification 发送静默通知
func (a *APNsService) sendSilentNotification(ctx context.Context, deviceToken string, payload *APNsPayload) (*domain.PushNotificationResult, error) {
	authToken, err := a.getAuthToken()
	if err != nil {
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     false,
			Error:       err.Error(),
		}, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	url := a.getAPNsURL(deviceToken)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apns-topic", a.bundleID)
	req.Header.Set("apns-push-type", "background")
	req.Header.Set("apns-priority", "5")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     false,
			Error:       err.Error(),
		}, err
	}
	defer resp.Body.Close()

	apnsID := resp.Header.Get("apns-id")

	if resp.StatusCode == http.StatusOK {
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     true,
			MessageID:   apnsID,
		}, nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return &domain.PushNotificationResult{
		DeviceToken: deviceToken,
		Success:     false,
		Error:       string(respBody),
	}, nil
}

// UpdateBadge 更新应用徽章
func (a *APNsService) UpdateBadge(ctx context.Context, deviceToken string, badge int) (*domain.PushNotificationResult, error) {
	if !a.enabled {
		return &domain.PushNotificationResult{
			DeviceToken: deviceToken,
			Success:     false,
			Error:       "APNs service not enabled",
		}, nil
	}

	payload := &APNsPayload{
		Aps: APNsAps{
			Badge:            &badge,
			ContentAvailable: 1,
		},
	}

	return a.sendSilentNotification(ctx, deviceToken, payload)
}

// ========== Service 结构体扩展 ==========

var apnsService *APNsService

// SetAPNsService 设置 APNs 服务
func (s *Service) SetAPNsService(apns *APNsService) {
	apnsService = apns
}

// apns 获取 APNs 服务实例
func (s *Service) apns() *APNsService {
	return apnsService
}

// SendNotificationToAPNs 发送通知到 Apple 设备
func (s *Service) SendNotificationToAPNs(ctx context.Context, userID string, notification *domain.Notification) error {
	if s.apns() == nil || !s.apns().IsEnabled() {
		s.logger.Debug("APNs not enabled, skipping push notification")
		return nil
	}

	// 获取用户的 iOS 设备
	devices, err := s.repo.GetActiveUserDevicesByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user devices", zap.Error(err))
		return err
	}

	// 过滤 Apple 设备
	appleDevices := make([]*domain.UserDevice, 0)
	for _, d := range devices {
		if d.PushProvider == "apns" {
			appleDevices = append(appleDevices, d)
		}
	}

	if len(appleDevices) == 0 {
		return nil
	}

	// 构建推送载荷
	payload := &domain.PushNotificationPayload{
		Title:    notification.Title,
		Body:     notification.Content,
		Sound:    "default",
		Category: notification.Type,
		Data: map[string]string{
			"notificationId": notification.ID,
			"type":           notification.Type,
			"link":           notification.Link,
		},
	}

	// 发送到每个设备
	for _, device := range appleDevices {
		result, _ := s.apns().SendToDevice(ctx, device.DeviceToken, payload)
		if result != nil && !result.Success {
			s.logger.Warn("APNs notification failed",
				zap.String("deviceId", device.ID),
				zap.String("error", result.Error))

			// 如果是无效 token，标记设备为非活跃
			if result.Error == "BadDeviceToken" || result.Error == "Unregistered" {
				_ = s.repo.DeactivateUserDevice(ctx, device.DeviceToken)
			}
		}
	}

	return nil
}

// RegisterDeviceToken 注册设备token（完整实现）
func (s *Service) RegisterDeviceToken(ctx context.Context, userID, deviceToken, platform string) error {
	now := time.Now().Unix()

	// 确定推送提供商
	pushProvider := "apns"
	if platform == "android" {
		pushProvider = "fcm"
	}

	device := &domain.UserDevice{
		UserID:       userID,
		DeviceToken:  deviceToken,
		Platform:     domain.DevicePlatform(platform),
		PushProvider: pushProvider,
		IsActive:     true,
		LastActiveAt: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	// 检查是否已存在
	existing, err := s.repo.GetUserDeviceByToken(ctx, deviceToken)
	if err == nil && existing != nil {
		// 更新现有设备
		existing.UserID = userID
		existing.Platform = domain.DevicePlatform(platform)
		existing.PushProvider = pushProvider
		existing.IsActive = true
		existing.LastActiveAt = now
		existing.UpdatedAt = now
		return s.repo.UpdateUserDevice(ctx, existing)
	}

	// 创建新设备
	device.ID = generateDeviceID()
	return s.repo.CreateUserDevice(ctx, device)
}

// UnregisterDeviceToken 注销设备token
func (s *Service) UnregisterDeviceToken(ctx context.Context, userID, deviceToken string) error {
	return s.repo.DeleteUserDeviceByToken(ctx, deviceToken)
}

// UpdateDeviceTokenBadge 更新设备徽章数
func (s *Service) UpdateDeviceTokenBadge(ctx context.Context, userID string, count int) error {
	if s.apns() == nil || !s.apns().IsEnabled() {
		return nil
	}

	devices, err := s.repo.GetActiveUserDevicesByUserID(ctx, userID)
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.PushProvider == "apns" {
			_, _ = s.apns().UpdateBadge(ctx, device.DeviceToken, count)
		}
	}

	return nil
}

// generateDeviceID 生成唯一设备 ID
func generateDeviceID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ========== 通知发送时自动推送 ==========

// CreateNotificationWithPush 创建通知并推送到所有设备
func (s *Service) CreateNotificationWithPush(ctx context.Context, notification *domain.Notification) error {
	// 先保存到数据库
	if err := s.CreateNotification(ctx, notification); err != nil {
		return err
	}

	// 异步推送到所有设备
	go func() {
		bgCtx := context.Background()
		
		// 推送到 Apple 设备
		if err := s.SendNotificationToAPNs(bgCtx, notification.UserID, notification); err != nil {
			s.logger.Error("failed to send APNs notification",
				zap.Error(err),
				zap.String("notificationId", notification.ID))
		}

		// 推送到 Android 设备
		if err := s.SendNotificationToFCM(bgCtx, notification.UserID, notification); err != nil {
			s.logger.Error("failed to send FCM notification",
				zap.Error(err),
				zap.String("notificationId", notification.ID))
		}
	}()

	return nil
}

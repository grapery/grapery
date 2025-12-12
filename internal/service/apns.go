package service

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// APNsService Apple Push Notification Service
type APNsService struct {
	// TODO: 集成实际的 APNs SDK
	// 例如: github.com/sideshow/apns2
	enabled bool
	logger  *zap.Logger
}

// NewAPNsService 创建 APNs 服务实例
func NewAPNsService(logger *zap.Logger) *APNsService {
	return &APNsService{
		enabled: false, // 默认关闭，需要配置后启用
		logger:  logger,
	}
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
	Badge            int         `json:"badge,omitempty"`
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

// SendNotificationToAPNs 发送通知到 Apple 设备
func (s *Service) SendNotificationToAPNs(ctx context.Context, userID string, notification *domain.Notification) error {
	if s.apns() == nil || !s.apns().enabled {
		s.logger.Debug("APNs not enabled, skipping push notification")
		return nil
	}

	// 获取用户的设备 token（需要从数据库获取）
	deviceTokens, err := s.getUserDeviceTokens(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user device tokens", zap.Error(err))
		return err
	}

	if len(deviceTokens) == 0 {
		return nil // 用户没有注册设备
	}

	// 构建 APNs 载荷
	payload := buildAPNsPayload(notification)

	// 发送到每个设备
	for _, token := range deviceTokens {
		if err := s.apns().sendPushNotification(token, payload); err != nil {
			s.logger.Error("failed to send APNs notification",
				zap.Error(err),
				zap.String("userId", userID),
				zap.String("deviceToken", token))
		}
	}

	return nil
}

// sendPushNotification 发送推送通知到指定设备
func (a *APNsService) sendPushNotification(deviceToken string, payload *APNsPayload) error {
	if !a.enabled {
		return nil
	}

	// TODO: 使用实际的 APNs SDK 发送
	// 示例：
	// notification := &apns2.Notification{
	//     DeviceToken: deviceToken,
	//     Topic:       "com.grapery.app",
	//     Payload:     payload,
	// }
	// res, err := a.client.Push(notification)

	a.logger.Info("APNs notification sent",
		zap.String("deviceToken", deviceToken),
		zap.String("payload", fmt.Sprintf("%+v", payload)))

	return nil
}

// buildAPNsPayload 构建 APNs 载荷
func buildAPNsPayload(notification *domain.Notification) *APNsPayload {
	alert := APNsAlert{
		Title: notification.Title,
		Body:  notification.Content,
	}

	// 根据通知类型设置分类
	category := "DEFAULT"
	switch notification.Type {
	case "like":
		category = "LIKE"
	case "comment":
		category = "COMMENT"
	case "follow":
		category = "FOLLOW"
	case "mention":
		category = "MENTION"
	}

	return &APNsPayload{
		Aps: APNsAps{
			Alert:            alert,
			Badge:            1, // 可以根据未读数量动态设置
			Sound:            "default",
			Category:         category,
			ContentAvailable: 1,
			MutableContent:   1,
		},
		NotificationID: notification.ID,
		Type:           notification.Type,
		Link:           notification.Link,
	}
}

// getUserDeviceTokens 获取用户的设备token列表
func (s *Service) getUserDeviceTokens(ctx context.Context, userID string) ([]string, error) {
	// TODO: 从数据库获取用户注册的设备token
	// 这需要一个新的表来存储用户设备信息
	return []string{}, nil
}

// RegisterDeviceToken 注册设备token
func (s *Service) RegisterDeviceToken(ctx context.Context, userID, deviceToken, platform string) error {
	// TODO: 保存设备token到数据库
	// 表结构：
	// - user_id
	// - device_token
	// - platform (ios, macos, ipados)
	// - app_version
	// - created_at
	// - updated_at

	s.logger.Info("device token registered",
		zap.String("userId", userID),
		zap.String("platform", platform))

	return nil
}

// UnregisterDeviceToken 注销设备token
func (s *Service) UnregisterDeviceToken(ctx context.Context, userID, deviceToken string) error {
	// TODO: 从数据库删除设备token

	s.logger.Info("device token unregistered",
		zap.String("userId", userID))

	return nil
}

// UpdateDeviceTokenBadge 更新设备徽章数
func (s *Service) UpdateDeviceTokenBadge(ctx context.Context, userID string, count int) error {
	if s.apns() == nil || !s.apns().enabled {
		return nil
	}

	deviceTokens, err := s.getUserDeviceTokens(ctx, userID)
	if err != nil {
		return err
	}

	// 发送静默通知更新徽章
	for _, token := range deviceTokens {
		payload := &APNsPayload{
			Aps: APNsAps{
				Badge:            count,
				ContentAvailable: 1,
			},
		}
		_ = s.apns().sendPushNotification(token, payload)
	}

	return nil
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

// ========== 通知发送时自动推送到 APNs ==========

// CreateNotificationWithPush 创建通知并推送到APNs
func (s *Service) CreateNotificationWithPush(ctx context.Context, notification *domain.Notification) error {
	// 先保存到数据库
	if err := s.CreateNotification(ctx, notification); err != nil {
		return err
	}

	// 异步推送到 APNs
	go func() {
		if err := s.SendNotificationToAPNs(context.Background(), notification.UserID, notification); err != nil {
			s.logger.Error("failed to send APNs notification",
				zap.Error(err),
				zap.String("notificationId", notification.ID))
		}
	}()

	return nil
}

// ========== 集成示例 ==========

/*
使用 APNs 的完整示例：

1. 安装依赖:
   go get github.com/sideshow/apns2

2. 初始化 APNs 服务:
   apnsService := service.NewAPNsService(logger)
   apnsService.enabled = true
   // 配置证书或Token认证
   svc.SetAPNsService(apnsService)

3. 注册设备:
   POST /api/devices/register
   {
     "deviceToken": "xxx",
     "platform": "ios"
   }

4. 发送通知时自动推送:
   svc.CreateNotificationWithPush(ctx, notification)

5. 客户端处理:
   - 在 AppDelegate 中处理推送通知
   - 实现 UNUserNotificationCenterDelegate
   - 根据通知类型跳转到相应页面
*/

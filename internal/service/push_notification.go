package service

import (
	"context"
	"sync"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// PushNotificationService 统一推送通知服务
type PushNotificationService struct {
	apns   *APNsService
	fcm    *FCMService
	repo   domain.Repository
	logger *zap.Logger
}

// NewPushNotificationService 创建统一推送通知服务
func NewPushNotificationService(apns *APNsService, fcm *FCMService, repo domain.Repository, logger *zap.Logger) *PushNotificationService {
	return &PushNotificationService{
		apns:   apns,
		fcm:    fcm,
		repo:   repo,
		logger: logger,
	}
}

// SendToUser 发送推送通知到用户的所有设备
func (p *PushNotificationService) SendToUser(ctx context.Context, userID string, payload *domain.PushNotificationPayload) ([]*domain.PushNotificationResult, error) {
	devices, err := p.repo.GetActiveUserDevicesByUserID(ctx, userID)
	if err != nil {
		p.logger.Error("failed to get user devices", zap.Error(err), zap.String("userId", userID))
		return nil, err
	}

	if len(devices) == 0 {
		p.logger.Debug("no active devices for user", zap.String("userId", userID))
		return nil, nil
	}

	results := make([]*domain.PushNotificationResult, 0, len(devices))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, device := range devices {
		wg.Add(1)
		go func(d *domain.UserDevice) {
			defer wg.Done()

			var result *domain.PushNotificationResult
			var sendErr error

			switch d.PushProvider {
			case "apns":
				if p.apns != nil && p.apns.IsEnabled() {
					result, sendErr = p.apns.SendToDevice(ctx, d.DeviceToken, payload)
				}
			case "fcm":
				if p.fcm != nil && p.fcm.IsEnabled() {
					result, sendErr = p.fcm.SendToDevice(ctx, d.DeviceToken, payload)
				}
			default:
				p.logger.Warn("unknown push provider", zap.String("provider", d.PushProvider))
				return
			}

			if result != nil {
				result.DeviceID = d.ID
				mu.Lock()
				results = append(results, result)
				mu.Unlock()

				// 处理无效 token
				if !result.Success && isInvalidTokenError(result.Error) {
					_ = p.repo.DeactivateUserDevice(ctx, d.DeviceToken)
					p.logger.Info("deactivated invalid device",
						zap.String("deviceId", d.ID),
						zap.String("error", result.Error))
				}
			}

			if sendErr != nil {
				p.logger.Error("push notification failed",
					zap.Error(sendErr),
					zap.String("userId", userID),
					zap.String("deviceId", d.ID),
					zap.String("provider", d.PushProvider))
			}
		}(device)
	}

	wg.Wait()
	return results, nil
}

// SendToDevices 发送推送通知到指定设备列表
func (p *PushNotificationService) SendToDevices(ctx context.Context, devices []*domain.UserDevice, payload *domain.PushNotificationPayload) []*domain.PushNotificationResult {
	results := make([]*domain.PushNotificationResult, 0, len(devices))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, device := range devices {
		wg.Add(1)
		go func(d *domain.UserDevice) {
			defer wg.Done()

			var result *domain.PushNotificationResult

			switch d.PushProvider {
			case "apns":
				if p.apns != nil && p.apns.IsEnabled() {
					result, _ = p.apns.SendToDevice(ctx, d.DeviceToken, payload)
				}
			case "fcm":
				if p.fcm != nil && p.fcm.IsEnabled() {
					result, _ = p.fcm.SendToDevice(ctx, d.DeviceToken, payload)
				}
			}

			if result != nil {
				result.DeviceID = d.ID
				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			}
		}(device)
	}

	wg.Wait()
	return results
}

// SendNotification 发送应用内通知并推送
func (p *PushNotificationService) SendNotification(ctx context.Context, notification *domain.Notification) error {
	payload := &domain.PushNotificationPayload{
		Title:    notification.Title,
		Body:     notification.Content,
		Sound:    "default",
		Category: getCategoryForType(notification.Type),
		Data: map[string]string{
			"notificationId": notification.ID,
			"type":           notification.Type,
			"link":           notification.Link,
		},
	}

	_, err := p.SendToUser(ctx, notification.UserID, payload)
	return err
}

// SendToTopic 发送通知到主题（仅 FCM 支持）
func (p *PushNotificationService) SendToTopic(ctx context.Context, topic string, payload *domain.PushNotificationPayload) (*domain.PushNotificationResult, error) {
	if p.fcm == nil || !p.fcm.IsEnabled() {
		return &domain.PushNotificationResult{
			Success: false,
			Error:   "FCM not enabled",
		}, nil
	}

	return p.fcm.SendToTopic(ctx, topic, payload)
}

// UpdateUserBadge 更新用户所有设备的徽章数
func (p *PushNotificationService) UpdateUserBadge(ctx context.Context, userID string, badge int) error {
	devices, err := p.repo.GetActiveUserDevicesByUserID(ctx, userID)
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.PushProvider == "apns" && p.apns != nil && p.apns.IsEnabled() {
			_, _ = p.apns.UpdateBadge(ctx, device.DeviceToken, badge)
		}
	}

	return nil
}

// SendSilentNotification 发送静默通知（后台数据更新）
func (p *PushNotificationService) SendSilentNotification(ctx context.Context, userID string, data map[string]string) error {
	devices, err := p.repo.GetActiveUserDevicesByUserID(ctx, userID)
	if err != nil {
		return err
	}

	for _, device := range devices {
		if device.PushProvider == "apns" && p.apns != nil && p.apns.IsEnabled() {
			_, _ = p.apns.SendSilentNotification(ctx, device.DeviceToken, data)
		}
		// FCM 使用 data-only 消息
		if device.PushProvider == "fcm" && p.fcm != nil && p.fcm.IsEnabled() {
			payload := &domain.PushNotificationPayload{Data: data}
			_, _ = p.fcm.SendToDevice(ctx, device.DeviceToken, payload)
		}
	}

	return nil
}

// GetStatus 获取推送服务状态
func (p *PushNotificationService) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"apns": map[string]interface{}{
			"enabled": p.apns != nil && p.apns.IsEnabled(),
		},
		"fcm": map[string]interface{}{
			"enabled": p.fcm != nil && p.fcm.IsEnabled(),
		},
	}
}

// isInvalidTokenError 检查是否是无效 token 错误
func isInvalidTokenError(errMsg string) bool {
	invalidTokenErrors := []string{
		"BadDeviceToken",
		"Unregistered",
		"UNREGISTERED",
		"INVALID_ARGUMENT",
		"NotRegistered",
	}

	for _, e := range invalidTokenErrors {
		if errMsg == e {
			return true
		}
	}
	return false
}

// getCategoryForType 根据通知类型获取分类
func getCategoryForType(notificationType string) string {
	categories := map[string]string{
		"like":          "LIKE",
		"comment":       "COMMENT",
		"follow":        "FOLLOW",
		"mention":       "MENTION",
		"message":       "MESSAGE",
		"story_update":  "STORY_UPDATE",
		"system":        "SYSTEM",
		"subscription":  "SUBSCRIPTION",
	}

	if cat, ok := categories[notificationType]; ok {
		return cat
	}
	return "DEFAULT"
}

// ========== 便捷方法：发送特定类型的通知 ==========

// NotifyLike 发送点赞通知
func (p *PushNotificationService) NotifyLike(ctx context.Context, userID, likerName, targetTitle, link string) error {
	payload := &domain.PushNotificationPayload{
		Title:    "New Like",
		Body:     likerName + " liked your " + targetTitle,
		Sound:    "default",
		Category: "LIKE",
		Data: map[string]string{
			"type": "like",
			"link": link,
		},
	}
	_, err := p.SendToUser(ctx, userID, payload)
	return err
}

// NotifyComment 发送评论通知
func (p *PushNotificationService) NotifyComment(ctx context.Context, userID, commenterName, comment, link string) error {
	// 截断评论内容
	if len(comment) > 100 {
		comment = comment[:97] + "..."
	}

	payload := &domain.PushNotificationPayload{
		Title:    "New Comment",
		Body:     commenterName + ": " + comment,
		Sound:    "default",
		Category: "COMMENT",
		Data: map[string]string{
			"type": "comment",
			"link": link,
		},
	}
	_, err := p.SendToUser(ctx, userID, payload)
	return err
}

// NotifyFollow 发送关注通知
func (p *PushNotificationService) NotifyFollow(ctx context.Context, userID, followerName, link string) error {
	payload := &domain.PushNotificationPayload{
		Title:    "New Follower",
		Body:     followerName + " started following you",
		Sound:    "default",
		Category: "FOLLOW",
		Data: map[string]string{
			"type": "follow",
			"link": link,
		},
	}
	_, err := p.SendToUser(ctx, userID, payload)
	return err
}

// NotifyMention 发送提及通知
func (p *PushNotificationService) NotifyMention(ctx context.Context, userID, mentionerName, content, link string) error {
	if len(content) > 100 {
		content = content[:97] + "..."
	}

	payload := &domain.PushNotificationPayload{
		Title:    "You were mentioned",
		Body:     mentionerName + " mentioned you: " + content,
		Sound:    "default",
		Category: "MENTION",
		Data: map[string]string{
			"type": "mention",
			"link": link,
		},
	}
	_, err := p.SendToUser(ctx, userID, payload)
	return err
}

// NotifySubscriptionUpdate 发送订阅更新通知
func (p *PushNotificationService) NotifySubscriptionUpdate(ctx context.Context, userID, title, message string) error {
	payload := &domain.PushNotificationPayload{
		Title:    title,
		Body:     message,
		Sound:    "default",
		Category: "SUBSCRIPTION",
		Data: map[string]string{
			"type": "subscription",
		},
	}
	_, err := p.SendToUser(ctx, userID, payload)
	return err
}

// NotifySystem 发送系统通知
func (p *PushNotificationService) NotifySystem(ctx context.Context, userID, title, message string) error {
	payload := &domain.PushNotificationPayload{
		Title:    title,
		Body:     message,
		Sound:    "default",
		Category: "SYSTEM",
		Data: map[string]string{
			"type": "system",
		},
	}
	_, err := p.SendToUser(ctx, userID, payload)
	return err
}

// BroadcastToAllUsers 广播通知到所有用户（谨慎使用）
func (p *PushNotificationService) BroadcastToAllUsers(ctx context.Context, topic string, payload *domain.PushNotificationPayload) (*domain.PushNotificationResult, error) {
	// 通过 FCM Topic 广播
	return p.SendToTopic(ctx, topic, payload)
}


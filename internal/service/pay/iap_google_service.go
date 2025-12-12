package pay

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/sirupsen/logrus"
)

// GoogleIAPServiceRefactored Google IAP服务实现（重构后）
type GoogleIAPServiceRefactored struct {
	config     *IAPConfig
	httpClient *HTTPClientWrapper
	logger     *logrus.Logger
}

// NewGoogleIAPServiceRefactored 创建Google IAP服务
func NewGoogleIAPServiceRefactored(config *IAPConfig, logger *logrus.Logger) *GoogleIAPServiceRefactored {
	return &GoogleIAPServiceRefactored{
		config:     config,
		httpClient: NewHTTPClientWrapper(),
		logger:     logger,
	}
}

// GetPlatform 获取平台
func (s *GoogleIAPServiceRefactored) GetPlatform() IAPPlatform {
	return IAPPlatformGoogle
}

// VerifyReceipt 验证收据
func (s *GoogleIAPServiceRefactored) VerifyReceipt(ctx context.Context, receipt string, sandbox bool) (*IAPReceipt, error) {
	s.logger.WithFields(logrus.Fields{
		"receipt_length": len(receipt),
		"sandbox":        sandbox,
	}).Info("开始验证Google收据")

	// 创建Google Play Developer客户端
	googleClient := NewGooglePlayDeveloper(s.config.Google.GetPackageName(sandbox), s.config.Google.GetServiceAccountKey(sandbox))

	// 解析收据数据
	var receiptData map[string]interface{}
	if err := json.Unmarshal([]byte(receipt), &receiptData); err != nil {
		s.logger.WithError(err).Error("解析Google收据数据失败")
		return nil, fmt.Errorf("解析收据数据失败: %w", err)
	}

	// 提取必要信息
	purchaseToken, ok := receiptData["purchaseToken"].(string)
	if !ok {
		return nil, fmt.Errorf("收据中缺少purchaseToken")
	}

	productID, ok := receiptData["productId"].(string)
	if !ok {
		return nil, fmt.Errorf("收据中缺少productId")
	}

	// 验证购买
	purchaseInfo, err := googleClient.GetPurchaseInfo(ctx, purchaseToken, productID)
	if err != nil {
		s.logger.WithError(err).Error("验证Google购买失败")
		return nil, fmt.Errorf("验证购买失败: %w", err)
	}

	// 转换为统一格式
	iapReceipt := &IAPReceipt{
		Platform:       IAPPlatformGoogle,
		UserID:         0, // 需要从其他地方获取
		ReceiptData:    receipt,
		BundleID:       s.config.Google.PackageName,
		CreationDate:   time.Unix(purchaseInfo.PurchaseTime/1000, 0),
		ExpirationDate: nil, // 一次性购买没有过期时间
		Status:         s.mapGooglePaymentState(purchaseInfo.PurchaseState),
		Environment:    map[bool]string{true: "Sandbox", false: "Production"}[sandbox],
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	s.logger.WithFields(logrus.Fields{
		"product_id":     productID,
		"transaction_id": purchaseToken,
		"purchase_state": purchaseInfo.PurchaseState,
		"environment":    iapReceipt.Environment,
	}).Info("Google收据验证成功")

	return iapReceipt, nil
}

// GetSubscription 获取订阅信息
func (s *GoogleIAPServiceRefactored) GetSubscription(ctx context.Context, subscriptionID string) (*IAPSubscription, error) {
	s.logger.WithField("subscription_id", subscriptionID).Info("获取Google订阅信息")

	// 从数据库获取Google订阅信息
	var googleSub paymodels.GoogleSubscription
	err := paymodels.DataBase().WithContext(ctx).
		Where("purchase_token = ?", subscriptionID).
		First(&googleSub).Error

	if err != nil {
		s.logger.WithError(err).Error("获取Google订阅信息失败")
		return nil, fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 转换为统一格式
	iapSub := &IAPSubscription{
		ID:                     googleSub.ID,
		UserID:                 googleSub.UserID,
		Platform:               IAPPlatformGoogle,
		SubscriptionID:         googleSub.PurchaseToken,
		ProductID:              googleSub.ProductID,
		PurchaseDate:           googleSub.StartTime,
		ExpiresDate:            &googleSub.ExpiryTime,
		Status:                 s.mapGooglePaymentState(googleSub.PaymentState),
		AutoRenewStatus:        map[bool]string{true: "On", false: "Off"}[googleSub.AutoRenewing],
		IsInIntroOfferPeriod:   false, // Google需要单独处理
		IsInGracePeriod:        googleSub.GracePeriodEndTime != nil,
		GracePeriodExpiresDate: googleSub.GracePeriodEndTime,
		OfferType:              "", // Google需要单独处理
		PriceIncreaseStatus:    "", // Google需要单独处理
		LastNotificationType:   googleSub.LastNotificationType,
		LastNotificationDate:   googleSub.LastNotificationDate,
		CreatedAt:              googleSub.CreatedAt,
		UpdatedAt:              googleSub.UpdatedAt,
	}

	return iapSub, nil
}

// SyncSubscription 同步订阅
func (s *GoogleIAPServiceRefactored) SyncSubscription(ctx context.Context, subscriptionID string) error {
	s.logger.WithField("subscription_id", subscriptionID).Info("同步Google订阅")

	// 获取当前订阅信息
	subscription, err := s.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 从Google API获取最新信息
	googleClient := NewGooglePlayDeveloper(s.config.Google.GetPackageName(false), s.config.Google.GetServiceAccountKey(false))
	subInfo, err := googleClient.GetSubscriptionInfo(ctx, subscriptionID, subscription.ProductID)
	if err != nil {
		s.logger.WithError(err).Error("从Google API获取订阅信息失败")
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 更新数据库记录
	var googleSub paymodels.GoogleSubscription
	err = paymodels.DataBase().WithContext(ctx).
		Where("purchase_token = ?", subscriptionID).
		First(&googleSub).Error

	if err != nil {
		s.logger.WithError(err).Error("查找Google订阅记录失败")
		return fmt.Errorf("查找订阅记录失败: %w", err)
	}

	// 更新订阅信息
	googleSub.StartTime = time.Unix(subInfo.StartTimeMillis/1000, 0)
	googleSub.ExpiryTime = time.Unix(subInfo.ExpiryTimeMillis/1000, 0)
	googleSub.AutoRenewing = subInfo.AutoRenewing
	googleSub.PaymentState = subInfo.PaymentState
	googleSub.UpdatedAt = time.Now()

	if err := paymodels.DataBase().WithContext(ctx).Save(&googleSub).Error; err != nil {
		s.logger.WithError(err).Error("更新Google订阅记录失败")
		return fmt.Errorf("更新订阅记录失败: %w", err)
	}

	s.logger.WithField("subscription_id", subscriptionID).Info("Google订阅同步成功")
	return nil
}

// AcknowledgePurchase 确认购买
func (s *GoogleIAPServiceRefactored) AcknowledgePurchase(ctx context.Context, purchaseToken string) error {
	s.logger.WithField("purchase_token", purchaseToken).Info("确认Google购买")

	// 创建Google Play Developer客户端
	googleClient := NewGooglePlayDeveloper(s.config.Google.GetPackageName(false), s.config.Google.GetServiceAccountKey(false))

	// 确认购买
	err := googleClient.AcknowledgePurchase(ctx, purchaseToken)
	if err != nil {
		s.logger.WithError(err).Error("Google购买确认失败")
		return fmt.Errorf("购买确认失败: %w", err)
	}

	s.logger.WithField("purchase_token", purchaseToken).Info("Google购买确认成功")
	return nil
}

// ConsumePurchase 消费购买
func (s *GoogleIAPServiceRefactored) ConsumePurchase(ctx context.Context, purchaseToken string) error {
	s.logger.WithField("purchase_token", purchaseToken).Info("消费Google购买")

	// 创建Google Play Developer客户端
	googleClient := NewGooglePlayDeveloper(s.config.Google.GetPackageName(false), s.config.Google.GetServiceAccountKey(false))

	// 消费购买
	err := googleClient.ConsumePurchase(ctx, purchaseToken)
	if err != nil {
		s.logger.WithError(err).Error("Google购买消费失败")
		return fmt.Errorf("购买消费失败: %w", err)
	}

	s.logger.WithField("purchase_token", purchaseToken).Info("Google购买消费成功")
	return nil
}

// HandleNotification 处理通知
func (s *GoogleIAPServiceRefactored) HandleNotification(ctx context.Context, notification *IAPNotification) error {
	s.logger.WithFields(logrus.Fields{
		"notification_id":   notification.NotificationID,
		"notification_type": notification.NotificationType,
	}).Info("处理Google通知")

	// 解析Google通知数据
	notificationData, err := s.parseGoogleNotificationData(notification)
	if err != nil {
		s.logger.WithError(err).Error("解析Google通知数据失败")
		return fmt.Errorf("解析通知数据失败: %w", err)
	}

	// 根据通知类型处理
	switch notification.NotificationType {
	case "SUBSCRIPTION_RECOVERED":
		return s.handleSubscriptionRecovered(ctx, notification, notificationData)
	case "SUBSCRIPTION_RENEWED":
		return s.handleSubscriptionRenewed(ctx, notification, notificationData)
	case "SUBSCRIPTION_CANCELED":
		return s.handleSubscriptionCanceled(ctx, notification, notificationData)
	case "SUBSCRIPTION_PURCHASED":
		return s.handleSubscriptionPurchased(ctx, notification, notificationData)
	case "SUBSCRIPTION_ON_HOLD":
		return s.handleSubscriptionOnHold(ctx, notification, notificationData)
	case "SUBSCRIPTION_IN_GRACE_PERIOD":
		return s.handleSubscriptionInGracePeriod(ctx, notification, notificationData)
	case "SUBSCRIPTION_RESTARTED":
		return s.handleSubscriptionRestarted(ctx, notification, notificationData)
	case "SUBSCRIPTION_PRICE_CHANGE_CONFIRMED":
		return s.handleSubscriptionPriceChangeConfirmed(ctx, notification, notificationData)
	case "SUBSCRIPTION_DEFERRED":
		return s.handleSubscriptionDeferred(ctx, notification, notificationData)
	case "SUBSCRIPTION_PAUSED":
		return s.handleSubscriptionPaused(ctx, notification, notificationData)
	case "SUBSCRIPTION_PAUSE_SCHEDULE_CHANGED":
		return s.handleSubscriptionPauseScheduleChanged(ctx, notification, notificationData)
	case "SUBSCRIPTION_REVOKED":
		return s.handleSubscriptionRevoked(ctx, notification, notificationData)
	case "SUBSCRIPTION_EXPIRED":
		return s.handleSubscriptionExpired(ctx, notification, notificationData)
	default:
		s.logger.WithField("notification_type", notification.NotificationType).Warn("未知的Google通知类型")
		return nil
	}
}

// SyncProducts 同步产品
func (s *GoogleIAPServiceRefactored) SyncProducts(ctx context.Context) error {
	s.logger.Info("开始同步Google产品")

	// 创建产品服务
	productService := NewIAPProductService()

	// 同步Google产品
	err := productService.SyncProductsFromGoogle(ctx)
	if err != nil {
		s.logger.WithError(err).Error("同步Google产品失败")
		return fmt.Errorf("同步Google产品失败: %w", err)
	}

	s.logger.Info("Google产品同步成功")
	return nil
}

// 辅助方法
func (s *GoogleIAPServiceRefactored) mapGooglePaymentState(paymentState int) string {
	switch paymentState {
	case 0:
		return "Pending"
	case 1:
		return "Active"
	case 2:
		return "Trial"
	case 3:
		return "Deferred"
	default:
		return "Unknown"
	}
}

func (s *GoogleIAPServiceRefactored) convertGoogleTimestamp(timestamp int64) *time.Time {
	if timestamp == 0 {
		return nil
	}
	t := time.Unix(timestamp/1000, 0)
	return &t
}

// saveGoogleNotification 保存Google通知记录
func (s *GoogleIAPServiceRefactored) saveGoogleNotification(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData, status string) error {
	googleNotification := &paymodels.GoogleNotification{
		NotificationID:   notification.NotificationID,
		Version:          data.Version,
		NotificationType: data.NotificationType,
		EventTimeMillis:  data.EventTimeMillis,
		SubscriptionID:   data.SubscriptionNotification.SubscriptionID,
		PackageName:      s.config.Google.PackageName,
		EventTime:        time.Unix(data.EventTimeMillis/1000, 0),
		RawData:          notification.RawData,
		ProcessedAt:      time.Now(),
		Status:           status,
	}

	return paymodels.DataBase().WithContext(ctx).Create(googleNotification).Error
}

// updateGoogleSubscriptionFromAPI 从Google API更新订阅信息的通用方法
func (s *GoogleIAPServiceRefactored) updateGoogleSubscriptionFromAPI(ctx context.Context, subscriptionID string, notification *IAPNotification, data *GoogleNotificationData, updateFunc func(*paymodels.GoogleSubscription, *GoogleSubscriptionPurchase)) error {
	// 1. 从Google API获取最新订阅信息
	googleClient := NewGooglePlayDeveloper(s.config.Google.GetPackageName(false), s.config.Google.GetServiceAccountKey(false))
	subInfo, err := googleClient.GetSubscriptionInfo(ctx, subscriptionID, "")
	if err != nil {
		s.logger.WithError(err).Error("从Google API获取订阅信息失败")
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 2. 更新订阅记录
	var googleSub paymodels.GoogleSubscription
	err = paymodels.DataBase().WithContext(ctx).
		Where("purchase_token = ?", subscriptionID).
		First(&googleSub).Error

	if err != nil {
		s.logger.WithError(err).Error("查找Google订阅记录失败")
		return fmt.Errorf("查找订阅记录失败: %w", err)
	}

	// 3. 应用更新函数
	updateFunc(&googleSub, subInfo)

	// 4. 更新通用字段
	googleSub.LastNotificationType = notification.NotificationType
	googleSub.LastNotificationDate = s.convertGoogleTimestamp(data.EventTimeMillis)
	googleSub.UpdatedAt = time.Now()

	// 5. 保存到数据库
	if err := paymodels.DataBase().WithContext(ctx).Save(&googleSub).Error; err != nil {
		s.logger.WithError(err).Error("更新Google订阅记录失败")
		return fmt.Errorf("更新订阅记录失败: %w", err)
	}

	return nil
}

func (s *GoogleIAPServiceRefactored) parseGoogleNotificationData(notification *IAPNotification) (*GoogleNotificationData, error) {
	s.logger.WithFields(logrus.Fields{
		"notification_id":   notification.NotificationID,
		"notification_type": notification.NotificationType,
		"raw_data_length":   len(notification.RawData),
	}).Debug("开始解析Google通知数据")

	// 1. 解析原始数据
	var notificationData GoogleNotificationData
	if notification.RawData != "" {
		if err := json.Unmarshal([]byte(notification.RawData), &notificationData); err != nil {
			s.logger.WithError(err).Error("解析Google通知原始数据失败")
			return nil, fmt.Errorf("解析通知原始数据失败: %w", err)
		}
	} else {
		// 如果没有原始数据，从通知对象中提取信息
		notificationData = GoogleNotificationData{
			Version:          notification.Version,
			NotificationType: notification.NotificationType,
			EventTimeMillis:  notification.EventTimeMillis,
		}

		// 设置订阅ID
		if notification.SubscriptionID != "" {
			notificationData.SubscriptionNotification.SubscriptionID = notification.SubscriptionID
		}
	}

	// 2. 验证必要字段
	if notificationData.NotificationType == "" {
		notificationData.NotificationType = notification.NotificationType
	}

	if notificationData.EventTimeMillis == 0 {
		notificationData.EventTimeMillis = notification.EventTimeMillis
	}

	// 3. 记录解析结果
	s.logger.WithFields(logrus.Fields{
		"version":           notificationData.Version,
		"notification_type": notificationData.NotificationType,
		"event_time_millis": notificationData.EventTimeMillis,
		"subscription_id":   notificationData.SubscriptionNotification.SubscriptionID,
	}).Debug("Google通知数据解析完成")

	return &notificationData, nil
}

// Google通知处理方法的实现
func (s *GoogleIAPServiceRefactored) handleSubscriptionRecovered(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅恢复通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录 - 恢复意味着订阅重新激活
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.AutoRenewing = subInfo.AutoRenewing
		googleSub.ExpiryTime = time.Unix(subInfo.ExpiryTimeMillis/1000, 0)
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅恢复通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionRenewed(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅续费通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.ExpiryTime = time.Unix(subInfo.ExpiryTimeMillis/1000, 0)
		googleSub.AutoRenewing = subInfo.AutoRenewing
		googleSub.PaymentState = subInfo.PaymentState
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅续费通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionCanceled(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅取消通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.AutoRenewing = subInfo.AutoRenewing
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.CancelReason = subInfo.CancelReason
		googleSub.UserCancellationTime = s.convertGoogleTimestamp(subInfo.UserCancellationTimeMillis)
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅取消通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionPurchased(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅购买通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 从Google API获取最新订阅信息
	googleClient := NewGooglePlayDeveloper(s.config.Google.GetPackageName(false), s.config.Google.GetServiceAccountKey(false))
	subInfo, err := googleClient.GetSubscriptionInfo(ctx, subscriptionID, "")
	if err != nil {
		s.logger.WithError(err).Error("从Google API获取订阅信息失败")
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 4. 查找或创建订阅记录
	var googleSub paymodels.GoogleSubscription
	err = paymodels.DataBase().WithContext(ctx).
		Where("purchase_token = ?", subscriptionID).
		First(&googleSub).Error

	if err != nil {
		// 订阅不存在，创建新记录
		googleSub = paymodels.GoogleSubscription{
			UserID:               0, // 需要从其他地方获取用户ID
			PurchaseToken:        subscriptionID,
			ProductID:            "", // 需要从其他地方获取
			PackageName:          s.config.Google.PackageName,
			StartTime:            time.Unix(subInfo.StartTimeMillis/1000, 0),
			ExpiryTime:           time.Unix(subInfo.ExpiryTimeMillis/1000, 0),
			AutoRenewing:         subInfo.AutoRenewing,
			PaymentState:         subInfo.PaymentState,
			LastNotificationType: notification.NotificationType,
			LastNotificationDate: s.convertGoogleTimestamp(data.EventTimeMillis),
		}

		if err := paymodels.DataBase().WithContext(ctx).Create(&googleSub).Error; err != nil {
			s.logger.WithError(err).Error("创建Google订阅记录失败")
			return fmt.Errorf("创建订阅记录失败: %w", err)
		}

		s.logger.WithField("subscription_id", subscriptionID).Info("创建新的Google订阅记录")
	} else {
		// 更新现有订阅记录
		googleSub.StartTime = time.Unix(subInfo.StartTimeMillis/1000, 0)
		googleSub.ExpiryTime = time.Unix(subInfo.ExpiryTimeMillis/1000, 0)
		googleSub.AutoRenewing = subInfo.AutoRenewing
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.LastNotificationType = notification.NotificationType
		googleSub.LastNotificationDate = s.convertGoogleTimestamp(data.EventTimeMillis)
		googleSub.UpdatedAt = time.Now()

		if err := paymodels.DataBase().WithContext(ctx).Save(&googleSub).Error; err != nil {
			s.logger.WithError(err).Error("更新Google订阅记录失败")
			return fmt.Errorf("更新订阅记录失败: %w", err)
		}

		s.logger.WithField("subscription_id", subscriptionID).Info("更新Google订阅记录")
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
		"payment_state":   subInfo.PaymentState,
		"auto_renewing":   subInfo.AutoRenewing,
	}).Info("Google订阅购买通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionOnHold(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅暂停通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.AccountHoldTime = s.convertGoogleTimestamp(subInfo.AccountHoldTimeMillis)
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅暂停通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionInGracePeriod(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅宽限期通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.GracePeriodEndTime = s.convertGoogleTimestamp(subInfo.GracePeriodEndTimeMillis)
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅宽限期通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionRestarted(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅重启通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.StartTime = time.Unix(subInfo.StartTimeMillis/1000, 0)
		googleSub.ExpiryTime = time.Unix(subInfo.ExpiryTimeMillis/1000, 0)
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.AutoRenewing = subInfo.AutoRenewing
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅重启通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionPriceChangeConfirmed(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅价格变更确认通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.AutoRenewing = subInfo.AutoRenewing
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅价格变更确认通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionDeferred(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅延迟通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.AutoRenewing = subInfo.AutoRenewing
		googleSub.RetryTime = s.convertGoogleTimestamp(subInfo.RetryTimeMillis)
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅延迟通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionPaused(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅暂停通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.AutoRenewing = subInfo.AutoRenewing
		googleSub.PauseStartTime = s.convertGoogleTimestamp(subInfo.PauseStartTimeMillis)
		googleSub.PauseDurationTime = s.convertGoogleTimestamp(subInfo.PauseDurationTimeMillis)
		googleSub.AutoResumeTime = s.convertGoogleTimestamp(subInfo.AutoResumeTimeMillis)
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅暂停通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionPauseScheduleChanged(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅暂停计划变更通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = subInfo.PaymentState
		googleSub.AutoRenewing = subInfo.AutoRenewing
		googleSub.PauseStartTime = s.convertGoogleTimestamp(subInfo.PauseStartTimeMillis)
		googleSub.PauseDurationTime = s.convertGoogleTimestamp(subInfo.PauseDurationTimeMillis)
		googleSub.AutoResumeTime = s.convertGoogleTimestamp(subInfo.AutoResumeTimeMillis)
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅暂停计划变更通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionRevoked(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅撤销通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = 0 // 设置为未支付状态
		googleSub.AutoRenewing = false
		googleSub.CancelReason = subInfo.CancelReason
		googleSub.UserCancellationTime = s.convertGoogleTimestamp(subInfo.UserCancellationTimeMillis)
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅撤销通知处理完成")

	return nil
}

func (s *GoogleIAPServiceRefactored) handleSubscriptionExpired(ctx context.Context, notification *IAPNotification, data *GoogleNotificationData) error {
	s.logger.WithFields(logrus.Fields{
		"subscription_id": data.SubscriptionNotification.SubscriptionID,
		"event_time":      time.Unix(data.EventTimeMillis/1000, 0),
	}).Info("处理Google订阅过期通知")

	// 1. 保存通知记录
	if err := s.saveGoogleNotification(ctx, notification, data, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Google通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 获取订阅信息
	subscriptionID := data.SubscriptionNotification.SubscriptionID
	if subscriptionID == "" {
		return fmt.Errorf("订阅ID为空")
	}

	// 3. 更新订阅记录
	if err := s.updateGoogleSubscriptionFromAPI(ctx, subscriptionID, notification, data, func(googleSub *paymodels.GoogleSubscription, subInfo *GoogleSubscriptionPurchase) {
		googleSub.PaymentState = 0 // 设置为未支付状态
		googleSub.AutoRenewing = false
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"subscription_id": subscriptionID,
	}).Info("Google订阅过期通知处理完成")

	return nil
}

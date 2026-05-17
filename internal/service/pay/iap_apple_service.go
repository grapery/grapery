package pay

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/grapestree/fgrapery/grapery/internal/telemetry"
	"github.com/sirupsen/logrus"
)

// AppleIAPService Apple IAP服务实现
type AppleIAPService struct {
	config     *IAPConfig
	httpClient *HTTPClientWrapper
	logger     *logrus.Logger
	keyCache   *ApplePublicKeyCache
}

// NewAppleIAPService 创建Apple IAP服务
func NewAppleIAPService(config *IAPConfig) *AppleIAPService {
	return &AppleIAPService{
		config:     config,
		httpClient: NewHTTPClientWrapper(),
		logger:     logrus.New(),
		keyCache: &ApplePublicKeyCache{
			Keys: make(map[string]*rsa.PublicKey),
		},
	}
}

// getUserIDFromContext 从上下文获取用户ID
func getUserIDFromContext(ctx context.Context) (int64, error) {
	userID, ok := ctx.Value("userID").(int64)
	if !ok {
		// 尝试其他类型转换
		if _, ok := ctx.Value("userID").(string); ok {
			// 如果存储为字符串，这里可以转换
			return 0, fmt.Errorf("userID is string type, needs conversion")
		}
		if uid, ok := ctx.Value("userID").(uint64); ok {
			return int64(uid), nil
		}
		return 0, fmt.Errorf("userID not found in context")
	}
	return userID, nil
}

// GetPlatform 获取平台
func (s *AppleIAPService) GetPlatform() IAPPlatform {
	return IAPPlatformApple
}

// VerifyReceipt 验证收据
func (s *AppleIAPService) VerifyReceipt(ctx context.Context, receipt string, sandbox bool) (*IAPReceipt, error) {
	startTime := time.Now()
	s.logger.WithFields(logrus.Fields{
		"sandbox":          sandbox,
		"receipt_length":   len(receipt),
		"bundle_id":        s.config.Apple.GetBundleID(sandbox),
		"verification_url": s.config.Apple.GetVerificationURL(sandbox),
	}).Info("验证Apple收据")

	// 1. 根据环境获取配置
	bundleID := s.config.Apple.GetBundleID(sandbox)
	issuerID := s.config.Apple.GetIssuerID(sandbox)
	keyID := s.config.Apple.GetKeyID(sandbox)
	privateKey := s.config.Apple.GetPrivateKey(sandbox)
	_ = s.config.Apple.GetVerificationURL(sandbox) // 验证URL配置正确性

	// 2. 创建Apple App Store Connect客户端
	appleClient := NewAppleAppStoreConnect(issuerID, keyID, privateKey)

	// 3. 验证收据（使用sandbox参数）
	resp, err := appleClient.VerifyReceipt(ctx, receipt, sandbox)
	if err != nil {
		s.logger.WithError(err).Error("Apple收据验证失败")
		// 记录验证失败的指标
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordPaymentVerify("apple", "failed", time.Since(startTime))
		}
		return nil, fmt.Errorf("收据验证失败: %w", err)
	}

	// 3. 检查验证状态
	if resp.Status != 0 {
		s.logger.WithField("status", resp.Status).Error("Apple收据验证失败")
		// 记录验证失败的指标
		if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
			metrics.RecordPaymentVerify("apple", "failed", time.Since(startTime))
		}
		return nil, fmt.Errorf("收据验证失败，状态码: %d", resp.Status)
	}

	// 4. 从上下文获取用户ID
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("无法从上下文获取用户ID")
		return nil, fmt.Errorf("无法获取用户ID: %w", err)
	}

	// 5. 构建返回对象
	iapReceipt := &IAPReceipt{
		UserID:             uint64(userID),
		Platform:           IAPPlatformApple,
		ReceiptData:        receipt,
		BundleID:           bundleID, // 使用配置中的Bundle ID
		ApplicationVersion: resp.Receipt.ApplicationVersion,
		Environment:        map[bool]string{true: "Sandbox", false: "Production"}[sandbox],
		Status:             "Valid",
		CreationDate:       time.Now(),
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	// 6. 完善收据解析逻辑（必须传入 latest_receipt_info，否则无法解析订阅 SKU / transaction_id）
	if err := s.parseReceiptData(resp, iapReceipt); err != nil {
		s.logger.WithError(err).Error("解析收据数据失败")
		return nil, fmt.Errorf("解析收据数据失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":     iapReceipt.UserID,
		"bundle_id":   iapReceipt.BundleID,
		"status":      iapReceipt.Status,
		"environment": iapReceipt.Environment,
	}).Info("Apple收据验证成功")

	// 记录验证成功的指标
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.RecordPaymentVerify("apple", "success", time.Since(startTime))
	}

	return iapReceipt, nil
}

// GetSubscription 获取订阅信息
func (s *AppleIAPService) GetSubscription(ctx context.Context, subscriptionID string) (*IAPSubscription, error) {
	s.logger.WithField("subscription_id", subscriptionID).Info("获取Apple订阅信息")

	// 从数据库获取Apple订阅信息
	var appleSub paymodels.AppleSubscription
	err := paymodels.DataBase().WithContext(ctx).
		Where("original_transaction_id = ?", subscriptionID).
		First(&appleSub).Error

	if err != nil {
		s.logger.WithError(err).Error("获取Apple订阅信息失败")
		return nil, fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 转换为统一格式
	iapSub := &IAPSubscription{
		ID:                     appleSub.ID,
		UserID:                 appleSub.UserID,
		Platform:               IAPPlatformApple,
		SubscriptionID:         appleSub.OriginalTransactionID,
		ProductID:              appleSub.ProductID,
		PurchaseDate:           appleSub.PurchaseDate,
		ExpiresDate:            appleSub.ExpiresDate,
		Status:                 appleSub.Status,
		AutoRenewStatus:        appleSub.AutoRenewStatus,
		IsInIntroOfferPeriod:   appleSub.IsInIntroOfferPeriod,
		IsInGracePeriod:        appleSub.IsInGracePeriod,
		GracePeriodExpiresDate: appleSub.GracePeriodExpiresDate,
		OfferType:              appleSub.OfferType,
		PriceIncreaseStatus:    appleSub.PriceIncreaseStatus,
		LastNotificationType:   appleSub.LastNotificationType,
		LastNotificationDate:   appleSub.LastNotificationDate,
		CreatedAt:              appleSub.CreatedAt,
		UpdatedAt:              appleSub.UpdatedAt,
	}

	return iapSub, nil
}

// SyncSubscription 同步订阅
func (s *AppleIAPService) SyncSubscription(ctx context.Context, subscriptionID string) error {
	s.logger.WithField("subscription_id", subscriptionID).Info("同步Apple订阅")

	// 获取订阅信息
	subscription, err := s.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("获取订阅信息失败: %w", err)
	}

	// 验证收据
	receipt, err := s.VerifyReceipt(ctx, subscriptionID, false) // 假设是生产环境
	if err != nil {
		return fmt.Errorf("验证收据失败: %w", err)
	}

	// 更新订阅状态
	appleSub := receipt.ConvertToAppleSubscription()
	appleSub.ID = subscription.ID
	appleSub.UserID = subscription.UserID

	err = paymodels.DataBase().WithContext(ctx).Save(appleSub).Error
	if err != nil {
		s.logger.WithError(err).Error("更新Apple订阅失败")
		return fmt.Errorf("更新订阅失败: %w", err)
	}

	s.logger.WithField("subscription_id", subscriptionID).Info("Apple订阅同步成功")
	return nil
}

// AcknowledgePurchase 确认购买
func (s *AppleIAPService) AcknowledgePurchase(ctx context.Context, purchaseToken string) error {
	s.logger.WithField("purchase_token", purchaseToken).Info("确认Apple购买")

	// 1. 从上下文获取用户ID
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("无法从上下文获取用户ID")
		return fmt.Errorf("无法获取用户ID: %w", err)
	}

	// 2. 查找对应的收据记录
	var appleReceipt paymodels.AppleReceipt
	err = paymodels.DataBase().WithContext(ctx).
		Where("user_id = ? AND receipt_data LIKE ?", userID, "%"+purchaseToken+"%").
		First(&appleReceipt).Error

	if err != nil {
		s.logger.WithError(err).Error("查找Apple收据记录失败")
		return fmt.Errorf("查找收据记录失败: %w", err)
	}

	// 3. 更新收据状态为已确认
	appleReceipt.Status = "Acknowledged"
	appleReceipt.UpdatedAt = time.Now()

	if err := paymodels.DataBase().WithContext(ctx).Save(&appleReceipt).Error; err != nil {
		s.logger.WithError(err).Error("更新Apple收据状态失败")
		return fmt.Errorf("更新收据状态失败: %w", err)
	}

	// 4. 如果是订阅类型，同时更新订阅状态
	if appleReceipt.ExpirationDate != nil {
		var appleSub paymodels.AppleSubscription
		err = paymodels.DataBase().WithContext(ctx).
			Where("user_id = ? AND product_id IN (SELECT product_id FROM apple_receipts WHERE id = ?)", userID, appleReceipt.ID).
			First(&appleSub).Error

		if err == nil {
			// 订阅存在，更新状态
			appleSub.Status = "Active"
			appleSub.UpdatedAt = time.Now()
			paymodels.DataBase().WithContext(ctx).Save(&appleSub)
		}
	}

	// 5. 记录确认操作到支付记录
	paymentRecord := &paymodels.PaymentRecord{
		UserID:          int64(userID),
		TransactionID:   purchaseToken,
		ProductID:       "", // 需要从收据中提取
		Amount:          0,  // Apple不提供金额信息
		Currency:        "USD",
		Status:          paymodels.PaymentStatusSuccess,
		PaymentMethod:   paymodels.PaymentMethodApplePay,
		PaymentProvider: "App Store",
	}

	if err := paymodels.DataBase().WithContext(ctx).Create(paymentRecord).Error; err != nil {
		s.logger.WithError(err).Warn("创建支付记录失败，但不影响主要流程")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"purchase_token": purchaseToken,
		"receipt_id":     appleReceipt.ID,
		"receipt_status": appleReceipt.Status,
	}).Info("Apple购买确认完成")

	return nil
}

// ConsumePurchase 消费购买
func (s *AppleIAPService) ConsumePurchase(ctx context.Context, purchaseToken string) error {
	s.logger.WithField("purchase_token", purchaseToken).Info("消费Apple购买")

	// 1. 从上下文获取用户ID
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("无法从上下文获取用户ID")
		return fmt.Errorf("无法获取用户ID: %w", err)
	}

	// 2. 查找对应的收据记录
	var appleReceipt paymodels.AppleReceipt
	err = paymodels.DataBase().WithContext(ctx).
		Where("user_id = ? AND receipt_data LIKE ?", userID, "%"+purchaseToken+"%").
		First(&appleReceipt).Error

	if err != nil {
		s.logger.WithError(err).Error("查找Apple收据记录失败")
		return fmt.Errorf("查找收据记录失败: %w", err)
	}

	// 3. 检查是否已经消费过
	if appleReceipt.Status == "Consumed" {
		s.logger.WithField("purchase_token", purchaseToken).Warn("Apple购买已经消费过")
		return fmt.Errorf("购买已经消费过")
	}

	// 4. 检查是否为一次性购买（订阅类型不需要消费）
	if appleReceipt.ExpirationDate != nil {
		s.logger.WithField("purchase_token", purchaseToken).Warn("Apple订阅类型不需要消费")
		return fmt.Errorf("订阅类型不需要消费")
	}

	// 5. 更新收据状态为已消费
	appleReceipt.Status = "Consumed"
	appleReceipt.UpdatedAt = time.Now()

	if err := paymodels.DataBase().WithContext(ctx).Save(&appleReceipt).Error; err != nil {
		s.logger.WithError(err).Error("更新Apple收据状态失败")
		return fmt.Errorf("更新收据状态失败: %w", err)
	}

	// 6. 记录消费操作到支付记录
	paymentRecord := &paymodels.PaymentRecord{
		UserID:          int64(userID),
		TransactionID:   purchaseToken,
		ProductID:       "", // 需要从收据中提取
		Amount:          0,  // Apple不提供金额信息
		Currency:        "USD",
		Status:          paymodels.PaymentStatusSuccess,
		PaymentMethod:   paymodels.PaymentMethodApplePay,
		PaymentProvider: "App Store",
	}

	if err := paymodels.DataBase().WithContext(ctx).Create(paymentRecord).Error; err != nil {
		s.logger.WithError(err).Warn("创建支付记录失败，但不影响主要流程")
	}

	// 7. 记录消费历史
	consumptionRecord := &paymodels.PaymentRecord{
		UserID:          int64(userID),
		TransactionID:   purchaseToken + "_consumed",
		ProductID:       "",
		Amount:          0,
		Currency:        "USD",
		Status:          paymodels.PaymentStatusSuccess,
		PaymentMethod:   paymodels.PaymentMethodApplePay,
		PaymentProvider: "App Store",
	}

	if err := paymodels.DataBase().WithContext(ctx).Create(consumptionRecord).Error; err != nil {
		s.logger.WithError(err).Warn("创建消费记录失败，但不影响主要流程")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"purchase_token": purchaseToken,
		"receipt_id":     appleReceipt.ID,
		"receipt_status": appleReceipt.Status,
	}).Info("Apple购买消费完成")

	return nil
}

// HandleNotification 处理通知
func (s *AppleIAPService) HandleNotification(ctx context.Context, notification *IAPNotification) error {
	s.logger.WithFields(logrus.Fields{
		"notification_id":   notification.NotificationID,
		"notification_type": notification.NotificationType,
		"subtype":           notification.Subtype,
	}).Info("处理Apple通知")

	// 1. 验证通知签名（简化实现）
	if err := s.verifyAppleNotificationSignature(notification); err != nil {
		s.logger.WithError(err).Error("Apple通知签名验证失败")
		return fmt.Errorf("通知签名验证失败: %w", err)
	}

	// 2. 解析通知数据
	transactionInfo, err := s.parseNotificationData(notification)
	if err != nil {
		s.logger.WithError(err).Error("解析Apple通知数据失败")
		return fmt.Errorf("解析通知数据失败: %w", err)
	}

	// 3. 根据通知类型处理
	switch notification.NotificationType {
	case "INITIAL_BUY":
		return s.handleInitialPurchase(ctx, notification, transactionInfo)
	case "DID_RENEW":
		return s.handleSubscriptionRenewal(ctx, notification, transactionInfo)
	case "DID_CHANGE_RENEWAL_STATUS":
		return s.handleRenewalStatusChange(ctx, notification, transactionInfo)
	case "DID_FAIL_TO_RENEW":
		return s.handleSubscriptionExpiration(ctx, notification, transactionInfo)
	case "DID_RECOVER":
		return s.handleSubscriptionRecovery(ctx, notification, transactionInfo)
	case "REVOKE":
		return s.handleSubscriptionRevocation(ctx, notification, transactionInfo)
	case "REFUND":
		return s.handleRefund(ctx, notification, transactionInfo)
	default:
		s.logger.WithField("notification_type", notification.NotificationType).Warn("未知的Apple通知类型")
		return nil
	}
}

// SyncProducts 同步产品
func (s *AppleIAPService) SyncProducts(ctx context.Context) error {
	s.logger.Info("开始同步Apple产品")

	// 创建产品服务
	productService := NewIAPProductService()

	// 同步Apple产品
	err := productService.SyncProductsFromApple(ctx)
	if err != nil {
		s.logger.WithError(err).Error("同步Apple产品失败")
		return fmt.Errorf("同步Apple产品失败: %w", err)
	}

	s.logger.Info("Apple产品同步成功")
	return nil
}

// parseReceiptData 解析收据数据
func (s *AppleIAPService) parseReceiptData(resp *AppleReceiptResponse, iapReceipt *IAPReceipt) error {
	iapReceipt.ApplicationVersion = resp.Receipt.ApplicationVersion
	iapReceipt.BundleID = resp.Receipt.BundleID
	iapReceipt.Environment = resp.Environment
	iapReceipt.Status = fmt.Sprintf("%d", resp.Status)

	// 解析创建日期
	if creationDate, err := parseAppleTimestamp(resp.Receipt.CreationDate); err == nil {
		iapReceipt.CreationDate = creationDate
	} else {
		s.logger.WithError(err).Warn("解析收据创建日期失败")
	}

	// 查找最新的订阅信息
	var latestReceiptInfo *AppleReceiptInfo
	if len(resp.LatestReceiptInfo) > 0 {
		for _, info := range resp.LatestReceiptInfo {
			if latestReceiptInfo == nil {
				latestReceiptInfo = &info
			} else {
				// 比较过期时间找到最新的
				currentExpiry, err1 := parseAppleTimestamp(latestReceiptInfo.ExpiresDate)
				newExpiry, err2 := parseAppleTimestamp(info.ExpiresDate)
				if err1 == nil && err2 == nil && newExpiry.After(currentExpiry) {
					latestReceiptInfo = &info
				}
			}
		}
	}

	if latestReceiptInfo != nil {
		// 填充产品 ID
		if latestReceiptInfo.ProductID != "" {
			iapReceipt.ProductID = latestReceiptInfo.ProductID
		}
		if latestReceiptInfo.OriginalTransactionID != "" {
			iapReceipt.OriginalTransactionID = latestReceiptInfo.OriginalTransactionID
		}
		if latestReceiptInfo.TransactionID != "" {
			iapReceipt.SubscriptionTransactionID = latestReceiptInfo.TransactionID
		}

		if purchaseDate, err := parseAppleTimestamp(latestReceiptInfo.PurchaseDate); err == nil {
			iapReceipt.CreationDate = purchaseDate
		}

		if latestReceiptInfo.ExpiresDate != "" {
			if expiryDate, err := parseAppleTimestamp(latestReceiptInfo.ExpiresDate); err == nil {
				iapReceipt.ExpirationDate = &expiryDate
				// 判断订阅状态
				if expiryDate.After(time.Now()) {
					iapReceipt.Status = "Active"
				} else {
					iapReceipt.Status = "Expired"
				}
			} else {
				s.logger.WithError(err).Warn("解析订阅过期日期失败")
			}
		} else {
			// 如果没有过期日期，则可能是一次性购买或非订阅产品
			iapReceipt.Status = "OneTimePurchase"
		}
	} else {
		// 如果没有LatestReceiptInfo，则可能是一个非常旧的收据或非订阅产品
		iapReceipt.Status = "NoSubscriptionInfo"
	}

	// 生成验证哈希
	iapReceipt.VerificationHash = s.generateReceiptHash(iapReceipt)

	return nil
}

// 其他辅助方法...
func (s *AppleIAPService) verifyAppleNotificationSignature(notification *IAPNotification) error {
	s.logger.WithFields(logrus.Fields{
		"notification_id":   notification.NotificationID,
		"notification_type": notification.NotificationType,
	}).Debug("开始验证Apple通知签名")

	// 1. 检查是否有签名数据
	if notification.RawData == "" {
		s.logger.Warn("Apple通知缺少原始数据，跳过签名验证")
		return nil
	}

	// 2. 解析通知数据
	var notificationData map[string]interface{}
	if err := json.Unmarshal([]byte(notification.RawData), &notificationData); err != nil {
		s.logger.WithError(err).Error("解析Apple通知数据失败")
		return fmt.Errorf("解析通知数据失败: %w", err)
	}

	// 3. 检查是否有signedPayload
	signedPayload, ok := notificationData["signedPayload"].(string)
	if !ok || signedPayload == "" {
		s.logger.Warn("Apple通知缺少signedPayload，跳过签名验证")
		return nil
	}

	// 4. 解析JWT header获取签名算法和key ID
	parts := strings.Split(signedPayload, ".")
	if len(parts) != 3 {
		return fmt.Errorf("无效的JWT格式")
	}

	// 解析header
	header, err := s.parseJWTHeader(parts[0])
	if err != nil {
		s.logger.WithError(err).Error("解析JWT header失败")
		return fmt.Errorf("解析JWT header失败: %w", err)
	}

	// 5. 获取Apple公钥
	keyID, ok := header["kid"].(string)
	if !ok {
		return fmt.Errorf("JWT header中缺少key ID")
	}

	publicKey, err := s.getApplePublicKey(keyID)
	if err != nil {
		s.logger.WithError(err).Error("获取Apple公钥失败")
		return fmt.Errorf("获取Apple公钥失败: %w", err)
	}

	// 6. 验证JWT签名
	if err := s.verifyJWTSignature(signedPayload, publicKey); err != nil {
		s.logger.WithError(err).Error("JWT签名验证失败")
		return fmt.Errorf("JWT签名验证失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"notification_id": notification.NotificationID,
		"key_id":          keyID,
	}).Debug("Apple通知签名验证成功")

	return nil
}

func (s *AppleIAPService) parseNotificationData(notification *IAPNotification) (*AppleTransactionInfo, error) {
	s.logger.WithFields(logrus.Fields{
		"notification_id":   notification.NotificationID,
		"notification_type": notification.NotificationType,
		"subtype":           notification.Subtype,
		"raw_data_length":   len(notification.RawData),
	}).Debug("开始解析Apple通知数据")

	// 1. 解析原始数据
	var notificationData map[string]interface{}
	if notification.RawData != "" {
		if err := json.Unmarshal([]byte(notification.RawData), &notificationData); err != nil {
			s.logger.WithError(err).Error("解析Apple通知原始数据失败")
			return nil, fmt.Errorf("解析通知原始数据失败: %w", err)
		}
	} else {
		// 如果没有原始数据，从通知对象中提取信息
		notificationData = map[string]interface{}{
			"notificationType": notification.NotificationType,
			"subtype":          notification.Subtype,
			"version":          notification.Version,
		}
	}

	// 2. 提取交易信息
	transactionInfo := &AppleTransactionInfo{}

	// 从signedPayload中提取交易信息
	if signedPayload, ok := notificationData["signedPayload"].(string); ok && signedPayload != "" {
		// 解析JWT payload
		payload, err := s.parseJWTPayload(signedPayload)
		if err != nil {
			s.logger.WithError(err).Error("解析Apple通知JWT payload失败")
			return nil, fmt.Errorf("解析JWT payload失败: %w", err)
		}

		// 提取交易信息
		if data, ok := payload["data"].(map[string]interface{}); ok {
			if transactionID, ok := data["transaction_id"].(string); ok {
				transactionInfo.TransactionID = transactionID
			}
			if originalTransactionID, ok := data["original_transaction_id"].(string); ok {
				transactionInfo.OriginalTransactionID = originalTransactionID
			}
			if productID, ok := data["product_id"].(string); ok {
				transactionInfo.ProductID = productID
			}
			if purchaseDate, ok := data["purchase_date"].(string); ok {
				if parsedDate, err := parseAppleTimestamp(purchaseDate); err == nil {
					transactionInfo.PurchaseDate = parsedDate
				}
			}
			if expiresDate, ok := data["expires_date"].(string); ok {
				if parsedDate, err := parseAppleTimestamp(expiresDate); err == nil {
					transactionInfo.ExpiresDate = &parsedDate
				}
			}
			if isInIntroOfferPeriod, ok := data["is_in_intro_offer_period"].(string); ok {
				transactionInfo.IsInIntroOfferPeriod = isInIntroOfferPeriod == "true"
			}
			if isInGracePeriod, ok := data["is_in_grace_period"].(string); ok {
				transactionInfo.IsInGracePeriod = isInGracePeriod == "true"
			}
			if autoRenewStatus, ok := data["auto_renew_status"].(string); ok {
				transactionInfo.AutoRenewStatus = autoRenewStatus
			}
		}
	}

	// 3. 记录解析结果
	s.logger.WithFields(logrus.Fields{
		"original_transaction_id":  transactionInfo.OriginalTransactionID,
		"transaction_id":           transactionInfo.TransactionID,
		"product_id":               transactionInfo.ProductID,
		"purchase_date":            transactionInfo.PurchaseDate,
		"expires_date":             transactionInfo.ExpiresDate,
		"is_in_intro_offer_period": transactionInfo.IsInIntroOfferPeriod,
		"is_in_grace_period":       transactionInfo.IsInGracePeriod,
		"auto_renew_status":        transactionInfo.AutoRenewStatus,
	}).Debug("Apple通知数据解析完成")

	return transactionInfo, nil
}

func (s *AppleIAPService) handleInitialPurchase(ctx context.Context, notification *IAPNotification, transactionInfo *AppleTransactionInfo) error {
	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"transaction_id":          transactionInfo.TransactionID,
		"product_id":              transactionInfo.ProductID,
		"purchase_date":           transactionInfo.PurchaseDate,
	}).Info("处理Apple初始购买通知")

	// 记录支付成功指标
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.RecordPaymentSimple("apple", "subscription", "success")
		metrics.IncPaymentSubscription("apple", transactionInfo.ProductID)
	}

	// 1. 保存通知记录
	if err := s.saveAppleNotification(ctx, notification, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Apple通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 检查订阅是否已存在
	var existingSub paymodels.AppleSubscription
	err := paymodels.DataBase().WithContext(ctx).
		Where("original_transaction_id = ?", transactionInfo.OriginalTransactionID).
		First(&existingSub).Error

	if err != nil {
		// 订阅不存在，创建新记录
		if err := s.createAppleSubscriptionFromTransaction(ctx, transactionInfo, notification); err != nil {
			return err
		}

		s.logger.WithFields(logrus.Fields{
			"original_transaction_id": transactionInfo.OriginalTransactionID,
			"product_id":              transactionInfo.ProductID,
		}).Info("创建新的Apple订阅记录")
	} else {
		// 订阅已存在，更新记录
		if err := s.updateAppleSubscriptionFromTransaction(ctx, transactionInfo, notification, func(appleSub *paymodels.AppleSubscription) {
			appleSub.PurchaseDate = transactionInfo.PurchaseDate
			appleSub.ExpiresDate = transactionInfo.ExpiresDate
			appleSub.Status = "Active"
			appleSub.AutoRenewStatus = transactionInfo.AutoRenewStatus
			appleSub.IsInIntroOfferPeriod = transactionInfo.IsInIntroOfferPeriod
			appleSub.IsInGracePeriod = transactionInfo.IsInGracePeriod
		}); err != nil {
			return err
		}

		s.logger.WithFields(logrus.Fields{
			"original_transaction_id": transactionInfo.OriginalTransactionID,
			"product_id":              transactionInfo.ProductID,
		}).Info("更新Apple订阅记录")
	}

	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"product_id":              transactionInfo.ProductID,
		"purchase_date":           transactionInfo.PurchaseDate,
	}).Info("Apple初始购买通知处理完成")

	return nil
}

func (s *AppleIAPService) handleSubscriptionRenewal(ctx context.Context, notification *IAPNotification, transactionInfo *AppleTransactionInfo) error {
	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"transaction_id":          transactionInfo.TransactionID,
		"product_id":              transactionInfo.ProductID,
		"expires_date":            transactionInfo.ExpiresDate,
	}).Info("处理Apple订阅续费通知")

	// 记录续费成功指标
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.RecordPaymentSimple("apple", "subscription", "success")
	}

	// 1. 保存通知记录
	if err := s.saveAppleNotification(ctx, notification, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Apple通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 更新订阅记录
	if err := s.updateAppleSubscriptionFromTransaction(ctx, transactionInfo, notification, func(appleSub *paymodels.AppleSubscription) {
		appleSub.ExpiresDate = transactionInfo.ExpiresDate
		appleSub.Status = "Active"
		appleSub.AutoRenewStatus = transactionInfo.AutoRenewStatus
		appleSub.IsInIntroOfferPeriod = transactionInfo.IsInIntroOfferPeriod
		appleSub.IsInGracePeriod = transactionInfo.IsInGracePeriod
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"product_id":              transactionInfo.ProductID,
		"expires_date":            transactionInfo.ExpiresDate,
	}).Info("Apple订阅续费通知处理完成")

	return nil
}

func (s *AppleIAPService) handleRenewalStatusChange(ctx context.Context, notification *IAPNotification, transactionInfo *AppleTransactionInfo) error {
	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"transaction_id":          transactionInfo.TransactionID,
		"product_id":              transactionInfo.ProductID,
		"auto_renew_status":       transactionInfo.AutoRenewStatus,
	}).Info("处理Apple续费状态变更通知")

	// 1. 保存通知记录
	if err := s.saveAppleNotification(ctx, notification, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Apple通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 更新订阅记录
	if err := s.updateAppleSubscriptionFromTransaction(ctx, transactionInfo, notification, func(appleSub *paymodels.AppleSubscription) {
		appleSub.AutoRenewStatus = transactionInfo.AutoRenewStatus
		// 如果自动续费关闭，状态可能变为即将过期
		if transactionInfo.AutoRenewStatus == "Off" {
			appleSub.Status = "WillExpire"
		} else {
			appleSub.Status = "Active"
		}
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"product_id":              transactionInfo.ProductID,
		"auto_renew_status":       transactionInfo.AutoRenewStatus,
	}).Info("Apple续费状态变更通知处理完成")

	return nil
}

func (s *AppleIAPService) handleSubscriptionExpiration(ctx context.Context, notification *IAPNotification, transactionInfo *AppleTransactionInfo) error {
	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"transaction_id":          transactionInfo.TransactionID,
		"product_id":              transactionInfo.ProductID,
		"expires_date":            transactionInfo.ExpiresDate,
	}).Info("处理Apple订阅过期通知")

	// 记录订阅过期指标
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.DecPaymentSubscription("apple", transactionInfo.ProductID)
	}

	// 1. 保存通知记录
	if err := s.saveAppleNotification(ctx, notification, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Apple通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 更新订阅记录
	if err := s.updateAppleSubscriptionFromTransaction(ctx, transactionInfo, notification, func(appleSub *paymodels.AppleSubscription) {
		appleSub.Status = "Expired"
		appleSub.AutoRenewStatus = "Off"
		appleSub.IsInGracePeriod = false
		appleSub.IsInIntroOfferPeriod = false
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"product_id":              transactionInfo.ProductID,
		"expires_date":            transactionInfo.ExpiresDate,
	}).Info("Apple订阅过期通知处理完成")

	return nil
}

func (s *AppleIAPService) handleSubscriptionRecovery(ctx context.Context, notification *IAPNotification, transactionInfo *AppleTransactionInfo) error {
	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"transaction_id":          transactionInfo.TransactionID,
		"product_id":              transactionInfo.ProductID,
		"expires_date":            transactionInfo.ExpiresDate,
	}).Info("处理Apple订阅恢复通知")

	// 1. 保存通知记录
	if err := s.saveAppleNotification(ctx, notification, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Apple通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 更新订阅记录
	if err := s.updateAppleSubscriptionFromTransaction(ctx, transactionInfo, notification, func(appleSub *paymodels.AppleSubscription) {
		appleSub.Status = "Active"
		appleSub.AutoRenewStatus = transactionInfo.AutoRenewStatus
		appleSub.ExpiresDate = transactionInfo.ExpiresDate
		appleSub.IsInGracePeriod = transactionInfo.IsInGracePeriod
		appleSub.IsInIntroOfferPeriod = transactionInfo.IsInIntroOfferPeriod
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"product_id":              transactionInfo.ProductID,
		"expires_date":            transactionInfo.ExpiresDate,
	}).Info("Apple订阅恢复通知处理完成")

	return nil
}

func (s *AppleIAPService) handleSubscriptionRevocation(ctx context.Context, notification *IAPNotification, transactionInfo *AppleTransactionInfo) error {
	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"transaction_id":          transactionInfo.TransactionID,
		"product_id":              transactionInfo.ProductID,
	}).Info("处理Apple订阅撤销通知")

	// 1. 保存通知记录
	if err := s.saveAppleNotification(ctx, notification, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Apple通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 更新订阅记录
	if err := s.updateAppleSubscriptionFromTransaction(ctx, transactionInfo, notification, func(appleSub *paymodels.AppleSubscription) {
		appleSub.Status = "Revoked"
		appleSub.AutoRenewStatus = "Off"
		appleSub.IsInGracePeriod = false
		appleSub.IsInIntroOfferPeriod = false
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"product_id":              transactionInfo.ProductID,
	}).Info("Apple订阅撤销通知处理完成")

	return nil
}

func (s *AppleIAPService) handleRefund(ctx context.Context, notification *IAPNotification, transactionInfo *AppleTransactionInfo) error {
	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"transaction_id":          transactionInfo.TransactionID,
		"product_id":              transactionInfo.ProductID,
	}).Info("处理Apple退款通知")

	// 记录退款指标
	if metrics := telemetry.GetDefaultMetrics(); metrics != nil {
		metrics.PaymentRefundsTotal.WithLabelValues("apple", "user_request").Inc()
		metrics.DecPaymentSubscription("apple", transactionInfo.ProductID)
	}

	// 1. 保存通知记录
	if err := s.saveAppleNotification(ctx, notification, "Success"); err != nil {
		s.logger.WithError(err).Error("保存Apple通知记录失败")
		return fmt.Errorf("保存通知记录失败: %w", err)
	}

	// 2. 更新订阅记录
	if err := s.updateAppleSubscriptionFromTransaction(ctx, transactionInfo, notification, func(appleSub *paymodels.AppleSubscription) {
		appleSub.Status = "Refunded"
		appleSub.AutoRenewStatus = "Off"
		appleSub.IsInGracePeriod = false
		appleSub.IsInIntroOfferPeriod = false
	}); err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"product_id":              transactionInfo.ProductID,
	}).Info("Apple退款通知处理完成")

	return nil
}

func (s *AppleIAPService) generateReceiptHash(receipt *IAPReceipt) string {
	// 实现收据哈希生成
	// 使用收据的关键信息生成哈希值，用于去重和验证

	// 构建哈希源字符串
	hashSource := fmt.Sprintf("%s_%d_%s_%s_%s",
		receipt.Platform,
		receipt.UserID,
		receipt.BundleID,
		receipt.ReceiptData,
		receipt.CreationDate.Format(time.RFC3339),
	)

	// 使用简单的哈希算法（实际应用中可以使用更安全的哈希算法）
	// 这里使用字符串长度和字符码的简单组合
	hash := 0
	for _, char := range hashSource {
		hash = hash*31 + int(char)
	}

	// 转换为十六进制字符串
	return fmt.Sprintf("%x", hash)
}

// GetPurchaseStatus 获取购买状态
func (s *AppleIAPService) GetPurchaseStatus(ctx context.Context, purchaseToken string) (string, error) {
	s.logger.WithField("purchase_token", purchaseToken).Debug("获取Apple购买状态")

	// 从上下文获取用户ID
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("无法从上下文获取用户ID")
		return "", fmt.Errorf("无法获取用户ID: %w", err)
	}

	// 查找收据记录
	var appleReceipt paymodels.AppleReceipt
	err = paymodels.DataBase().WithContext(ctx).
		Where("user_id = ? AND receipt_data LIKE ?", userID, "%"+purchaseToken+"%").
		First(&appleReceipt).Error

	if err != nil {
		s.logger.WithError(err).Error("查找Apple收据记录失败")
		return "", fmt.Errorf("查找收据记录失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"purchase_token": purchaseToken,
		"status":         appleReceipt.Status,
	}).Debug("获取Apple购买状态成功")

	return appleReceipt.Status, nil
}

// IsPurchaseConsumed 检查购买是否已消费
func (s *AppleIAPService) IsPurchaseConsumed(ctx context.Context, purchaseToken string) (bool, error) {
	status, err := s.GetPurchaseStatus(ctx, purchaseToken)
	if err != nil {
		return false, err
	}

	return status == "Consumed", nil
}

// IsPurchaseAcknowledged 检查购买是否已确认
func (s *AppleIAPService) IsPurchaseAcknowledged(ctx context.Context, purchaseToken string) (bool, error) {
	status, err := s.GetPurchaseStatus(ctx, purchaseToken)
	if err != nil {
		return false, err
	}

	return status == "Acknowledged" || status == "Consumed", nil
}

// GetPurchaseHistory 获取用户购买历史
func (s *AppleIAPService) GetPurchaseHistory(ctx context.Context, userID uint64, limit int) ([]paymodels.PaymentRecord, error) {
	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"limit":   limit,
	}).Debug("获取Apple购买历史")

	var records []paymodels.PaymentRecord
	query := paymodels.DataBase().WithContext(ctx).
		Where("user_id = ? AND platform = ?", userID, "apple").
		Order("create_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&records).Error
	if err != nil {
		s.logger.WithError(err).Error("获取Apple购买历史失败")
		return nil, fmt.Errorf("获取购买历史失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"count":   len(records),
	}).Debug("获取Apple购买历史成功")

	return records, nil
}

// ValidatePurchaseToken 验证购买令牌
func (s *AppleIAPService) ValidatePurchaseToken(ctx context.Context, purchaseToken string) error {
	s.logger.WithField("purchase_token", purchaseToken).Debug("验证Apple购买令牌")

	// 1. 检查令牌格式
	if purchaseToken == "" {
		return fmt.Errorf("购买令牌不能为空")
	}

	// 2. 从上下文获取用户ID
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("无法从上下文获取用户ID")
		return fmt.Errorf("无法获取用户ID: %w", err)
	}

	// 3. 查找对应的收据记录
	var appleReceipt paymodels.AppleReceipt
	err = paymodels.DataBase().WithContext(ctx).
		Where("user_id = ? AND receipt_data LIKE ?", userID, "%"+purchaseToken+"%").
		First(&appleReceipt).Error

	if err != nil {
		s.logger.WithError(err).Error("查找Apple收据记录失败")
		return fmt.Errorf("无效的购买令牌: %w", err)
	}

	// 4. 检查收据状态
	if appleReceipt.Status == "Invalid" {
		return fmt.Errorf("收据状态无效")
	}

	// 5. 检查收据是否过期（对于订阅类型）
	if appleReceipt.ExpirationDate != nil && appleReceipt.ExpirationDate.Before(time.Now()) {
		return fmt.Errorf("收据已过期")
	}

	s.logger.WithFields(logrus.Fields{
		"purchase_token": purchaseToken,
		"receipt_id":     appleReceipt.ID,
		"status":         appleReceipt.Status,
	}).Debug("Apple购买令牌验证成功")

	return nil
}

// GetActiveSubscriptions 获取用户活跃订阅
func (s *AppleIAPService) GetActiveSubscriptions(ctx context.Context, userID uint64) ([]paymodels.AppleSubscription, error) {
	s.logger.WithField("user_id", userID).Debug("获取Apple活跃订阅")

	var subscriptions []paymodels.AppleSubscription
	err := paymodels.DataBase().WithContext(ctx).
		Where("user_id = ? AND status = ? AND (expires_date IS NULL OR expires_date > ?)",
			userID, "Active", time.Now()).
		Find(&subscriptions).Error

	if err != nil {
		s.logger.WithError(err).Error("获取Apple活跃订阅失败")
		return nil, fmt.Errorf("获取活跃订阅失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id": userID,
		"count":   len(subscriptions),
	}).Debug("获取Apple活跃订阅成功")

	return subscriptions, nil
}

// TrackPurchaseEvent 跟踪购买事件
func (s *AppleIAPService) TrackPurchaseEvent(ctx context.Context, eventType string, purchaseToken string, metadata map[string]interface{}) error {
	s.logger.WithFields(logrus.Fields{
		"event_type":     eventType,
		"purchase_token": purchaseToken,
		"metadata":       metadata,
	}).Debug("跟踪Apple购买事件")

	// 从上下文获取用户ID
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("无法从上下文获取用户ID")
		return fmt.Errorf("无法获取用户ID: %w", err)
	}

	// 创建事件记录
	eventRecord := &paymodels.PaymentRecord{
		UserID:          int64(userID),
		TransactionID:   purchaseToken + "_" + eventType,
		ProductID:       "",
		Amount:          0,
		Currency:        "USD",
		Status:          paymodels.PaymentStatusSuccess,
		PaymentMethod:   paymodels.PaymentMethodApplePay,
		PaymentProvider: "App Store",
	}

	// 如果有元数据，可以存储到备注字段或其他字段
	if metadata != nil {
		if metadataJSON, err := json.Marshal(metadata); err == nil {
			// 这里可以扩展PaymentRecord结构来存储元数据
			s.logger.WithField("metadata", string(metadataJSON)).Debug("购买事件元数据")
		}
	}

	if err := paymodels.DataBase().WithContext(ctx).Create(eventRecord).Error; err != nil {
		s.logger.WithError(err).Error("创建购买事件记录失败")
		return fmt.Errorf("创建事件记录失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"event_type":     eventType,
		"purchase_token": purchaseToken,
	}).Info("Apple购买事件跟踪完成")

	return nil
}

// GetPurchaseAnalytics 获取购买分析数据
func (s *AppleIAPService) GetPurchaseAnalytics(ctx context.Context, userID uint64, startDate, endDate time.Time) (map[string]interface{}, error) {
	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"start_date": startDate,
		"end_date":   endDate,
	}).Debug("获取Apple购买分析数据")

	// 1. 获取购买记录统计
	var totalPurchases int64
	err := paymodels.DataBase().WithContext(ctx).
		Model(&paymodels.PaymentRecord{}).
		Where("user_id = ? AND platform = ? AND create_at BETWEEN ? AND ?",
			userID, "apple", startDate, endDate).
		Count(&totalPurchases).Error

	if err != nil {
		s.logger.WithError(err).Error("获取购买记录统计失败")
		return nil, fmt.Errorf("获取购买统计失败: %w", err)
	}

	// 2. 获取按状态分组的统计
	var statusStats []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}

	err = paymodels.DataBase().WithContext(ctx).
		Model(&paymodels.PaymentRecord{}).
		Select("status, COUNT(*) as count").
		Where("user_id = ? AND platform = ? AND create_at BETWEEN ? AND ?",
			userID, "apple", startDate, endDate).
		Group("status").
		Scan(&statusStats).Error

	if err != nil {
		s.logger.WithError(err).Error("获取状态统计失败")
		return nil, fmt.Errorf("获取状态统计失败: %w", err)
	}

	// 3. 获取活跃订阅数量
	var activeSubscriptions int64
	err = paymodels.DataBase().WithContext(ctx).
		Model(&paymodels.AppleSubscription{}).
		Where("user_id = ? AND status = ? AND (expires_date IS NULL OR expires_date > ?)",
			userID, "Active", time.Now()).
		Count(&activeSubscriptions).Error

	if err != nil {
		s.logger.WithError(err).Error("获取活跃订阅统计失败")
		return nil, fmt.Errorf("获取活跃订阅统计失败: %w", err)
	}

	// 4. 构建分析结果
	analytics := map[string]interface{}{
		"total_purchases":      totalPurchases,
		"active_subscriptions": activeSubscriptions,
		"status_breakdown":     statusStats,
		"period": map[string]interface{}{
			"start_date": startDate,
			"end_date":   endDate,
		},
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":              userID,
		"total_purchases":      totalPurchases,
		"active_subscriptions": activeSubscriptions,
	}).Debug("Apple购买分析数据获取成功")

	return analytics, nil
}

// CleanupExpiredPurchases 清理过期的购买记录
func (s *AppleIAPService) CleanupExpiredPurchases(ctx context.Context, olderThan time.Time) error {
	s.logger.WithField("older_than", olderThan).Info("开始清理过期的Apple购买记录")

	// 1. 清理过期的收据记录
	result := paymodels.DataBase().WithContext(ctx).
		Where("expiration_date < ? AND status IN (?)", olderThan, []string{"Expired", "Invalid"}).
		Delete(&paymodels.AppleReceipt{})

	if result.Error != nil {
		s.logger.WithError(result.Error).Error("清理过期收据记录失败")
		return fmt.Errorf("清理过期收据记录失败: %w", result.Error)
	}

	// 2. 清理过期的订阅记录
	result = paymodels.DataBase().WithContext(ctx).
		Where("expires_date < ? AND status IN (?)", olderThan, []string{"Expired", "Canceled", "Revoked"}).
		Delete(&paymodels.AppleSubscription{})

	if result.Error != nil {
		s.logger.WithError(result.Error).Error("清理过期订阅记录失败")
		return fmt.Errorf("清理过期订阅记录失败: %w", result.Error)
	}

	// 3. 清理过期的通知记录
	result = paymodels.DataBase().WithContext(ctx).
		Where("created_at < ?", olderThan).
		Delete(&paymodels.AppleNotification{})

	if result.Error != nil {
		s.logger.WithError(result.Error).Error("清理过期通知记录失败")
		return fmt.Errorf("清理过期通知记录失败: %w", result.Error)
	}

	s.logger.WithFields(logrus.Fields{
		"older_than":       olderThan,
		"receipts_deleted": result.RowsAffected,
	}).Info("Apple过期购买记录清理完成")

	return nil
}

// parseJWTPayload 解析JWT payload
func (s *AppleIAPService) parseJWTPayload(jwt string) (map[string]interface{}, error) {
	// JWT格式: header.payload.signature
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("无效的JWT格式")
	}

	// 解析payload部分
	payload := parts[1]

	// 添加padding如果需要
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}

	// Base64解码
	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("Base64解码失败: %w", err)
	}

	// JSON解析
	var result map[string]interface{}
	if err := json.Unmarshal(decoded, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	return result, nil
}

// parseJWTHeader 解析JWT header
func (s *AppleIAPService) parseJWTHeader(headerPart string) (map[string]interface{}, error) {
	// 添加padding如果需要
	if len(headerPart)%4 != 0 {
		headerPart += strings.Repeat("=", 4-len(headerPart)%4)
	}

	// Base64解码
	decoded, err := base64.URLEncoding.DecodeString(headerPart)
	if err != nil {
		return nil, fmt.Errorf("Base64解码失败: %w", err)
	}

	// JSON解析
	var result map[string]interface{}
	if err := json.Unmarshal(decoded, &result); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w", err)
	}

	return result, nil
}

// getApplePublicKey 获取Apple公钥
func (s *AppleIAPService) getApplePublicKey(keyID string) (*rsa.PublicKey, error) {
	// 检查缓存
	s.keyCache.mu.RLock()
	if key, exists := s.keyCache.Keys[keyID]; exists && time.Now().Before(s.keyCache.ExpiresAt) {
		s.keyCache.mu.RUnlock()
		s.logger.WithField("key_id", keyID).Debug("从缓存获取Apple公钥")
		return key, nil
	}
	s.keyCache.mu.RUnlock()

	// 从Apple JWKS端点获取公钥
	s.logger.WithField("key_id", keyID).Info("从Apple JWKS端点获取公钥")

	// Apple JWKS 端点
	jwksURL := "https://api.storekit.itunes.apple.com/inApps/v1/notifications/keys"

	// 创建HTTP请求
	req, err := http.NewRequest("GET", jwksURL, nil)
	if err != nil {
		s.logger.WithError(err).Error("创建JWKS请求失败")
		return nil, fmt.Errorf("创建JWKS请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "AppleIAPService/1.0")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.WithError(err).Error("请求Apple JWKS端点失败")
		return nil, fmt.Errorf("请求JWKS端点失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		s.logger.WithField("status_code", resp.StatusCode).Error("Apple JWKS端点返回错误状态")
		return nil, fmt.Errorf("JWKS端点返回错误状态: %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.WithError(err).Error("读取JWKS响应失败")
		return nil, fmt.Errorf("读取JWKS响应失败: %w", err)
	}

	// 解析JWKS响应
	var jwksResponse AppleJWKSResponse
	if err := json.Unmarshal(body, &jwksResponse); err != nil {
		s.logger.WithError(err).Error("解析JWKS响应失败")
		return nil, fmt.Errorf("解析JWKS响应失败: %w", err)
	}

	// 查找指定keyID的公钥
	var targetJWK *AppleJWK
	for _, jwk := range jwksResponse.Keys {
		if jwk.Kid == keyID {
			targetJWK = &jwk
			break
		}
	}

	if targetJWK == nil {
		s.logger.WithField("key_id", keyID).Error("未找到指定的公钥")
		return nil, fmt.Errorf("未找到keyID为%s的公钥", keyID)
	}

	// 将JWK转换为RSA公钥
	publicKey, err := s.jwkToRSAPublicKey(targetJWK)
	if err != nil {
		s.logger.WithError(err).Error("转换JWK为RSA公钥失败")
		return nil, fmt.Errorf("转换JWK为RSA公钥失败: %w", err)
	}

	// 更新缓存
	s.keyCache.mu.Lock()
	if s.keyCache.Keys == nil {
		s.keyCache.Keys = make(map[string]*rsa.PublicKey)
	}
	s.keyCache.Keys[keyID] = publicKey
	s.keyCache.ExpiresAt = time.Now().Add(24 * time.Hour) // 缓存24小时
	s.keyCache.mu.Unlock()

	s.logger.WithField("key_id", keyID).Info("成功获取并缓存Apple公钥")
	return publicKey, nil
}

// jwkToRSAPublicKey 将JWK转换为RSA公钥
func (s *AppleIAPService) jwkToRSAPublicKey(jwk *AppleJWK) (*rsa.PublicKey, error) {
	// 验证JWK类型
	if jwk.Kty != "RSA" {
		return nil, fmt.Errorf("不支持的密钥类型: %s", jwk.Kty)
	}

	// 解码RSA模数 (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("解码RSA模数失败: %w", err)
	}

	// 解码RSA指数 (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("解码RSA指数失败: %w", err)
	}

	// 将字节转换为大整数
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	// 创建RSA公钥
	publicKey := &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}

	// 基本验证：检查指数是否有效
	if publicKey.E <= 1 {
		return nil, fmt.Errorf("无效的RSA指数: %d", publicKey.E)
	}

	return publicKey, nil
}

// verifyJWTSignature 验证JWT签名
func (s *AppleIAPService) verifyJWTSignature(jwt string, publicKey *rsa.PublicKey) error {
	// 如果公钥为nil，跳过签名验证
	if publicKey == nil {
		s.logger.Warn("Apple公钥为空，跳过JWT签名验证")
		return nil
	}

	s.logger.Debug("开始验证JWT签名")

	// 1. 分割JWT
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return fmt.Errorf("无效的JWT格式，应该有3个部分")
	}

	headerPart := parts[0]
	payloadPart := parts[1]
	signaturePart := parts[2]

	// 2. 构建要验证的消息
	message := headerPart + "." + payloadPart

	// 3. 解码签名
	signature, err := base64.RawURLEncoding.DecodeString(signaturePart)
	if err != nil {
		s.logger.WithError(err).Error("解码JWT签名失败")
		return fmt.Errorf("解码JWT签名失败: %w", err)
	}

	// 4. 计算消息的哈希值
	hasher := sha256.New()
	hasher.Write([]byte(message))
	hashed := hasher.Sum(nil)

	// 5. 验证RSA签名
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed, signature)
	if err != nil {
		s.logger.WithError(err).Error("JWT签名验证失败")
		return fmt.Errorf("JWT签名验证失败: %w", err)
	}

	s.logger.Debug("JWT签名验证成功")
	return nil
}

// saveAppleNotification 保存Apple通知记录
func (s *AppleIAPService) saveAppleNotification(ctx context.Context, notification *IAPNotification, status string) error {
	appleNotification := &paymodels.AppleNotification{
		NotificationID:   notification.NotificationID,
		NotificationType: notification.NotificationType,
		Subtype:          notification.Subtype,
		Version:          notification.Version,
		SignedPayload:    "", // 从RawData中提取
		RawData:          notification.RawData,
		ProcessedAt:      time.Now(),
		Status:           status,
	}

	// 从RawData中提取signedPayload
	if notification.RawData != "" {
		var notificationData map[string]interface{}
		if err := json.Unmarshal([]byte(notification.RawData), &notificationData); err == nil {
			if signedPayload, ok := notificationData["signedPayload"].(string); ok {
				appleNotification.SignedPayload = signedPayload
			}
		}
	}

	return paymodels.DataBase().WithContext(ctx).Create(appleNotification).Error
}

// updateAppleSubscriptionFromTransaction 从交易信息更新Apple订阅记录
func (s *AppleIAPService) updateAppleSubscriptionFromTransaction(ctx context.Context, transactionInfo *AppleTransactionInfo, notification *IAPNotification, updateFunc func(*paymodels.AppleSubscription)) error {
	// 1. 查找订阅记录
	var appleSub paymodels.AppleSubscription
	err := paymodels.DataBase().WithContext(ctx).
		Where("original_transaction_id = ?", transactionInfo.OriginalTransactionID).
		First(&appleSub).Error

	if err != nil {
		s.logger.WithError(err).Error("查找Apple订阅记录失败")
		return fmt.Errorf("查找订阅记录失败: %w", err)
	}

	// 2. 应用更新函数
	updateFunc(&appleSub)

	// 3. 更新通用字段
	appleSub.LastNotificationType = notification.NotificationType
	appleSub.LastNotificationDate = &time.Time{}
	*appleSub.LastNotificationDate = time.Now()
	appleSub.UpdatedAt = time.Now()

	// 4. 保存到数据库
	if err := paymodels.DataBase().WithContext(ctx).Save(&appleSub).Error; err != nil {
		s.logger.WithError(err).Error("更新Apple订阅记录失败")
		return fmt.Errorf("更新订阅记录失败: %w", err)
	}

	return nil
}

// createAppleSubscriptionFromTransaction 从交易信息创建Apple订阅记录
func (s *AppleIAPService) createAppleSubscriptionFromTransaction(ctx context.Context, transactionInfo *AppleTransactionInfo, notification *IAPNotification) error {
	// 从上下文获取用户ID
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		s.logger.WithError(err).Error("无法从上下文获取用户ID")
		return fmt.Errorf("无法获取用户ID: %w", err)
	}

	// 创建新的订阅记录
	appleSub := paymodels.AppleSubscription{
		UserID:                uint64(userID),
		OriginalTransactionID: transactionInfo.OriginalTransactionID,
		ProductID:             transactionInfo.ProductID,
		PurchaseDate:          transactionInfo.PurchaseDate,
		ExpiresDate:           transactionInfo.ExpiresDate,
		Status:                "Active", // 默认状态
		AutoRenewStatus:       transactionInfo.AutoRenewStatus,
		IsInIntroOfferPeriod:  transactionInfo.IsInIntroOfferPeriod,
		IsInGracePeriod:       transactionInfo.IsInGracePeriod,
		LastNotificationType:  notification.NotificationType,
		LastNotificationDate:  &time.Time{},
	}

	*appleSub.LastNotificationDate = time.Now()

	if err := paymodels.DataBase().WithContext(ctx).Create(&appleSub).Error; err != nil {
		s.logger.WithError(err).Error("创建Apple订阅记录失败")
		return fmt.Errorf("创建订阅记录失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":                 userID,
		"original_transaction_id": transactionInfo.OriginalTransactionID,
		"product_id":              transactionInfo.ProductID,
	}).Info("创建新的Apple订阅记录")

	return nil
}

// parseCertificateFromPEM 从PEM格式解析证书
func (s *AppleIAPService) parseCertificateFromPEM(pemData string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("无法解析PEM数据")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析证书失败: %w", err)
	}

	return cert, nil
}

// extractPublicKeyFromCertificate 从证书中提取公钥
func (s *AppleIAPService) extractPublicKeyFromCertificate(cert *x509.Certificate) (*rsa.PublicKey, error) {
	publicKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("证书中的公钥不是RSA类型")
	}

	return publicKey, nil
}

// validateCertificateChain 验证证书链
func (s *AppleIAPService) validateCertificateChain(cert *x509.Certificate, rootCAs *x509.CertPool) error {
	opts := x509.VerifyOptions{
		Roots: rootCAs,
	}

	_, err := cert.Verify(opts)
	if err != nil {
		return fmt.Errorf("证书链验证失败: %w", err)
	}

	return nil
}

// isCertificateExpired 检查证书是否过期
func (s *AppleIAPService) isCertificateExpired(cert *x509.Certificate) bool {
	now := time.Now()
	return now.Before(cert.NotBefore) || now.After(cert.NotAfter)
}

// getAppleRootCAs 获取Apple根证书颁发机构
func (s *AppleIAPService) getAppleRootCAs() *x509.CertPool {
	// 创建系统根证书池
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		s.logger.WithError(err).Warn("无法获取系统根证书池，创建空证书池")
		rootCAs = x509.NewCertPool()
	}

	// 这里可以添加Apple特定的根证书
	// 在实际应用中，可能需要添加Apple的根证书到证书池中

	return rootCAs
}

// validateAppleNotificationPayload 验证Apple通知payload
func (s *AppleIAPService) validateAppleNotificationPayload(payload map[string]interface{}) error {
	// 检查必要字段
	requiredFields := []string{"notificationType", "subtype", "version"}
	for _, field := range requiredFields {
		if _, exists := payload[field]; !exists {
			return fmt.Errorf("缺少必要字段: %s", field)
		}
	}

	// 验证通知类型
	notificationType, ok := payload["notificationType"].(string)
	if !ok {
		return fmt.Errorf("通知类型必须是字符串")
	}

	// 验证版本
	version, ok := payload["version"].(string)
	if !ok {
		return fmt.Errorf("版本必须是字符串")
	}

	// 记录验证信息
	s.logger.WithFields(logrus.Fields{
		"notification_type": notificationType,
		"version":           version,
	}).Debug("Apple通知payload验证通过")

	return nil
}

// refreshApplePublicKeys 刷新Apple公钥缓存
func (s *AppleIAPService) refreshApplePublicKeys() error {
	s.logger.Info("开始刷新Apple公钥缓存")

	// 清空现有缓存
	s.keyCache.mu.Lock()
	s.keyCache.Keys = make(map[string]*rsa.PublicKey)
	s.keyCache.ExpiresAt = time.Time{}
	s.keyCache.mu.Unlock()

	s.logger.Info("Apple公钥缓存已清空，将在下次使用时重新获取")
	return nil
}

package pay

import (
	"context"
	"fmt"
	"time"

	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// IAPPersistenceService IAP数据持久化服务
type IAPPersistenceService struct {
	logger *logrus.Logger
}

// NewIAPPersistenceService 创建IAP数据持久化服务
func NewIAPPersistenceService() *IAPPersistenceService {
	return &IAPPersistenceService{
		logger: logrus.New(),
	}
}

// PersistReceiptValidation 持久化收据验证结果
func (s *IAPPersistenceService) PersistReceiptValidation(ctx context.Context, receipt *IAPReceipt) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":     receipt.UserID,
		"platform":    receipt.Platform,
		"environment": receipt.Environment,
		"status":      receipt.Status,
	}).Info("开始持久化收据验证结果")

	// 开始数据库事务
	tx := paymodels.DataBase().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 1. 保存收据验证记录
	if err := s.saveReceiptValidation(tx, receipt); err != nil {
		tx.Rollback()
		return fmt.Errorf("保存收据验证记录失败: %w", err)
	}

	// 2. 根据平台保存具体的收据数据
	switch receipt.Platform {
	case IAPPlatformApple:
		if err := s.saveAppleReceipt(tx, receipt); err != nil {
			tx.Rollback()
			return fmt.Errorf("保存Apple收据失败: %w", err)
		}
	case IAPPlatformGoogle:
		if err := s.saveGooglePurchase(tx, receipt); err != nil {
			tx.Rollback()
			return fmt.Errorf("保存Google购买记录失败: %w", err)
		}
	default:
		tx.Rollback()
		return fmt.Errorf("不支持的平台: %s", receipt.Platform)
	}

	// 3. 如果是订阅类型，保存订阅信息
	if receipt.ExpirationDate != nil {
		if err := s.saveSubscription(tx, receipt); err != nil {
			tx.Rollback()
			return fmt.Errorf("保存订阅信息失败: %w", err)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":     receipt.UserID,
		"platform":    receipt.Platform,
		"environment": receipt.Environment,
	}).Info("收据验证结果持久化成功")

	return nil
}

// saveReceiptValidation 保存收据验证记录
func (s *IAPPersistenceService) saveReceiptValidation(tx *gorm.DB, receipt *IAPReceipt) error {
	validation := &paymodels.IAPReceiptValidation{
		UserID:           receipt.UserID,
		Platform:         string(receipt.Platform),
		ReceiptData:      receipt.ReceiptData,
		ValidationStatus: "success",
		ResponseData:     fmt.Sprintf("BundleID: %s, Status: %s, Environment: %s", receipt.BundleID, receipt.Status, receipt.Environment),
	}

	return tx.Create(validation).Error
}

// saveAppleReceipt 保存Apple收据
func (s *IAPPersistenceService) saveAppleReceipt(tx *gorm.DB, receipt *IAPReceipt) error {
	appleReceipt := &paymodels.AppleReceipt{
		UserID:             receipt.UserID,
		ReceiptData:        receipt.ReceiptData,
		BundleID:           receipt.BundleID,
		ApplicationVersion: receipt.ApplicationVersion,
		CreationDate:       receipt.CreationDate,
		ExpirationDate:     receipt.ExpirationDate,
		Environment:        receipt.Environment,
		Status:             receipt.Status,
		VerificationHash:   receipt.VerificationHash,
	}

	return tx.Create(appleReceipt).Error
}

// saveGooglePurchase 保存Google购买记录
func (s *IAPPersistenceService) saveGooglePurchase(tx *gorm.DB, receipt *IAPReceipt) error {
	googlePurchase := &paymodels.GooglePurchase{
		UserID:           receipt.UserID,
		PurchaseToken:    receipt.ReceiptData, // Google的收据数据就是购买令牌
		ProductID:        "",                  // 需要从其他地方获取
		PackageName:      receipt.BundleID,    // 使用BundleID作为PackageName
		PurchaseTime:     receipt.CreationDate,
		PurchaseState:    1, // 1表示已购买
		ConsumptionState: 0, // 0表示未消费
		Acknowledged:     false,
		RawData:          receipt.ReceiptData,
	}

	return tx.Create(googlePurchase).Error
}

// saveSubscription 保存订阅信息
func (s *IAPPersistenceService) saveSubscription(tx *gorm.DB, receipt *IAPReceipt) error {
	// 检查是否已存在相同的订阅
	var existingSubscription interface{}
	var err error

	switch receipt.Platform {
	case IAPPlatformApple:
		var appleSub paymodels.AppleSubscription
		err = tx.Where("user_id = ? AND receipt_data = ?", receipt.UserID, receipt.ReceiptData).First(&appleSub).Error
		existingSubscription = &appleSub
	case IAPPlatformGoogle:
		var googleSub paymodels.GoogleSubscription
		err = tx.Where("user_id = ? AND purchase_token = ?", receipt.UserID, receipt.ReceiptData).First(&googleSub).Error
		existingSubscription = &googleSub
	}

	if err == nil {
		// 订阅已存在，更新信息
		return s.updateExistingSubscription(tx, receipt, existingSubscription)
	} else if err == gorm.ErrRecordNotFound {
		// 订阅不存在，创建新订阅
		return s.createNewSubscription(tx, receipt)
	} else {
		return fmt.Errorf("查询现有订阅失败: %w", err)
	}
}

// updateExistingSubscription 更新现有订阅
func (s *IAPPersistenceService) updateExistingSubscription(tx *gorm.DB, receipt *IAPReceipt, existingSubscription interface{}) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":  receipt.UserID,
		"platform": receipt.Platform,
	}).Info("更新现有订阅信息")

	switch receipt.Platform {
	case IAPPlatformApple:
		appleSub := existingSubscription.(*paymodels.AppleSubscription)
		appleSub.ExpiresDate = receipt.ExpirationDate
		appleSub.Status = s.mapReceiptStatusToSubscriptionStatus(receipt.Status)
		appleSub.UpdatedAt = time.Now()
		return tx.Save(appleSub).Error

	case IAPPlatformGoogle:
		googleSub := existingSubscription.(*paymodels.GoogleSubscription)
		googleSub.ExpiryTime = *receipt.ExpirationDate
		googleSub.UpdatedAt = time.Now()
		return tx.Save(googleSub).Error

	default:
		return fmt.Errorf("不支持的平台: %s", receipt.Platform)
	}
}

// createNewSubscription 创建新订阅
func (s *IAPPersistenceService) createNewSubscription(tx *gorm.DB, receipt *IAPReceipt) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":  receipt.UserID,
		"platform": receipt.Platform,
	}).Info("创建新订阅")

	switch receipt.Platform {
	case IAPPlatformApple:
		appleSub := &paymodels.AppleSubscription{
			UserID:                receipt.UserID,
			OriginalTransactionID: receipt.ReceiptData, // 使用收据数据作为事务ID
			ProductID:             "",                  // 需要从其他地方获取
			PurchaseDate:          receipt.CreationDate,
			ExpiresDate:           receipt.ExpirationDate,
			Status:                s.mapReceiptStatusToSubscriptionStatus(receipt.Status),
			AutoRenewStatus:       "On", // 默认自动续费
		}
		return tx.Create(appleSub).Error

	case IAPPlatformGoogle:
		googleSub := &paymodels.GoogleSubscription{
			UserID:        receipt.UserID,
			PurchaseToken: receipt.ReceiptData,
			ProductID:     "", // 需要从其他地方获取
			PackageName:   receipt.BundleID,
			StartTime:     receipt.CreationDate,
			ExpiryTime:    *receipt.ExpirationDate,
			AutoRenewing:  true,
			PaymentState:  1, // 1表示已支付
		}
		return tx.Create(googleSub).Error

	default:
		return fmt.Errorf("不支持的平台: %s", receipt.Platform)
	}
}

// mapReceiptStatusToSubscriptionStatus 将收据状态映射为订阅状态
func (s *IAPPersistenceService) mapReceiptStatusToSubscriptionStatus(receiptStatus string) string {
	switch receiptStatus {
	case "Valid", "Active":
		return "Active"
	case "Expired":
		return "Expired"
	case "Invalid":
		return "Canceled"
	default:
		return "Active"
	}
}

// PersistPaymentRecord 持久化支付记录
func (s *IAPPersistenceService) PersistPaymentRecord(ctx context.Context, receipt *IAPReceipt, productID string, amount int64) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":    receipt.UserID,
		"platform":   receipt.Platform,
		"product_id": productID,
		"amount":     amount,
	}).Info("开始持久化支付记录")

	// 确定支付方式
	var paymentMethod paymodels.PaymentMethod
	var paymentProvider string

	switch receipt.Platform {
	case IAPPlatformApple:
		paymentMethod = paymodels.PaymentMethodApplePay
		paymentProvider = "App Store"
	case IAPPlatformGoogle:
		paymentMethod = paymodels.PaymentMethodGooglePay
		paymentProvider = "Google Play"
	default:
		return fmt.Errorf("不支持的平台: %s", receipt.Platform)
	}

	// 创建支付记录
	paymentRecord := &paymodels.PaymentRecord{
		UserID:           int64(receipt.UserID),
		Amount:           amount,
		Currency:         "USD", // 默认货币，实际应用中应该从产品信息中获取
		Status:           paymodels.PaymentStatusSuccess,
		PaymentMethod:    paymentMethod,
		PaymentProvider:  paymentProvider,
		TransactionID:    receipt.ReceiptData,
		ProductID:        productID,
		PurchaseToken:    receipt.ReceiptData,
		ReceiptData:      receipt.ReceiptData,
		PaymentTime:      &receipt.CreationDate,
		ExpiresTime:      receipt.ExpirationDate,
		Environment:      receipt.Environment,
		IsSubscription:   receipt.ExpirationDate != nil,
		VerificationHash: receipt.VerificationHash,
		IsTest:           receipt.Environment == "Sandbox",
	}

	return paymodels.CreatePaymentRecord(ctx, paymentRecord)
}

// PersistUserSubscription 持久化用户订阅
func (s *IAPPersistenceService) PersistUserSubscription(ctx context.Context, receipt *IAPReceipt, productID string, packagePlanID uint) error {
	s.logger.WithFields(logrus.Fields{
		"user_id":         receipt.UserID,
		"platform":        receipt.Platform,
		"product_id":      productID,
		"package_plan_id": packagePlanID,
	}).Info("开始持久化用户订阅")

	// 确定支付方式
	var paymentMethod paymodels.PaymentMethod
	var paymentProvider string

	switch receipt.Platform {
	case IAPPlatformApple:
		paymentMethod = paymodels.PaymentMethodApplePay
		paymentProvider = "App Store"
	case IAPPlatformGoogle:
		paymentMethod = paymodels.PaymentMethodGooglePay
		paymentProvider = "Google Play"
	default:
		return fmt.Errorf("不支持的平台: %s", receipt.Platform)
	}

	// 创建用户订阅记录
	userSubscription := &paymodels.UserSubscription{
		UserID:          int64(receipt.UserID),
		PackagePlanID:   packagePlanID,
		OrderID:         0, // 需要从订单系统获取
		Status:          paymodels.UserSubscriptionStatusActive,
		StartTime:       receipt.CreationDate,
		EndTime:         *receipt.ExpirationDate,
		AutoRenew:       true,
		PaymentMethod:   paymentMethod,
		PaymentProvider: paymentProvider,
		ProviderSubID:   receipt.ReceiptData,
		Amount:          0, // 需要从产品信息中获取
		Currency:        "USD",
		QuotaLimit:      1000, // 默认额度，实际应用中应该从套餐计划中获取
		QuotaUsed:       0,
		MaxRoles:        2, // 默认值
		MaxContexts:     5, // 默认值
	}

	return paymodels.CreateUserSubscription(ctx, userSubscription)
}

// GetReceiptValidationHistory 获取收据验证历史
func (s *IAPPersistenceService) GetReceiptValidationHistory(ctx context.Context, userID uint64, limit int) ([]*paymodels.IAPReceiptValidation, error) {
	var validations []*paymodels.IAPReceiptValidation
	err := paymodels.DataBase().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&validations).Error

	return validations, err
}

// GetUserPaymentHistory 获取用户支付历史
func (s *IAPPersistenceService) GetUserPaymentHistory(ctx context.Context, userID uint64, limit int) ([]*paymodels.PaymentRecord, error) {
	return paymodels.GetUserPaymentRecords(ctx, int64(userID), 0, limit)
}

// GetUserSubscriptionHistory 获取用户订阅历史
func (s *IAPPersistenceService) GetUserSubscriptionHistory(ctx context.Context, userID uint64, limit int) ([]*paymodels.UserSubscription, error) {
	return paymodels.GetUserSubscriptionsByUserID(ctx, int64(userID), 0, limit)
}

package pay

import (
	"context"
	"fmt"

	models "github.com/grapery/grapery/models/vippay"
	"github.com/sirupsen/logrus"
)

// IAPServiceFactory IAP服务工厂
type IAPServiceFactory struct {
	config *IAPConfig
	logger *logrus.Logger
}

// NewIAPServiceFactory 创建IAP服务工厂
func NewIAPServiceFactory(config *IAPConfig) *IAPServiceFactory {
	return &IAPServiceFactory{
		config: config,
		logger: logrus.New(),
	}
}

// CreateAppleService 创建Apple IAP服务
func (f *IAPServiceFactory) CreateAppleService() IAPService {
	return NewAppleIAPService(f.config)
}

// CreateGoogleService 创建Google IAP服务
func (f *IAPServiceFactory) CreateGoogleService() IAPService {
	return NewGoogleIAPServiceRefactored(f.config, f.logger)
}

// CreateCompositeService 创建复合IAP服务
func (f *IAPServiceFactory) CreateCompositeService() *CompositeIAPService {
	return &CompositeIAPService{
		appleService:       f.CreateAppleService(),
		googleService:      f.CreateGoogleService(),
		logger:             f.logger,
		persistenceService: NewIAPPersistenceService(),
	}
}

// CreateServiceByPlatform 根据平台创建服务
func (f *IAPServiceFactory) CreateServiceByPlatform(platform IAPPlatform) (IAPService, error) {
	switch platform {
	case IAPPlatformApple:
		return f.CreateAppleService(), nil
	case IAPPlatformGoogle:
		return f.CreateGoogleService(), nil
	default:
		return nil, fmt.Errorf("不支持的平台: %s", platform)
	}
}

// CompositeIAPService 复合IAP服务
type CompositeIAPService struct {
	appleService       IAPService
	googleService      IAPService
	logger             *logrus.Logger
	persistenceService *IAPPersistenceService
}

// GetAppleService 获取Apple服务
func (c *CompositeIAPService) GetAppleService() IAPService {
	return c.appleService
}

// GetGoogleService 获取Google服务
func (c *CompositeIAPService) GetGoogleService() IAPService {
	return c.googleService
}

// GetPlatform 获取平台（复合服务不实现此方法）
func (c *CompositeIAPService) GetPlatform() IAPPlatform {
	return "" // 复合服务不返回特定平台
}

// VerifyReceipt 验证收据（自动检测平台）
func (c *CompositeIAPService) VerifyReceipt(ctx context.Context, receipt string, sandbox bool) (*IAPReceipt, error) {
	c.logger.WithFields(logrus.Fields{
		"receipt_length": len(receipt),
		"sandbox":        sandbox,
	}).Info("开始验证收据（自动检测平台）")

	// 1. 检测收据平台
	platform, err := c.detectReceiptPlatform(receipt)
	if err != nil {
		c.logger.WithError(err).Error("检测收据平台失败")
		return nil, fmt.Errorf("检测收据平台失败: %w", err)
	}

	c.logger.WithField("detected_platform", platform).Info("检测到收据平台")

	// 2. 根据检测到的平台选择服务并验证收据
	var service IAPService
	switch platform {
	case IAPPlatformApple:
		service = c.appleService
	case IAPPlatformGoogle:
		service = c.googleService
	default:
		return nil, fmt.Errorf("不支持的收据平台: %s", platform)
	}

	// 3. 验证收据
	receiptResult, err := service.VerifyReceipt(ctx, receipt, sandbox)
	if err != nil {
		c.logger.WithError(err).WithField("platform", platform).Error("收据验证失败")
		return nil, fmt.Errorf("收据验证失败: %w", err)
	}

	// 4. 持久化收据验证结果到数据库
	if err := c.persistenceService.PersistReceiptValidation(ctx, receiptResult); err != nil {
		c.logger.WithError(err).WithFields(logrus.Fields{
			"platform": platform,
			"user_id":  receiptResult.UserID,
		}).Error("持久化收据验证结果失败")
		// 注意：这里不返回错误，因为收据验证已经成功，只是持久化失败
		// 可以根据业务需求决定是否要返回错误
	}

	c.logger.WithFields(logrus.Fields{
		"platform":    platform,
		"user_id":     receiptResult.UserID,
		"status":      receiptResult.Status,
		"environment": receiptResult.Environment,
	}).Info("收据验证成功")

	return receiptResult, nil
}

// VerifyReceiptWithAutoDetection 自动检测环境并验证收据
func (c *CompositeIAPService) VerifyReceiptWithAutoDetection(ctx context.Context, receipt string, platform IAPPlatform) (*IAPReceipt, error) {
	// 1. 自动检测环境
	isSandbox, err := AutoDetectEnvironment(platform, receipt)
	if err != nil {
		// 如果自动检测失败，先尝试生产环境
		isSandbox = false
	}

	// 2. 根据平台选择服务
	var service IAPService
	switch platform {
	case IAPPlatformApple:
		service = c.appleService
	case IAPPlatformGoogle:
		service = c.googleService
	default:
		return nil, fmt.Errorf("不支持的平台: %s", platform)
	}

	// 3. 验证收据
	receiptResult, err := service.VerifyReceipt(ctx, receipt, isSandbox)
	if err != nil {
		// 如果生产环境验证失败，尝试sandbox环境
		if !isSandbox {
			receiptResult, err = service.VerifyReceipt(ctx, receipt, true)
			if err == nil {
				receiptResult.Environment = "Sandbox"
			}
		}
	}

	return receiptResult, err
}

// VerifyAppleReceipt 验证Apple收据
func (c *CompositeIAPService) VerifyAppleReceipt(ctx context.Context, receipt string, sandbox bool) (*IAPReceipt, error) {
	return c.appleService.VerifyReceipt(ctx, receipt, sandbox)
}

// VerifyGoogleReceipt 验证Google收据
func (c *CompositeIAPService) VerifyGoogleReceipt(ctx context.Context, receipt string, sandbox bool) (*IAPReceipt, error) {
	return c.googleService.VerifyReceipt(ctx, receipt, sandbox)
}

// GetSubscription 获取订阅（需要指定平台）
func (c *CompositeIAPService) GetSubscription(ctx context.Context, subscriptionID string) (*IAPSubscription, error) {
	return nil, fmt.Errorf("复合服务需要指定平台获取订阅")
}

// GetAppleSubscription 获取Apple订阅
func (c *CompositeIAPService) GetAppleSubscription(ctx context.Context, subscriptionID string) (*IAPSubscription, error) {
	return c.appleService.GetSubscription(ctx, subscriptionID)
}

// GetGoogleSubscription 获取Google订阅
func (c *CompositeIAPService) GetGoogleSubscription(ctx context.Context, subscriptionID string) (*IAPSubscription, error) {
	return c.googleService.GetSubscription(ctx, subscriptionID)
}

// SyncSubscription 同步订阅（需要指定平台）
func (c *CompositeIAPService) SyncSubscription(ctx context.Context, subscriptionID string) error {
	return fmt.Errorf("复合服务需要指定平台同步订阅")
}

// SyncAppleSubscription 同步Apple订阅
func (c *CompositeIAPService) SyncAppleSubscription(ctx context.Context, subscriptionID string) error {
	return c.appleService.SyncSubscription(ctx, subscriptionID)
}

// SyncGoogleSubscription 同步Google订阅
func (c *CompositeIAPService) SyncGoogleSubscription(ctx context.Context, subscriptionID string) error {
	return c.googleService.SyncSubscription(ctx, subscriptionID)
}

// AcknowledgePurchase 确认购买（需要指定平台）
func (c *CompositeIAPService) AcknowledgePurchase(ctx context.Context, purchaseToken string) error {
	return fmt.Errorf("复合服务需要指定平台确认购买")
}

// AcknowledgeApplePurchase 确认Apple购买
func (c *CompositeIAPService) AcknowledgeApplePurchase(ctx context.Context, purchaseToken string) error {
	return c.appleService.AcknowledgePurchase(ctx, purchaseToken)
}

// AcknowledgeGooglePurchase 确认Google购买
func (c *CompositeIAPService) AcknowledgeGooglePurchase(ctx context.Context, purchaseToken string) error {
	return c.googleService.AcknowledgePurchase(ctx, purchaseToken)
}

// ConsumePurchase 消费购买（需要指定平台）
func (c *CompositeIAPService) ConsumePurchase(ctx context.Context, purchaseToken string) error {
	return fmt.Errorf("复合服务需要指定平台消费购买")
}

// ConsumeApplePurchase 消费Apple购买
func (c *CompositeIAPService) ConsumeApplePurchase(ctx context.Context, purchaseToken string) error {
	return c.appleService.ConsumePurchase(ctx, purchaseToken)
}

// ConsumeGooglePurchase 消费Google购买
func (c *CompositeIAPService) ConsumeGooglePurchase(ctx context.Context, purchaseToken string) error {
	return c.googleService.ConsumePurchase(ctx, purchaseToken)
}

// HandleNotification 处理通知（需要指定平台）
func (c *CompositeIAPService) HandleNotification(ctx context.Context, notification *IAPNotification) error {
	switch notification.Platform {
	case IAPPlatformApple:
		return c.appleService.HandleNotification(ctx, notification)
	case IAPPlatformGoogle:
		return c.googleService.HandleNotification(ctx, notification)
	default:
		return fmt.Errorf("不支持的平台: %s", notification.Platform)
	}
}

// HandleAppleNotification 处理Apple通知
func (c *CompositeIAPService) HandleAppleNotification(ctx context.Context, notification *IAPNotification) error {
	return c.appleService.HandleNotification(ctx, notification)
}

// HandleGoogleNotification 处理Google通知
func (c *CompositeIAPService) HandleGoogleNotification(ctx context.Context, notification *IAPNotification) error {
	return c.googleService.HandleNotification(ctx, notification)
}

// SyncProducts 同步产品（同步所有平台）
func (c *CompositeIAPService) SyncProducts(ctx context.Context) error {
	c.logger.Info("开始同步所有平台的产品")

	// 同步Apple产品
	if err := c.appleService.SyncProducts(ctx); err != nil {
		c.logger.WithError(err).Error("同步Apple产品失败")
		return fmt.Errorf("同步Apple产品失败: %w", err)
	}

	// 同步Google产品
	if err := c.googleService.SyncProducts(ctx); err != nil {
		c.logger.WithError(err).Error("同步Google产品失败")
		return fmt.Errorf("同步Google产品失败: %w", err)
	}

	c.logger.Info("所有平台产品同步完成")
	return nil
}

// SyncAppleProducts 同步Apple产品
func (c *CompositeIAPService) SyncAppleProducts(ctx context.Context) error {
	return c.appleService.SyncProducts(ctx)
}

// SyncGoogleProducts 同步Google产品
func (c *CompositeIAPService) SyncGoogleProducts(ctx context.Context) error {
	return c.googleService.SyncProducts(ctx)
}

// GetServiceByPlatform 根据平台获取服务
func (c *CompositeIAPService) GetServiceByPlatform(platform IAPPlatform) (IAPService, error) {
	switch platform {
	case IAPPlatformApple:
		return c.appleService, nil
	case IAPPlatformGoogle:
		return c.googleService, nil
	default:
		return nil, fmt.Errorf("不支持的平台: %s", platform)
	}
}

// GetPersistenceService 获取持久化服务
func (c *CompositeIAPService) GetPersistenceService() *IAPPersistenceService {
	return c.persistenceService
}

// PersistPaymentRecord 持久化支付记录
func (c *CompositeIAPService) PersistPaymentRecord(ctx context.Context, receipt *IAPReceipt, productID string, amount int64) error {
	return c.persistenceService.PersistPaymentRecord(ctx, receipt, productID, amount)
}

// PersistUserSubscription 持久化用户订阅
func (c *CompositeIAPService) PersistUserSubscription(ctx context.Context, receipt *IAPReceipt, productID string, packagePlanID uint) error {
	return c.persistenceService.PersistUserSubscription(ctx, receipt, productID, packagePlanID)
}

// GetReceiptValidationHistory 获取收据验证历史
func (c *CompositeIAPService) GetReceiptValidationHistory(ctx context.Context, userID uint64, limit int) ([]*models.IAPReceiptValidation, error) {
	return c.persistenceService.GetReceiptValidationHistory(ctx, userID, limit)
}

// GetUserPaymentHistory 获取用户支付历史
func (c *CompositeIAPService) GetUserPaymentHistory(ctx context.Context, userID uint64, limit int) ([]*models.PaymentRecord, error) {
	return c.persistenceService.GetUserPaymentHistory(ctx, userID, limit)
}

// GetUserSubscriptionHistory 获取用户订阅历史
func (c *CompositeIAPService) GetUserSubscriptionHistory(ctx context.Context, userID uint64, limit int) ([]*models.UserSubscription, error) {
	return c.persistenceService.GetUserSubscriptionHistory(ctx, userID, limit)
}

// detectReceiptPlatform 检测收据平台
func (c *CompositeIAPService) detectReceiptPlatform(receipt string) (IAPPlatform, error) {
	if receipt == "" {
		return "", fmt.Errorf("收据数据为空")
	}

	// 1. 尝试检测Apple收据格式
	if c.isAppleReceipt(receipt) {
		c.logger.Debug("检测到Apple收据格式")
		return IAPPlatformApple, nil
	}

	// 2. 尝试检测Google收据格式
	if c.isGoogleReceipt(receipt) {
		c.logger.Debug("检测到Google收据格式")
		return IAPPlatformGoogle, nil
	}

	// 3. 如果无法确定平台，尝试通过验证来检测
	return c.detectPlatformByVerification(receipt)
}

// isAppleReceipt 检测是否为Apple收据
func (c *CompositeIAPService) isAppleReceipt(receipt string) bool {
	// Apple收据通常是Base64编码的字符串，长度较长
	// 可以通过以下特征来判断：
	// 1. 长度通常在1000-5000字符之间
	// 2. 是Base64编码格式
	// 3. 可能包含特定的Apple收据标识符

	if len(receipt) < 100 || len(receipt) > 10000 {
		return false
	}

	// 检查是否为有效的Base64编码
	if !c.isValidBase64(receipt) {
		return false
	}

	// 检查是否包含Apple收据的特征标识
	// Apple收据在解码后通常包含特定的ASN.1结构
	// 这里简化处理，实际应用中可能需要更复杂的检测逻辑
	return true
}

// isGoogleReceipt 检测是否为Google收据
func (c *CompositeIAPService) isGoogleReceipt(receipt string) bool {
	// Google收据通常是购买令牌（Purchase Token）
	// 特征：
	// 1. 长度通常在50-200字符之间
	// 2. 包含字母、数字和特殊字符
	// 3. 不是Base64编码格式

	if len(receipt) < 20 || len(receipt) > 500 {
		return false
	}

	// Google购买令牌通常不是Base64编码
	if c.isValidBase64(receipt) {
		return false
	}

	// 检查是否包含Google购买令牌的特征
	// Google购买令牌通常包含特定的字符模式
	return true
}

// isValidBase64 检查字符串是否为有效的Base64编码
func (c *CompositeIAPService) isValidBase64(s string) bool {
	// 简单的Base64格式检查
	// 实际应用中可以使用更严格的验证
	if len(s)%4 != 0 {
		return false
	}

	// 检查是否只包含Base64字符
	for _, char := range s {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '+' || char == '/' || char == '=') {
			return false
		}
	}

	return true
}

// detectPlatformByVerification 通过验证来检测平台
func (c *CompositeIAPService) detectPlatformByVerification(receipt string) (IAPPlatform, error) {
	c.logger.Debug("通过验证检测收据平台")

	// 创建上下文用于验证
	ctx := context.Background()

	// 1. 先尝试Apple验证（生产环境）
	_, err := c.appleService.VerifyReceipt(ctx, receipt, false)
	if err == nil {
		c.logger.Debug("通过Apple生产环境验证检测到平台")
		return IAPPlatformApple, nil
	}

	// 2. 尝试Apple验证（沙盒环境）
	_, err = c.appleService.VerifyReceipt(ctx, receipt, true)
	if err == nil {
		c.logger.Debug("通过Apple沙盒环境验证检测到平台")
		return IAPPlatformApple, nil
	}

	// 3. 尝试Google验证（生产环境）
	_, err = c.googleService.VerifyReceipt(ctx, receipt, false)
	if err == nil {
		c.logger.Debug("通过Google生产环境验证检测到平台")
		return IAPPlatformGoogle, nil
	}

	// 4. 尝试Google验证（沙盒环境）
	_, err = c.googleService.VerifyReceipt(ctx, receipt, true)
	if err == nil {
		c.logger.Debug("通过Google沙盒环境验证检测到平台")
		return IAPPlatformGoogle, nil
	}

	// 5. 如果所有验证都失败，返回错误
	return "", fmt.Errorf("无法确定收据平台，所有验证尝试都失败")
}

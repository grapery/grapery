package pay

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/grapery/grapery/models"
)

var (
	ErrProductNotFound      = errors.New("product not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrPaymentNotFound      = errors.New("payment record not found")
	ErrInvalidAmount        = errors.New("invalid amount")
	ErrPaymentFailed        = errors.New("payment failed")
	ErrUserNotVIP           = errors.New("user is not VIP")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrInsufficientStock    = errors.New("insufficient stock")
	ErrOrderExpired         = errors.New("order expired")
	ErrPaymentExpired       = errors.New("payment expired")
	ErrSubscriptionExpired  = errors.New("subscription expired")
)

// paymentServiceImpl 支付服务实现
type paymentServiceImpl struct {
	config    *PaymentConfig
	providers map[models.PaymentMethod]PaymentProvider
}

// NewPaymentService 创建支付服务实例
func NewPaymentService(config *PaymentConfig) (PaymentService, error) {
	service := &paymentServiceImpl{
		config:    config,
		providers: make(map[models.PaymentMethod]PaymentProvider),
	}

	// 初始化支付提供商
	if err := service.initProviders(); err != nil {
		return nil, err
	}

	return service, nil
}

// initProviders 初始化支付提供商
func (s *paymentServiceImpl) initProviders() error {
	// 初始化Stripe
	if s.config.StripeConfig.SecretKey != "" {
		stripeProvider := NewStripeProvider(s.config.StripeConfig.SecretKey)
		s.providers[models.PaymentMethodStripe] = stripeProvider
	}

	// 初始化支付宝
	if s.config.AlipayConfig.AppID != "" {
		alipayProvider := NewAlipayProvider(s.config.AlipayConfig)
		s.providers[models.PaymentMethodAlipay] = alipayProvider
	}

	// 初始化微信支付
	if s.config.WechatPayConfig.AppID != "" {
		wechatConfig := struct {
			AppID  string `json:"app_id"`
			APIKey string `json:"api_key"`
		}{
			AppID:  s.config.WechatPayConfig.AppID,
			APIKey: s.config.WechatPayConfig.APIKey,
		}
		wechatProvider := NewWechatPayProvider(wechatConfig)
		s.providers[models.PaymentMethodWechatPay] = wechatProvider
	}

	// 初始化Apple Pay和Google Pay（暂时使用Stripe作为后端）
	if s.config.ApplePayConfig.BundleID != "" {
		appleProvider := NewStripeProvider(s.config.StripeConfig.SecretKey)
		s.providers[models.PaymentMethodApplePay] = appleProvider
	}

	if s.config.GooglePayConfig.MerchantID != "" {
		googleProvider := NewStripeProvider(s.config.StripeConfig.SecretKey)
		s.providers[models.PaymentMethodGooglePay] = googleProvider
	}

	return nil
}

// 商品管理实现
func (s *paymentServiceImpl) CreateProduct(ctx context.Context, product *models.Product) error {
	return models.CreateProduct(ctx, product)
}

func (s *paymentServiceImpl) GetProduct(ctx context.Context, id uint) (*models.Product, error) {
	return models.GetProduct(ctx, id)
}

func (s *paymentServiceImpl) GetProductBySKU(ctx context.Context, sku string) (*models.Product, error) {
	return models.GetProductBySKU(ctx, sku)
}

func (s *paymentServiceImpl) GetActiveProducts(ctx context.Context) ([]*models.Product, error) {
	return models.GetActiveProducts(ctx)
}

func (s *paymentServiceImpl) GetProductsByType(ctx context.Context, productType models.ProductType) ([]*models.Product, error) {
	return models.GetProductsByType(ctx, productType)
}

func (s *paymentServiceImpl) GetProductsByCategory(ctx context.Context, category string) ([]*models.Product, error) {
	return models.GetProductsByCategory(ctx, category)
}

func (s *paymentServiceImpl) GetHotProducts(ctx context.Context, limit int) ([]*models.Product, error) {
	return models.GetHotProducts(ctx, limit)
}

func (s *paymentServiceImpl) GetRecommendProducts(ctx context.Context, limit int) ([]*models.Product, error) {
	return models.GetRecommendProducts(ctx, limit)
}

func (s *paymentServiceImpl) UpdateProduct(ctx context.Context, product *models.Product) error {
	return models.UpdateProduct(ctx, product)
}

func (s *paymentServiceImpl) DeleteProduct(ctx context.Context, id uint, updatedBy int64) error {
	return models.DeleteProduct(ctx, id, updatedBy)
}

func (s *paymentServiceImpl) IncrementProductSoldCount(ctx context.Context, id uint) error {
	return models.IncrementSoldCount(ctx, id)
}

func (s *paymentServiceImpl) IncrementProductViewCount(ctx context.Context, id uint) error {
	return models.IncrementViewCount(ctx, id)
}

func (s *paymentServiceImpl) CheckProductStock(ctx context.Context, id uint, quantity int) (bool, error) {
	product, err := models.GetProduct(ctx, id)
	if err != nil {
		return false, err
	}
	return product.CheckStock(quantity), nil
}

func (s *paymentServiceImpl) DecreaseProductStock(ctx context.Context, id uint, quantity int) error {
	return models.DecreaseStock(ctx, id, quantity)
}

func (s *paymentServiceImpl) IncreaseProductStock(ctx context.Context, id uint, quantity int) error {
	return models.IncreaseStock(ctx, id, quantity)
}

// SKU管理实现
func (s *paymentServiceImpl) CreateProductSKU(ctx context.Context, sku *models.ProductSKU) error {
	return models.CreateProductSKU(ctx, sku)
}

func (s *paymentServiceImpl) GetProductSKU(ctx context.Context, id uint) (*models.ProductSKU, error) {
	return models.GetProductSKU(ctx, id)
}

func (s *paymentServiceImpl) GetProductSKUBySKU(ctx context.Context, skuCode string) (*models.ProductSKU, error) {
	return models.GetProductSKUBySKU(ctx, skuCode)
}

func (s *paymentServiceImpl) GetProductSKUs(ctx context.Context, productID uint) ([]*models.ProductSKU, error) {
	return models.GetProductSKUs(ctx, productID)
}

func (s *paymentServiceImpl) UpdateProductSKU(ctx context.Context, sku *models.ProductSKU) error {
	return models.UpdateProductSKU(ctx, sku)
}

func (s *paymentServiceImpl) DeleteProductSKU(ctx context.Context, id uint) error {
	return models.DeleteProductSKU(ctx, id)
}

// 订单管理实现
func (s *paymentServiceImpl) CreateOrder(ctx context.Context, userID int64, productID uint, skuID *uint, quantity int, paymentMethod models.PaymentMethod) (*models.Order, error) {
	// 获取商品信息
	product, err := models.GetProduct(ctx, productID)
	if err != nil {
		return nil, ErrProductNotFound
	}

	// 检查库存
	if !product.CheckStock(quantity) {
		return nil, ErrInsufficientStock
	}

	// 确定价格
	var unitPrice int64
	if skuID != nil {
		sku, err := models.GetProductSKU(ctx, *skuID)
		if err != nil {
			return nil, errors.New("sku not found")
		}
		unitPrice = sku.Price
	} else {
		unitPrice = product.Price
	}

	// 计算总金额
	totalAmount := unitPrice * int64(quantity)

	// 设置订单过期时间
	expireTime := time.Now().Add(time.Duration(s.config.OrderExpireTime) * time.Minute)

	// 创建订单
	order := &models.Order{
		UserID:      userID,
		ProductID:   int64(productID),
		SKUID:       skuID,
		Amount:      totalAmount,
		Status:      int(models.OrderStatusPending),
		OrderNo:     s.generateOrderNo(),
		Currency:    s.config.DefaultCurrency,
		Quantity:    quantity,
		UnitPrice:   unitPrice,
		TotalAmount: totalAmount,
		ExpireTime:  &expireTime,
		Description: product.Name,
		Metadata:    fmt.Sprintf(`{"payment_method":%d}`, paymentMethod),
	}

	if err := models.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	// 减少库存
	if err := models.DecreaseStock(ctx, productID, quantity); err != nil {
		return nil, err
	}

	// 增加销售数量
	if err := models.IncrementSoldCount(ctx, productID); err != nil {
		// 记录错误但不影响订单创建
		fmt.Printf("Failed to increment sold count: %v\n", err)
	}

	return order, nil
}

func (s *paymentServiceImpl) GetOrder(ctx context.Context, id uint) (*models.Order, error) {
	return models.GetOrder(ctx, id)
}

func (s *paymentServiceImpl) GetOrderByOrderNo(ctx context.Context, orderNo string) (*models.Order, error) {
	return models.GetOrderByOrderNo(ctx, orderNo)
}

func (s *paymentServiceImpl) GetUserOrders(ctx context.Context, userID int64, offset, limit int) ([]*models.Order, error) {
	return models.GetUserOrders(ctx, userID, offset, limit)
}

func (s *paymentServiceImpl) GetUserOrdersByStatus(ctx context.Context, userID int64, status models.OrderStatus, offset, limit int) ([]*models.Order, error) {
	return models.GetUserOrdersByStatus(ctx, userID, status, offset, limit)
}

func (s *paymentServiceImpl) GetOrderStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	return models.GetOrderStats(ctx, userID)
}

func (s *paymentServiceImpl) UpdateOrderStatus(ctx context.Context, id uint, status models.OrderStatus) error {
	return models.UpdateOrderStatus(ctx, id, status)
}

func (s *paymentServiceImpl) UpdateOrderRefund(ctx context.Context, id uint, refundAmount int64, reason string) error {
	return models.UpdateOrderRefund(ctx, id, refundAmount, reason)
}

func (s *paymentServiceImpl) CancelOrder(ctx context.Context, id uint, reason string) error {
	return models.CancelOrder(ctx, id, reason)
}

func (s *paymentServiceImpl) GetExpiredOrders(ctx context.Context) ([]*models.Order, error) {
	return models.GetExpiredOrders(ctx)
}

// 订单项管理实现
func (s *paymentServiceImpl) CreateOrderItem(ctx context.Context, item *models.OrderItem) error {
	return models.CreateOrderItem(ctx, item)
}

func (s *paymentServiceImpl) GetOrderItems(ctx context.Context, orderID uint) ([]*models.OrderItem, error) {
	return models.GetOrderItems(ctx, orderID)
}

func (s *paymentServiceImpl) GetOrderItem(ctx context.Context, id uint) (*models.OrderItem, error) {
	return models.GetOrderItem(ctx, id)
}

func (s *paymentServiceImpl) UpdateOrderItem(ctx context.Context, item *models.OrderItem) error {
	return models.UpdateOrderItem(ctx, item)
}

func (s *paymentServiceImpl) DeleteOrderItem(ctx context.Context, id uint) error {
	return models.DeleteOrderItem(ctx, id)
}

// 支付管理实现
func (s *paymentServiceImpl) CreatePayment(ctx context.Context, orderID uint, paymentMethod models.PaymentMethod, req *CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// 获取订单信息
	order, err := models.GetOrder(ctx, orderID)
	if err != nil {
		return nil, errors.New("order not found")
	}

	// 检查订单状态
	if order.Status != int(models.OrderStatusPending) {
		return nil, errors.New("order status is not pending")
	}

	// 检查订单是否过期
	if order.ExpireTime != nil && time.Now().After(*order.ExpireTime) {
		return nil, ErrOrderExpired
	}

	// 获取支付提供商
	provider, exists := s.providers[paymentMethod]
	if !exists {
		return nil, fmt.Errorf("payment method %d not supported", paymentMethod)
	}

	// 设置支付过期时间
	expireTime := time.Now().Add(time.Duration(s.config.PaymentExpireTime) * time.Minute)
	if req.ExpireTime != nil {
		expireTime = *req.ExpireTime
	}

	// 创建支付请求
	paymentReq := &CreatePaymentRequest{
		UserID:        req.UserID,
		OrderID:       orderID,
		Amount:        order.TotalAmount,
		Currency:      order.Currency,
		PaymentMethod: paymentMethod,
		Description:   order.Description,
		ReturnURL:     s.config.ReturnURL,
		NotifyURL:     s.config.NotifyURL,
		Metadata:      req.Metadata,
		IPAddress:     req.IPAddress,
		UserAgent:     req.UserAgent,
		DeviceInfo:    req.DeviceInfo,
		ExpireTime:    &expireTime,
		IsTest:        s.config.EnableTestMode,
	}

	// 调用支付提供商
	response, err := provider.CreatePayment(ctx, paymentReq)
	if err != nil {
		return nil, err
	}

	// 创建支付记录
	paymentRecord := &models.PaymentRecord{
		UserID:          req.UserID,
		OrderID:         orderID,
		Amount:          order.TotalAmount,
		Currency:        order.Currency,
		Status:          models.PaymentStatusPending,
		PaymentMethod:   paymentMethod,
		PaymentProvider: provider.GetProviderName(),
		ProviderOrderID: response.ProviderOrderID,
		TransactionID:   response.TransactionID,
		ExpireTime:      &expireTime,
		IPAddress:       req.IPAddress,
		UserAgent:       req.UserAgent,
		DeviceInfo:      req.DeviceInfo,
		IsTest:          s.config.EnableTestMode,
	}

	if err := models.CreatePaymentRecord(ctx, paymentRecord); err != nil {
		return nil, err
	}

	return response, nil
}

func (s *paymentServiceImpl) QueryPaymentStatus(ctx context.Context, orderID uint) (*PaymentStatusResponse, error) {
	// 获取支付记录
	paymentRecords, err := models.GetPaymentRecordsByOrderID(ctx, orderID)
	if err != nil || len(paymentRecords) == 0 {
		return nil, ErrPaymentNotFound
	}

	paymentRecord := paymentRecords[0]
	provider, exists := s.providers[paymentRecord.PaymentMethod]
	if !exists {
		return nil, fmt.Errorf("payment provider not found")
	}

	// 查询支付状态
	status, err := provider.QueryPayment(ctx, paymentRecord.ProviderOrderID)
	if err != nil {
		return nil, err
	}

	// 更新支付记录状态
	if status.Status != paymentRecord.Status {
		if err := models.UpdatePaymentStatus(ctx, paymentRecord.ID, status.Status, status.PaymentTime); err != nil {
			return nil, err
		}
	}

	return status, nil
}

func (s *paymentServiceImpl) ProcessPaymentCallback(ctx context.Context, provider string, callbackData []byte, signature string) error {
	// 根据提供商名称找到对应的支付方式
	var paymentMethod models.PaymentMethod
	for method, p := range s.providers {
		if p.GetProviderName() == provider {
			paymentMethod = method
			break
		}
	}

	if paymentMethod == 0 {
		return fmt.Errorf("unknown payment provider: %s", provider)
	}

	// 验证回调签名
	providerInstance := s.providers[paymentMethod]
	valid, err := providerInstance.VerifyCallback(ctx, callbackData, signature)
	if err != nil || !valid {
		return fmt.Errorf("invalid callback signature")
	}

	// 处理回调
	response, err := providerInstance.HandleCallback(ctx, callbackData)
	if err != nil {
		return err
	}

	// 处理支付成功
	if response.Status == models.PaymentStatusSuccess {
		return s.handlePaymentSuccess(ctx, response)
	}

	return nil
}

func (s *paymentServiceImpl) RefundPayment(ctx context.Context, orderID uint, refundAmount int64, reason string) error {
	// 获取支付记录
	paymentRecords, err := models.GetPaymentRecordsByOrderID(ctx, orderID)
	if err != nil || len(paymentRecords) == 0 {
		return ErrPaymentNotFound
	}

	paymentRecord := paymentRecords[0]
	provider, exists := s.providers[paymentRecord.PaymentMethod]
	if !exists {
		return fmt.Errorf("payment provider not found")
	}

	// 调用退款
	req := &RefundRequest{
		ProviderOrderID: paymentRecord.ProviderOrderID,
		RefundAmount:    refundAmount,
		RefundReason:    reason,
	}

	_, err = provider.Refund(ctx, req)
	if err != nil {
		return err
	}

	// 更新支付记录
	return models.UpdatePaymentRefund(ctx, paymentRecord.ID, refundAmount, reason)
}

func (s *paymentServiceImpl) GetPaymentURL(ctx context.Context, orderID uint, paymentMethod models.PaymentMethod) (string, error) {
	// 获取订单信息
	order, err := models.GetOrder(ctx, orderID)
	if err != nil {
		return "", err
	}

	provider, exists := s.providers[paymentMethod]
	if !exists {
		return "", fmt.Errorf("payment method not supported")
	}

	req := &CreatePaymentRequest{
		OrderID:       orderID,
		Amount:        order.TotalAmount,
		Currency:      order.Currency,
		PaymentMethod: paymentMethod,
		Description:   order.Description,
		ReturnURL:     s.config.ReturnURL,
		NotifyURL:     s.config.NotifyURL,
	}

	return provider.GetPaymentURL(ctx, req)
}

func (s *paymentServiceImpl) GetQRCodeURL(ctx context.Context, orderID uint, paymentMethod models.PaymentMethod) (string, error) {
	// 获取订单信息
	order, err := models.GetOrder(ctx, orderID)
	if err != nil {
		return "", err
	}

	provider, exists := s.providers[paymentMethod]
	if !exists {
		return "", fmt.Errorf("payment method not supported")
	}

	req := &CreatePaymentRequest{
		OrderID:       orderID,
		Amount:        order.TotalAmount,
		Currency:      order.Currency,
		PaymentMethod: paymentMethod,
		Description:   order.Description,
		ReturnURL:     s.config.ReturnURL,
		NotifyURL:     s.config.NotifyURL,
	}

	return provider.GetQRCodeURL(ctx, req)
}

// 支付记录管理实现
func (s *paymentServiceImpl) GetPaymentRecord(ctx context.Context, id uint) (*models.PaymentRecord, error) {
	return models.GetPaymentRecord(ctx, id)
}

func (s *paymentServiceImpl) GetPaymentRecordByTransactionID(ctx context.Context, transactionID string) (*models.PaymentRecord, error) {
	return models.GetPaymentRecordByTransactionID(ctx, transactionID)
}

func (s *paymentServiceImpl) GetPaymentRecordByProviderOrderID(ctx context.Context, providerOrderID string) (*models.PaymentRecord, error) {
	return models.GetPaymentRecordByProviderOrderID(ctx, providerOrderID)
}

func (s *paymentServiceImpl) GetUserPaymentRecords(ctx context.Context, userID int64, offset, limit int) ([]*models.PaymentRecord, error) {
	return models.GetUserPaymentRecords(ctx, userID, offset, limit)
}

func (s *paymentServiceImpl) GetUserPaymentRecordsByStatus(ctx context.Context, userID int64, status models.PaymentStatus, offset, limit int) ([]*models.PaymentRecord, error) {
	return models.GetUserPaymentRecordsByStatus(ctx, userID, status, offset, limit)
}

func (s *paymentServiceImpl) GetUserPaymentRecordsByMethod(ctx context.Context, userID int64, method models.PaymentMethod, offset, limit int) ([]*models.PaymentRecord, error) {
	return models.GetUserPaymentRecordsByMethod(ctx, userID, method, offset, limit)
}

func (s *paymentServiceImpl) GetPaymentRecordsByOrderID(ctx context.Context, orderID uint) ([]*models.PaymentRecord, error) {
	return models.GetPaymentRecordsByOrderID(ctx, orderID)
}

func (s *paymentServiceImpl) UpdatePaymentRetry(ctx context.Context, id uint) error {
	return models.UpdatePaymentRetry(ctx, id)
}

func (s *paymentServiceImpl) UpdatePaymentRisk(ctx context.Context, id uint, riskLevel int, riskScore float64) error {
	return models.UpdatePaymentRisk(ctx, id, riskLevel, riskScore)
}

func (s *paymentServiceImpl) GetPaymentStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	return models.GetPaymentStats(ctx, userID)
}

func (s *paymentServiceImpl) GetPaymentMethodStats(ctx context.Context, userID int64) (map[models.PaymentMethod]map[string]interface{}, error) {
	return models.GetPaymentMethodStats(ctx, userID)
}

func (s *paymentServiceImpl) GetExpiredPayments(ctx context.Context) ([]*models.PaymentRecord, error) {
	return models.GetExpiredPayments(ctx)
}

// 订阅管理实现
func (s *paymentServiceImpl) CreateSubscription(ctx context.Context, userID int64, productID uint, orderID uint) (*models.Subscription, error) {
	// 获取商品信息
	product, err := models.GetProduct(ctx, productID)
	if err != nil {
		return nil, ErrProductNotFound
	}

	// 获取订单信息
	order, err := models.GetOrder(ctx, orderID)
	if err != nil {
		return nil, errors.New("order not found")
	}

	// 计算订阅时间
	now := time.Now()
	var endTime time.Time
	if product.Duration > 0 {
		endTime = now.Add(time.Duration(product.Duration) * time.Second)
	} else {
		endTime = now.AddDate(1, 0, 0) // 默认一年
	}

	// 创建订阅
	subscription := &models.Subscription{
		UserID:          userID,
		ProductID:       productID,
		OrderID:         orderID,
		Status:          models.SubscriptionStatusActive,
		StartTime:       now,
		EndTime:         endTime,
		AutoRenew:       true,
		Amount:          order.Amount,
		Currency:        s.config.DefaultCurrency,
		QuotaLimit:      product.QuotaLimit,
		QuotaUsed:       0,
		MaxRoles:        product.MaxRoles,
		MaxContexts:     product.MaxContexts,
		AvailableModels: product.AvailableModels,
	}

	return subscription, models.CreateSubscription(ctx, subscription)
}

func (s *paymentServiceImpl) GetUserActiveSubscription(ctx context.Context, userID int64) (*models.Subscription, error) {
	return models.GetUserActiveSubscription(ctx, userID)
}

func (s *paymentServiceImpl) GetUserSubscriptions(ctx context.Context, userID int64, offset, limit int) ([]*models.Subscription, error) {
	return models.GetUserSubscriptions(ctx, userID, offset, limit)
}

func (s *paymentServiceImpl) CancelSubscription(ctx context.Context, subscriptionID uint, reason string, canceledBy int64) error {
	return models.CancelSubscription(ctx, subscriptionID, reason, canceledBy)
}

func (s *paymentServiceImpl) UpdateSubscriptionQuota(ctx context.Context, subscriptionID uint, quotaUsed int) error {
	return models.UpdateSubscriptionQuota(ctx, subscriptionID, quotaUsed)
}

func (s *paymentServiceImpl) RenewSubscription(ctx context.Context, subscriptionID uint) error {
	subscription, err := models.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	// 检查订阅是否已过期
	if time.Now().After(subscription.EndTime) {
		return ErrSubscriptionExpired
	}

	// 延长订阅时间
	product, err := models.GetProduct(ctx, subscription.ProductID)
	if err != nil {
		return err
	}

	var extension time.Duration
	if product.Duration > 0 {
		extension = time.Duration(product.Duration) * time.Second
	} else {
		extension = 365 * 24 * time.Hour // 默认一年
	}

	subscription.EndTime = subscription.EndTime.Add(extension)
	return models.UpdateSubscription(ctx, subscription)
}

func (s *paymentServiceImpl) UpgradeSubscription(ctx context.Context, subscriptionID uint, newProductID uint) error {
	subscription, err := models.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return err
	}

	newProduct, err := models.GetProduct(ctx, newProductID)
	if err != nil {
		return ErrProductNotFound
	}

	// 更新订阅信息
	subscription.ProductID = newProductID
	subscription.QuotaLimit = newProduct.QuotaLimit
	subscription.MaxRoles = newProduct.MaxRoles
	subscription.MaxContexts = newProduct.MaxContexts
	subscription.AvailableModels = newProduct.AvailableModels

	return models.UpdateSubscription(ctx, subscription)
}

func (s *paymentServiceImpl) GetExpiredSubscriptions(ctx context.Context) ([]*models.Subscription, error) {
	return models.GetExpiredSubscriptions(ctx)
}

// VIP权限检查实现
func (s *paymentServiceImpl) IsUserVIP(ctx context.Context, userID int64) (bool, error) {
	subscription, err := models.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		return false, nil // 用户没有活跃订阅
	}
	return subscription.Status == models.SubscriptionStatusActive, nil
}

func (s *paymentServiceImpl) GetUserVIPInfo(ctx context.Context, userID int64) (*models.Subscription, error) {
	return models.GetUserActiveSubscription(ctx, userID)
}

func (s *paymentServiceImpl) CheckUserPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	subscription, err := models.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		return false, nil
	}

	switch permission {
	case "use_ai":
		return subscription.Status == models.SubscriptionStatusActive, nil
	case "create_roles":
		// 这里需要查询用户当前角色数量，暂时返回true
		return true, nil
	case "create_contexts":
		// 这里需要查询用户当前上下文数量，暂时返回true
		return true, nil
	default:
		return false, ErrPermissionDenied
	}
}

func (s *paymentServiceImpl) ConsumeUserQuota(ctx context.Context, userID int64, amount int) error {
	subscription, err := models.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		return ErrUserNotVIP
	}

	if subscription.QuotaUsed+amount > subscription.QuotaLimit {
		return errors.New("quota exceeded")
	}

	return models.UpdateSubscriptionQuota(ctx, subscription.ID, subscription.QuotaUsed+amount)
}

func (s *paymentServiceImpl) GetUserQuota(ctx context.Context, userID int64) (used int, limit int, err error) {
	subscription, err := models.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		return 0, 0, ErrUserNotVIP
	}

	return subscription.QuotaUsed, subscription.QuotaLimit, nil
}

func (s *paymentServiceImpl) GetUserMaxRoles(ctx context.Context, userID int64) (int, error) {
	subscription, err := models.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		return 2, nil // 默认值
	}
	return subscription.MaxRoles, nil
}

func (s *paymentServiceImpl) GetUserMaxContexts(ctx context.Context, userID int64) (int, error) {
	subscription, err := models.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		return 5, nil // 默认值
	}
	return subscription.MaxContexts, nil
}

func (s *paymentServiceImpl) GetUserAvailableModels(ctx context.Context, userID int64) ([]string, error) {
	subscription, err := models.GetUserActiveSubscription(ctx, userID)
	if err != nil {
		return []string{"gpt-3.5-turbo"}, nil // 默认模型
	}
	return subscription.GetAvailableModels()
}

// 系统管理实现
func (s *paymentServiceImpl) GetSystemStats(ctx context.Context) (map[string]interface{}, error) {
	// 这里实现系统统计信息
	// 包括总订单数、总支付金额、成功率等
	return map[string]interface{}{
		"total_orders":         0,
		"total_revenue":        0,
		"success_rate":         0.0,
		"active_users":         0,
		"active_subscriptions": 0,
	}, nil
}

func (s *paymentServiceImpl) GetRevenueStats(ctx context.Context, startTime, endTime time.Time) (map[string]interface{}, error) {
	// 这里实现收入统计
	return map[string]interface{}{
		"total_revenue":   0,
		"order_count":     0,
		"avg_order_value": 0.0,
	}, nil
}

func (s *paymentServiceImpl) GetPaymentProviderStats(ctx context.Context) (map[string]interface{}, error) {
	// 这里实现支付提供商统计
	return map[string]interface{}{
		"stripe": map[string]interface{}{
			"total_amount": 0,
			"success_rate": 0.0,
		},
		"alipay": map[string]interface{}{
			"total_amount": 0,
			"success_rate": 0.0,
		},
		"wechat_pay": map[string]interface{}{
			"total_amount": 0,
			"success_rate": 0.0,
		},
	}, nil
}

func (s *paymentServiceImpl) ProcessExpiredOrders(ctx context.Context) error {
	expiredOrders, err := models.GetExpiredOrders(ctx)
	if err != nil {
		return err
	}

	for _, order := range expiredOrders {
		if err := models.UpdateOrderStatus(ctx, order.ID, models.OrderStatusExpired); err != nil {
			fmt.Printf("Failed to update expired order %d: %v\n", order.ID, err)
		}
	}

	return nil
}

func (s *paymentServiceImpl) ProcessExpiredPayments(ctx context.Context) error {
	expiredPayments, err := models.GetExpiredPayments(ctx)
	if err != nil {
		return err
	}

	for _, payment := range expiredPayments {
		if err := models.UpdatePaymentStatus(ctx, payment.ID, models.PaymentStatusExpired, nil); err != nil {
			fmt.Printf("Failed to update expired payment %d: %v\n", payment.ID, err)
		}
	}

	return nil
}

func (s *paymentServiceImpl) ProcessExpiredSubscriptions(ctx context.Context) error {
	expiredSubscriptions, err := models.GetExpiredSubscriptions(ctx)
	if err != nil {
		return err
	}

	for _, subscription := range expiredSubscriptions {
		if err := models.UpdateSubscriptionStatus(ctx, subscription.ID, models.SubscriptionStatusExpired); err != nil {
			fmt.Printf("Failed to update expired subscription %d: %v\n", subscription.ID, err)
		}
	}

	return nil
}

func (s *paymentServiceImpl) ValidatePayment(ctx context.Context, paymentRecord *models.PaymentRecord) (bool, error) {
	// 这里实现支付验证逻辑
	// 包括金额验证、时间验证、风险检查等
	return true, nil
}

func (s *paymentServiceImpl) CalculateRiskScore(ctx context.Context, paymentRecord *models.PaymentRecord) (float64, error) {
	// 这里实现风险评分计算
	// 基于IP地址、设备信息、历史记录等
	return 0.0, nil
}

// 私有方法
func (s *paymentServiceImpl) generateOrderNo() string {
	timestamp := time.Now().Unix()
	random := make([]byte, 4)
	rand.Read(random)
	return fmt.Sprintf("ORDER%d%s", timestamp, hex.EncodeToString(random))
}

func (s *paymentServiceImpl) generateTransactionID() string {
	return uuid.New().String()
}

func (s *paymentServiceImpl) handlePaymentSuccess(ctx context.Context, response *PaymentCallbackResponse) error {
	// 根据第三方订单ID查找支付记录
	paymentRecord, err := models.GetPaymentRecordByProviderOrderID(ctx, response.ProviderOrderID)
	if err != nil {
		return err
	}

	// 更新支付记录状态
	if err := models.UpdatePaymentStatus(ctx, paymentRecord.ID, response.Status, response.PaymentTime); err != nil {
		return err
	}

	// 更新订单状态
	if err := models.UpdateOrderStatus(ctx, paymentRecord.OrderID, models.OrderStatusPaid); err != nil {
		return err
	}

	// 如果是订阅商品，创建订阅
	order, err := models.GetOrder(ctx, paymentRecord.OrderID)
	if err != nil {
		return err
	}

	product, err := models.GetProduct(ctx, uint(order.ProductID))
	if err != nil {
		return err
	}

	if product.ProductType == models.ProductTypeSubscription {
		_, err = s.CreateSubscription(ctx, order.UserID, uint(order.ProductID), paymentRecord.OrderID)
		if err != nil {
			return err
		}
	}

	return nil
}

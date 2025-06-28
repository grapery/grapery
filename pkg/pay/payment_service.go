package pay

import (
	"context"
	"time"

	"github.com/grapery/grapery/models"
)

// PaymentProvider 支付提供商接口
type PaymentProvider interface {
	// 创建支付订单
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*CreatePaymentResponse, error)
	// 查询支付状态
	QueryPayment(ctx context.Context, providerOrderID string) (*PaymentStatusResponse, error)
	// 退款
	Refund(ctx context.Context, req *RefundRequest) (*RefundResponse, error)
	// 处理支付回调
	HandleCallback(ctx context.Context, callbackData []byte) (*PaymentCallbackResponse, error)
	// 获取支付方式名称
	GetProviderName() string
	// 验证回调签名
	VerifyCallback(ctx context.Context, callbackData []byte, signature string) (bool, error)
	// 获取支付链接
	GetPaymentURL(ctx context.Context, req *CreatePaymentRequest) (string, error)
	// 获取二维码链接
	GetQRCodeURL(ctx context.Context, req *CreatePaymentRequest) (string, error)
}

// CreatePaymentRequest 创建支付请求
type CreatePaymentRequest struct {
	UserID        int64                  `json:"user_id"`        // 用户ID
	OrderID       uint                   `json:"order_id"`       // 订单ID
	Amount        int64                  `json:"amount"`         // 金额（分）
	Currency      string                 `json:"currency"`       // 货币类型
	PaymentMethod models.PaymentMethod   `json:"payment_method"` // 支付方式
	Description   string                 `json:"description"`    // 商品描述
	ReturnURL     string                 `json:"return_url"`     // 支付完成返回URL
	NotifyURL     string                 `json:"notify_url"`     // 支付通知URL
	Metadata      map[string]interface{} `json:"metadata"`       // 元数据
	// 新增字段
	IPAddress  string     `json:"ip_address"`  // 客户端IP地址
	UserAgent  string     `json:"user_agent"`  // 用户代理
	DeviceInfo string     `json:"device_info"` // 设备信息
	ExpireTime *time.Time `json:"expire_time"` // 支付过期时间
	IsTest     bool       `json:"is_test"`     // 是否为测试支付
}

// CreatePaymentResponse 创建支付响应
type CreatePaymentResponse struct {
	ProviderOrderID string                 `json:"provider_order_id"` // 第三方订单ID
	TransactionID   string                 `json:"transaction_id"`    // 交易ID
	PaymentURL      string                 `json:"payment_url"`       // 支付链接
	QRCodeURL       string                 `json:"qr_code_url"`       // 二维码链接
	Amount          int64                  `json:"amount"`            // 支付金额
	Currency        string                 `json:"currency"`          // 货币类型
	ExpireTime      *time.Time             `json:"expire_time"`       // 支付过期时间
	Metadata        map[string]interface{} `json:"metadata"`          // 元数据
}

// PaymentStatusResponse 支付状态响应
type PaymentStatusResponse struct {
	ProviderOrderID string                 `json:"provider_order_id"` // 第三方订单ID
	Status          models.PaymentStatus   `json:"status"`            // 支付状态
	Amount          int64                  `json:"amount"`            // 支付金额
	PaymentTime     *time.Time             `json:"payment_time"`      // 支付时间
	TransactionID   string                 `json:"transaction_id"`    // 交易ID
	Metadata        map[string]interface{} `json:"metadata"`          // 元数据
}

// RefundRequest 退款请求
type RefundRequest struct {
	ProviderOrderID string                 `json:"provider_order_id"` // 第三方订单ID
	RefundAmount    int64                  `json:"refund_amount"`     // 退款金额（分）
	RefundReason    string                 `json:"refund_reason"`     // 退款原因
	Metadata        map[string]interface{} `json:"metadata"`          // 元数据
}

// RefundResponse 退款响应
type RefundResponse struct {
	RefundID        string                 `json:"refund_id"`         // 退款ID
	ProviderOrderID string                 `json:"provider_order_id"` // 第三方订单ID
	RefundAmount    int64                  `json:"refund_amount"`     // 退款金额
	RefundTime      *time.Time             `json:"refund_time"`       // 退款时间
	Status          string                 `json:"status"`            // 退款状态
	Metadata        map[string]interface{} `json:"metadata"`          // 元数据
}

// PaymentCallbackResponse 支付回调响应
type PaymentCallbackResponse struct {
	ProviderOrderID string                 `json:"provider_order_id"` // 第三方订单ID
	Status          models.PaymentStatus   `json:"status"`            // 支付状态
	Amount          int64                  `json:"amount"`            // 支付金额
	PaymentTime     *time.Time             `json:"payment_time"`      // 支付时间
	TransactionID   string                 `json:"transaction_id"`    // 交易ID
	Metadata        map[string]interface{} `json:"metadata"`          // 元数据
}

// PaymentService 支付服务接口
type PaymentService interface {
	// 商品管理
	CreateProduct(ctx context.Context, product *models.Product) error
	GetProduct(ctx context.Context, id uint) (*models.Product, error)
	GetProductBySKU(ctx context.Context, sku string) (*models.Product, error)
	GetActiveProducts(ctx context.Context) ([]*models.Product, error)
	GetProductsByType(ctx context.Context, productType models.ProductType) ([]*models.Product, error)
	GetProductsByCategory(ctx context.Context, category string) ([]*models.Product, error)
	GetHotProducts(ctx context.Context, limit int) ([]*models.Product, error)
	GetRecommendProducts(ctx context.Context, limit int) ([]*models.Product, error)
	UpdateProduct(ctx context.Context, product *models.Product) error
	DeleteProduct(ctx context.Context, id uint, updatedBy int64) error
	IncrementProductSoldCount(ctx context.Context, id uint) error
	IncrementProductViewCount(ctx context.Context, id uint) error
	CheckProductStock(ctx context.Context, id uint, quantity int) (bool, error)
	DecreaseProductStock(ctx context.Context, id uint, quantity int) error
	IncreaseProductStock(ctx context.Context, id uint, quantity int) error

	// SKU管理
	CreateProductSKU(ctx context.Context, sku *models.ProductSKU) error
	GetProductSKU(ctx context.Context, id uint) (*models.ProductSKU, error)
	GetProductSKUBySKU(ctx context.Context, skuCode string) (*models.ProductSKU, error)
	GetProductSKUs(ctx context.Context, productID uint) ([]*models.ProductSKU, error)
	UpdateProductSKU(ctx context.Context, sku *models.ProductSKU) error
	DeleteProductSKU(ctx context.Context, id uint) error

	// 订单管理
	CreateOrder(ctx context.Context, userID int64, productID uint, skuID *uint, quantity int, paymentMethod models.PaymentMethod) (*models.Order, error)
	GetOrder(ctx context.Context, id uint) (*models.Order, error)
	GetOrderByOrderNo(ctx context.Context, orderNo string) (*models.Order, error)
	GetUserOrders(ctx context.Context, userID int64, offset, limit int) ([]*models.Order, error)
	GetUserOrdersByStatus(ctx context.Context, userID int64, status models.OrderStatus, offset, limit int) ([]*models.Order, error)
	GetOrderStats(ctx context.Context, userID int64) (map[string]interface{}, error)
	UpdateOrderStatus(ctx context.Context, id uint, status models.OrderStatus) error
	UpdateOrderRefund(ctx context.Context, id uint, refundAmount int64, reason string) error
	CancelOrder(ctx context.Context, id uint, reason string) error
	GetExpiredOrders(ctx context.Context) ([]*models.Order, error)

	// 订单项管理
	CreateOrderItem(ctx context.Context, item *models.OrderItem) error
	GetOrderItems(ctx context.Context, orderID uint) ([]*models.OrderItem, error)
	GetOrderItem(ctx context.Context, id uint) (*models.OrderItem, error)
	UpdateOrderItem(ctx context.Context, item *models.OrderItem) error
	DeleteOrderItem(ctx context.Context, id uint) error

	// 支付管理
	CreatePayment(ctx context.Context, orderID uint, paymentMethod models.PaymentMethod, req *CreatePaymentRequest) (*CreatePaymentResponse, error)
	QueryPaymentStatus(ctx context.Context, orderID uint) (*PaymentStatusResponse, error)
	ProcessPaymentCallback(ctx context.Context, provider string, callbackData []byte, signature string) error
	RefundPayment(ctx context.Context, orderID uint, refundAmount int64, reason string) error
	GetPaymentURL(ctx context.Context, orderID uint, paymentMethod models.PaymentMethod) (string, error)
	GetQRCodeURL(ctx context.Context, orderID uint, paymentMethod models.PaymentMethod) (string, error)

	// 支付记录管理
	GetPaymentRecord(ctx context.Context, id uint) (*models.PaymentRecord, error)
	GetPaymentRecordByTransactionID(ctx context.Context, transactionID string) (*models.PaymentRecord, error)
	GetPaymentRecordByProviderOrderID(ctx context.Context, providerOrderID string) (*models.PaymentRecord, error)
	GetUserPaymentRecords(ctx context.Context, userID int64, offset, limit int) ([]*models.PaymentRecord, error)
	GetUserPaymentRecordsByStatus(ctx context.Context, userID int64, status models.PaymentStatus, offset, limit int) ([]*models.PaymentRecord, error)
	GetUserPaymentRecordsByMethod(ctx context.Context, userID int64, method models.PaymentMethod, offset, limit int) ([]*models.PaymentRecord, error)
	GetPaymentRecordsByOrderID(ctx context.Context, orderID uint) ([]*models.PaymentRecord, error)
	UpdatePaymentRetry(ctx context.Context, id uint) error
	UpdatePaymentRisk(ctx context.Context, id uint, riskLevel int, riskScore float64) error
	GetPaymentStats(ctx context.Context, userID int64) (map[string]interface{}, error)
	GetPaymentMethodStats(ctx context.Context, userID int64) (map[models.PaymentMethod]map[string]interface{}, error)
	GetExpiredPayments(ctx context.Context) ([]*models.PaymentRecord, error)

	// 订阅管理
	CreateSubscription(ctx context.Context, userID int64, productID uint, orderID uint) (*models.Subscription, error)
	GetUserActiveSubscription(ctx context.Context, userID int64) (*models.Subscription, error)
	GetUserSubscriptions(ctx context.Context, userID int64, offset, limit int) ([]*models.Subscription, error)
	CancelSubscription(ctx context.Context, subscriptionID uint, reason string, canceledBy int64) error
	UpdateSubscriptionQuota(ctx context.Context, subscriptionID uint, quotaUsed int) error
	RenewSubscription(ctx context.Context, subscriptionID uint) error
	UpgradeSubscription(ctx context.Context, subscriptionID uint, newProductID uint) error
	GetExpiredSubscriptions(ctx context.Context) ([]*models.Subscription, error)

	// VIP权限检查
	IsUserVIP(ctx context.Context, userID int64) (bool, error)
	GetUserVIPInfo(ctx context.Context, userID int64) (*models.Subscription, error)
	CheckUserPermission(ctx context.Context, userID int64, permission string) (bool, error)
	ConsumeUserQuota(ctx context.Context, userID int64, amount int) error
	GetUserQuota(ctx context.Context, userID int64) (used int, limit int, err error)
	GetUserMaxRoles(ctx context.Context, userID int64) (int, error)
	GetUserMaxContexts(ctx context.Context, userID int64) (int, error)
	GetUserAvailableModels(ctx context.Context, userID int64) ([]string, error)

	// 系统管理
	GetSystemStats(ctx context.Context) (map[string]interface{}, error)
	GetRevenueStats(ctx context.Context, startTime, endTime time.Time) (map[string]interface{}, error)
	GetPaymentProviderStats(ctx context.Context) (map[string]interface{}, error)
	ProcessExpiredOrders(ctx context.Context) error
	ProcessExpiredPayments(ctx context.Context) error
	ProcessExpiredSubscriptions(ctx context.Context) error
	ValidatePayment(ctx context.Context, paymentRecord *models.PaymentRecord) (bool, error)
	CalculateRiskScore(ctx context.Context, paymentRecord *models.PaymentRecord) (float64, error)
}

// PaymentConfig 支付配置
type PaymentConfig struct {
	// 支付提供商配置
	StripeConfig struct {
		SecretKey      string `json:"secret_key"`
		PublishableKey string `json:"publishable_key"`
		WebhookSecret  string `json:"webhook_secret"`
		Currency       string `json:"currency"`
	} `json:"stripe_config"`

	AlipayConfig struct {
		AppID      string `json:"app_id"`
		PrivateKey string `json:"private_key"`
		PublicKey  string `json:"public_key"`
		Gateway    string `json:"gateway"`
		NotifyURL  string `json:"notify_url"`
		ReturnURL  string `json:"return_url"`
	} `json:"alipay_config"`

	WechatPayConfig struct {
		AppID     string `json:"app_id"`
		MchID     string `json:"mch_id"`
		APIKey    string `json:"api_key"`
		CertPath  string `json:"cert_path"`
		KeyPath   string `json:"key_path"`
		NotifyURL string `json:"notify_url"`
	} `json:"wechat_pay_config"`

	ApplePayConfig struct {
		BundleID string `json:"bundle_id"`
		KeyID    string `json:"key_id"`
		KeyPath  string `json:"key_path"`
		TeamID   string `json:"team_id"`
	} `json:"apple_pay_config"`

	GooglePayConfig struct {
		MerchantID string `json:"merchant_id"`
		KeyPath    string `json:"key_path"`
		Gateway    string `json:"gateway"`
	} `json:"google_pay_config"`

	// 通用配置
	DefaultCurrency   string  `json:"default_currency"`    // 默认货币
	ReturnURL         string  `json:"return_url"`          // 支付完成返回URL
	NotifyURL         string  `json:"notify_url"`          // 支付通知URL
	OrderExpireTime   int     `json:"order_expire_time"`   // 订单过期时间（分钟）
	PaymentExpireTime int     `json:"payment_expire_time"` // 支付过期时间（分钟）
	MaxRetryCount     int     `json:"max_retry_count"`     // 最大重试次数
	EnableTestMode    bool    `json:"enable_test_mode"`    // 是否启用测试模式
	EnableRiskCheck   bool    `json:"enable_risk_check"`   // 是否启用风险检查
	RiskThreshold     float64 `json:"risk_threshold"`      // 风险阈值
}

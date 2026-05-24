package pay

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	paymodels "github.com/grapestree/fgrapery/grapery/internal/repository/pay"
	"github.com/google/uuid"
)

// IAPPlatform IAP平台类型
type IAPPlatform string

const (
	IAPPlatformApple  IAPPlatform = "apple"
	IAPPlatformGoogle IAPPlatform = "google"
)

// IAPConfig IAP统一配置
type IAPConfig struct {
	Apple  AppleConfig  `json:"apple"`
	Google GoogleConfig `json:"google"`
}

// AppleConfig Apple IAP配置
type AppleConfig struct {
	BundleID       string `json:"bundle_id"`
	SandboxURL     string `json:"sandbox_url"`
	ProductionURL  string `json:"production_url"`
	IssuerID       string `json:"issuer_id"`
	KeyID          string `json:"key_id"`
	PrivateKey     string `json:"private_key"`
	APIBaseURL     string `json:"api_base_url"`
	TimeoutSeconds int    `json:"timeout_seconds"`
	MaxRetries     int    `json:"max_retries"`
	RetryDelayMs   int    `json:"retry_delay_ms"`
	// Sandbox特定配置
	SandboxBundleID   string `json:"sandbox_bundle_id,omitempty"`   // Sandbox环境的Bundle ID（如果不同）
	SandboxIssuerID   string `json:"sandbox_issuer_id,omitempty"`   // Sandbox环境的Issuer ID（如果不同）
	SandboxKeyID      string `json:"sandbox_key_id,omitempty"`      // Sandbox环境的Key ID（如果不同）
	SandboxPrivateKey string `json:"sandbox_private_key,omitempty"` // Sandbox环境的Private Key（如果不同）
}

// GoogleConfig Google IAP配置
type GoogleConfig struct {
	PackageName       string `json:"package_name"`
	ServiceAccountKey string `json:"service_account_key"`
	APIBaseURL        string `json:"api_base_url"`
	TimeoutSeconds    int    `json:"timeout_seconds"`
	MaxRetries        int    `json:"max_retries"`
	RetryDelayMs      int    `json:"retry_delay_ms"`
	// Sandbox特定配置
	SandboxPackageName       string `json:"sandbox_package_name,omitempty"`        // Sandbox环境的Package Name（如果不同）
	SandboxServiceAccountKey string `json:"sandbox_service_account_key,omitempty"` // Sandbox环境的Service Account Key（如果不同）
}

// GetBundleID 获取Bundle ID（支持sandbox）
func (c *AppleConfig) GetBundleID(sandbox bool) string {
	if sandbox && c.SandboxBundleID != "" {
		return c.SandboxBundleID
	}
	return c.BundleID
}

// GetIssuerID 获取Issuer ID（支持sandbox）
func (c *AppleConfig) GetIssuerID(sandbox bool) string {
	if sandbox && c.SandboxIssuerID != "" {
		return c.SandboxIssuerID
	}
	return c.IssuerID
}

// GetKeyID 获取Key ID（支持sandbox）
func (c *AppleConfig) GetKeyID(sandbox bool) string {
	if sandbox && c.SandboxKeyID != "" {
		return c.SandboxKeyID
	}
	return c.KeyID
}

// GetPrivateKey 获取Private Key（支持sandbox）
func (c *AppleConfig) GetPrivateKey(sandbox bool) string {
	if sandbox && c.SandboxPrivateKey != "" {
		return c.SandboxPrivateKey
	}
	return c.PrivateKey
}

// GetVerificationURL 获取验证URL（支持sandbox）
func (c *AppleConfig) GetVerificationURL(sandbox bool) string {
	if sandbox {
		return c.SandboxURL
	}
	return c.ProductionURL
}

// GetPackageName 获取Package Name（支持sandbox）
func (c *GoogleConfig) GetPackageName(sandbox bool) string {
	if sandbox && c.SandboxPackageName != "" {
		return c.SandboxPackageName
	}
	return c.PackageName
}

// GetServiceAccountKey 获取Service Account Key（支持sandbox）
func (c *GoogleConfig) GetServiceAccountKey(sandbox bool) string {
	if sandbox && c.SandboxServiceAccountKey != "" {
		return c.SandboxServiceAccountKey
	}
	return c.ServiceAccountKey
}

// EnvironmentDetector 环境检测器接口
type EnvironmentDetector interface {
	DetectEnvironment(receipt string) (bool, error) // 返回 (isSandbox, error)
}

// AppleEnvironmentDetector Apple环境检测器
type AppleEnvironmentDetector struct{}

// DetectEnvironment 检测Apple收据的环境
func (d *AppleEnvironmentDetector) DetectEnvironment(receipt string) (bool, error) {
	// Apple收据环境检测逻辑
	// 1. 首先尝试生产环境验证
	// 2. 如果返回21007错误码，则说明是sandbox环境
	// 这里简化实现，实际应用中需要调用Apple的验证API
	return false, nil // 默认返回生产环境
}

// GoogleEnvironmentDetector Google环境检测器
type GoogleEnvironmentDetector struct{}

// DetectEnvironment 检测Google收据的环境
func (d *GoogleEnvironmentDetector) DetectEnvironment(receipt string) (bool, error) {
	// Google收据环境检测逻辑
	// Google的sandbox环境通常通过测试用户账户来区分
	// 这里简化实现，实际应用中需要根据具体的Google API响应来判断
	return false, nil // 默认返回生产环境
}

// AutoDetectEnvironment 自动检测环境
func AutoDetectEnvironment(platform IAPPlatform, receipt string) (bool, error) {
	var detector EnvironmentDetector
	switch platform {
	case IAPPlatformApple:
		detector = &AppleEnvironmentDetector{}
	case IAPPlatformGoogle:
		detector = &GoogleEnvironmentDetector{}
	default:
		return false, fmt.Errorf("不支持的平台: %s", platform)
	}

	return detector.DetectEnvironment(receipt)
}

// IAPService IAP服务统一接口
type IAPService interface {
	// 平台信息
	GetPlatform() IAPPlatform

	// 收据验证
	VerifyReceipt(ctx context.Context, receipt string, sandbox bool) (*IAPReceipt, error)

	// 订阅管理
	GetSubscription(ctx context.Context, subscriptionID string) (*IAPSubscription, error)
	SyncSubscription(ctx context.Context, subscriptionID string) error

	// 购买管理
	AcknowledgePurchase(ctx context.Context, purchaseToken string) error
	ConsumePurchase(ctx context.Context, purchaseToken string) error

	// 通知处理
	HandleNotification(ctx context.Context, notification *IAPNotification) error

	// 产品同步
	SyncProducts(ctx context.Context) error
}

// IAPReceipt IAP收据统一结构
type IAPReceipt struct {
	ID                 uint        `json:"id"`
	UserID             uint64      `json:"user_id"`
	Platform           IAPPlatform `json:"platform"`
	ReceiptData        string      `json:"receipt_data"`
	BundleID           string      `json:"bundle_id"`
	ProductID          string      `json:"product_id"`
	// OriginalTransactionID Apple 订阅原始交易 ID（同一订阅链路不变）。
	OriginalTransactionID string `json:"original_transaction_id,omitempty"`
	// SubscriptionTransactionID 当前订阅周期对应的交易 ID（含续费）；用于按期重置点数幂等。
	SubscriptionTransactionID string `json:"subscription_transaction_id,omitempty"`
	ApplicationVersion string      `json:"application_version"`
	CreationDate       time.Time   `json:"creation_date"`
	ExpirationDate     *time.Time  `json:"expiration_date"`
	Environment        string      `json:"environment"`
	Status             string      `json:"status"`
	VerificationHash   string      `json:"verification_hash"`
	CreatedAt          time.Time   `json:"created_at"`
	UpdatedAt          time.Time   `json:"updated_at"`
}

// IAPSubscription IAP订阅统一结构
type IAPSubscription struct {
	ID                     uint        `json:"id"`
	UserID                 uint64      `json:"user_id"`
	Platform               IAPPlatform `json:"platform"`
	SubscriptionID         string      `json:"subscription_id"`
	ProductID              string      `json:"product_id"`
	PurchaseDate           time.Time   `json:"purchase_date"`
	ExpiresDate            *time.Time  `json:"expires_date"`
	Status                 string      `json:"status"`
	AutoRenewStatus        string      `json:"auto_renew_status"`
	IsInIntroOfferPeriod   bool        `json:"is_in_intro_offer_period"`
	IsInGracePeriod        bool        `json:"is_in_grace_period"`
	GracePeriodExpiresDate *time.Time  `json:"grace_period_expires_date"`
	OfferType              string      `json:"offer_type"`
	PriceIncreaseStatus    string      `json:"price_increase_status"`
	LastNotificationType   string      `json:"last_notification_type"`
	LastNotificationDate   *time.Time  `json:"last_notification_date"`
	CreatedAt              time.Time   `json:"created_at"`
	UpdatedAt              time.Time   `json:"updated_at"`
}

// IAPPurchase IAP购买统一结构
type IAPPurchase struct {
	ID               uint        `json:"id"`
	UserID           uint64      `json:"user_id"`
	Platform         IAPPlatform `json:"platform"`
	PurchaseToken    string      `json:"purchase_token"`
	ProductID        string      `json:"product_id"`
	PackageName      string      `json:"package_name"`
	PurchaseTime     time.Time   `json:"purchase_time"`
	PurchaseState    int         `json:"purchase_state"`
	ConsumptionState int         `json:"consumption_state"`
	DeveloperPayload string      `json:"developer_payload"`
	OrderID          string      `json:"order_id"`
	PurchaseType     string      `json:"purchase_type"`
	Acknowledged     bool        `json:"acknowledged"`
	RawData          string      `json:"raw_data"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// IAPNotification IAP通知统一结构
type IAPNotification struct {
	ID               uint        `json:"id"`
	Platform         IAPPlatform `json:"platform"`
	NotificationID   string      `json:"notification_id"`
	NotificationType string      `json:"notification_type"`
	Subtype          string      `json:"subtype"`
	Version          string      `json:"version"`
	SignedPayload    string      `json:"signed_payload"`
	RawData          string      `json:"raw_data"`
	ProcessedAt      time.Time   `json:"processed_at"`
	Status           string      `json:"status"`
	ErrorMessage     string      `json:"error_message"`
	SubscriptionID   string      `json:"subscription_id"`   // 订阅ID（Google使用）
	EventTime        time.Time   `json:"event_time"`        // 事件时间（Google使用）
	EventTimeMillis  int64       `json:"event_time_millis"` // 事件时间戳（Google使用）
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// IAPReceiptValidation IAP收据验证记录
type IAPReceiptValidation struct {
	ID               uint        `json:"id"`
	UserID           uint64      `json:"user_id"`
	Platform         IAPPlatform `json:"platform"`
	ReceiptData      string      `json:"receipt_data"`
	ValidationStatus string      `json:"validation_status"`
	ErrorMessage     string      `json:"error_message"`
	ResponseData     string      `json:"response_data"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// IAPSubscriptionSync IAP订阅同步记录
type IAPSubscriptionSync struct {
	ID             uint        `json:"id"`
	UserID         uint64      `json:"user_id"`
	Platform       IAPPlatform `json:"platform"`
	SubscriptionID string      `json:"subscription_id"`
	SyncType       string      `json:"sync_type"`
	SyncStatus     string      `json:"sync_status"`
	ErrorMessage   string      `json:"error_message"`
	PreviousStatus string      `json:"previous_status"`
	NewStatus      string      `json:"new_status"`
	LastSyncTime   time.Time   `json:"last_sync_time"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// ApplePublicKeyCache Apple公钥缓存
type ApplePublicKeyCache struct {
	Keys      map[string]*rsa.PublicKey
	ExpiresAt time.Time
	mu        sync.RWMutex
}

// AppleJWKSResponse Apple JWKS 响应结构
type AppleJWKSResponse struct {
	Keys []AppleJWK `json:"keys"`
}

// AppleJWK Apple JWK 结构
type AppleJWK struct {
	Kty string `json:"kty"` // Key Type
	Kid string `json:"kid"` // Key ID
	Use string `json:"use"` // Public Key Use
	Alg string `json:"alg"` // Algorithm
	N   string `json:"n"`   // RSA Modulus
	E   string `json:"e"`   // RSA Exponent
}

// AppleTransactionInfo Apple交易信息
type AppleTransactionInfo struct {
	OriginalTransactionID string     `json:"original_transaction_id"`
	TransactionID         string     `json:"transaction_id"`
	ProductID             string     `json:"product_id"`
	PurchaseDate          time.Time  `json:"purchase_date"`
	ExpiresDate           *time.Time `json:"expires_date"`
	IsInIntroOfferPeriod  bool       `json:"is_in_intro_offer_period"`
	IsInGracePeriod       bool       `json:"is_in_grace_period"`
	AutoRenewStatus       string     `json:"auto_renew_status"`
}

// parseAppleTimestamp 解析Apple时间戳
func parseAppleTimestamp(timestamp string) (time.Time, error) {
	// 检查空字符串
	if timestamp == "" {
		return time.Time{}, nil
	}

	// Apple 使用 RFC 3339 格式的时间戳，支持多种格式：
	// 1. 标准 RFC 3339 格式: "2023-12-01T10:30:00Z"
	// 2. 带毫秒的格式: "2023-12-01T10:30:00.123Z"
	// 3. 带时区的格式: "2023-12-01T10:30:00+00:00"
	// 4. 毫秒级时间戳: "1701427800000"

	// 首先尝试解析为毫秒级时间戳（纯数字）
	if isNumeric(timestamp) {
		// 尝试解析为毫秒级时间戳
		if millis, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
			// 检查时间戳长度来判断是秒还是毫秒
			if len(timestamp) == 13 {
				// 13位数字，毫秒级时间戳
				return time.Unix(0, millis*int64(time.Millisecond)), nil
			} else if len(timestamp) == 10 {
				// 10位数字，秒级时间戳
				return time.Unix(millis, 0), nil
			}
		}
	}

	// 尝试解析 RFC 3339 格式
	// 支持多种 RFC 3339 格式
	formats := []string{
		time.RFC3339,                     // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,                 // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05Z",           // "2006-01-02T15:04:05Z"
		"2006-01-02T15:04:05.000Z",       // "2006-01-02T15:04:05.000Z"
		"2006-01-02T15:04:05.000000Z",    // "2006-01-02T15:04:05.000000Z"
		"2006-01-02T15:04:05.000000000Z", // "2006-01-02T15:04:05.000000000Z"
		"2006-01-02T15:04:05+00:00",      // "2006-01-02T15:04:05+00:00"
		"2006-01-02T15:04:05.000+00:00",  // "2006-01-02T15:04:05.000+00:00"
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestamp); err == nil {
			return t, nil
		}
	}

	// 如果所有格式都解析失败，返回错误
	return time.Time{}, fmt.Errorf("无法解析Apple时间戳格式: %s", timestamp)
}

// isNumeric 检查字符串是否为纯数字
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, char := range s {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// AppleNotificationData Apple通知数据
type AppleNotificationData struct {
	NotificationUUID string `json:"notificationUUID"`
	NotificationType string `json:"notificationType"`
	Subtype          string `json:"subtype"`
}

// GoogleNotificationData Google通知数据
type GoogleNotificationData struct {
	Version                  string `json:"version"`
	NotificationType         string `json:"notificationType"`
	EventTimeMillis          int64  `json:"eventTimeMillis"`
	SubscriptionNotification struct {
		SubscriptionID string `json:"subscriptionId"`
	} `json:"subscriptionNotification"`
}

// ParseAppleNotification 解析 App Store Server Notifications V2 的 signedPayload（JWS payload 段）。
func ParseAppleNotification(signedPayload string) (*AppleNotificationData, error) {
	signedPayload = strings.TrimSpace(signedPayload)
	if signedPayload == "" {
		return nil, fmt.Errorf("empty signed payload")
	}
	parts := strings.Split(signedPayload, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWS format")
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode JWS payload: %w", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal JWS payload: %w", err)
	}

	out := &AppleNotificationData{}
	if v, ok := payload["notificationUUID"].(string); ok {
		out.NotificationUUID = v
	}
	if v, ok := payload["notificationType"].(string); ok {
		out.NotificationType = v
	}
	if v, ok := payload["subtype"].(string); ok {
		out.Subtype = v
	}
	if out.NotificationUUID == "" {
		out.NotificationUUID = uuid.New().String()
	}
	if out.NotificationType == "" {
		return nil, fmt.Errorf("missing notificationType in signed payload")
	}
	return out, nil
}

// ParseGoogleNotification 解析Google通知
func ParseGoogleNotification(data string) (*GoogleNotificationData, error) {
	// 简化的Google通知解析实现
	// 实际应用中需要解析JSON数据
	return &GoogleNotificationData{
		Version:          "1.0",
		NotificationType: "SUBSCRIPTION_PURCHASED",
		EventTimeMillis:  time.Now().UnixMilli(),
	}, nil
}

// ConvertToAppleReceipt 转换为Apple收据模型
func (r *IAPReceipt) ConvertToAppleReceipt() *paymodels.AppleReceipt {
	return &paymodels.AppleReceipt{
		ID:                 r.ID,
		UserID:             r.UserID,
		ReceiptData:        r.ReceiptData,
		BundleID:           r.BundleID,
		ApplicationVersion: r.ApplicationVersion,
		CreationDate:       r.CreationDate,
		ExpirationDate:     r.ExpirationDate,
		Environment:        r.Environment,
		Status:             r.Status,
		VerificationHash:   r.VerificationHash,
		CreatedAt:          r.CreatedAt,
		UpdatedAt:          r.UpdatedAt,
	}
}

// ConvertToGooglePurchase 转换为Google购买模型
func (p *IAPPurchase) ConvertToGooglePurchase() *paymodels.GooglePurchase {
	return &paymodels.GooglePurchase{
		ID:               p.ID,
		UserID:           p.UserID,
		PurchaseToken:    p.PurchaseToken,
		ProductID:        p.ProductID,
		PackageName:      p.PackageName,
		PurchaseTime:     p.PurchaseTime,
		PurchaseState:    p.PurchaseState,
		ConsumptionState: p.ConsumptionState,
		DeveloperPayload: p.DeveloperPayload,
		OrderID:          p.OrderID,
		PurchaseType:     p.PurchaseType,
		Acknowledged:     p.Acknowledged,
		RawData:          p.RawData,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

// ConvertToAppleSubscription 转换为Apple订阅模型
func (s *IAPSubscription) ConvertToAppleSubscription() *paymodels.AppleSubscription {
	return &paymodels.AppleSubscription{
		ID:                     s.ID,
		UserID:                 s.UserID,
		OriginalTransactionID:  s.SubscriptionID,
		ProductID:              s.ProductID,
		PurchaseDate:           s.PurchaseDate,
		ExpiresDate:            s.ExpiresDate,
		Status:                 s.Status,
		AutoRenewStatus:        s.AutoRenewStatus,
		IsInIntroOfferPeriod:   s.IsInIntroOfferPeriod,
		IsInGracePeriod:        s.IsInGracePeriod,
		GracePeriodExpiresDate: s.GracePeriodExpiresDate,
		OfferType:              s.OfferType,
		PriceIncreaseStatus:    s.PriceIncreaseStatus,
		LastNotificationType:   s.LastNotificationType,
		LastNotificationDate:   s.LastNotificationDate,
		CreatedAt:              s.CreatedAt,
		UpdatedAt:              s.UpdatedAt,
	}
}

// ConvertToGoogleSubscription 转换为Google订阅模型
func (s *IAPSubscription) ConvertToGoogleSubscription() *paymodels.GoogleSubscription {
	var expiryTime time.Time
	if s.ExpiresDate != nil {
		expiryTime = *s.ExpiresDate
	} else {
		expiryTime = s.PurchaseDate.Add(30 * 24 * time.Hour) // 默认30天后过期
	}

	return &paymodels.GoogleSubscription{
		ID:                   s.ID,
		UserID:               s.UserID,
		PurchaseToken:        s.SubscriptionID,
		ProductID:            s.ProductID,
		StartTime:            s.PurchaseDate,
		ExpiryTime:           expiryTime,
		AutoRenewing:         s.AutoRenewStatus == "On",
		PaymentState:         1, // Payment received
		LastNotificationType: s.LastNotificationType,
		LastNotificationDate: s.LastNotificationDate,
		CreatedAt:            s.CreatedAt,
		UpdatedAt:            s.UpdatedAt,
	}
}

// ConvertToAppleSubscription 转换为Apple订阅模型
func (r *IAPReceipt) ConvertToAppleSubscription() *paymodels.AppleSubscription {
	return &paymodels.AppleSubscription{
		UserID:                r.UserID,
		OriginalTransactionID: r.ReceiptData, // 使用收据数据作为事务ID
		ProductID:             "",            // 需要从其他地方获取
		PurchaseDate:          r.CreationDate,
		ExpiresDate:           r.ExpirationDate,
		Status:                r.Status,
		AutoRenewStatus:       "On", // 默认值
		CreatedAt:             r.CreatedAt,
		UpdatedAt:             r.UpdatedAt,
	}
}

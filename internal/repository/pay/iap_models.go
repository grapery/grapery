package pay

import (
	"time"

	"gorm.io/gorm"
)

// AppleReceipt Apple 收据记录
type AppleReceipt struct {
	ID                         uint           `gorm:"primaryKey" json:"id"`
	UserID                     uint64         `gorm:"index" json:"user_id"`
	ReceiptData                string         `gorm:"type:text" json:"receipt_data"`
	OriginalAppVersion         string         `gorm:"type:varchar(255)" json:"original_app_version"`
	BundleID                   string         `gorm:"type:varchar(255)" json:"bundle_id"`
	ApplicationVersion         string         `gorm:"type:varchar(255)" json:"application_version"`
	OriginalApplicationVersion string         `gorm:"type:varchar(255)" json:"original_application_version"`
	CreationDate               time.Time      `gorm:"index" json:"creation_date"`
	ExpirationDate             *time.Time     `gorm:"index" json:"expiration_date"`
	Environment                string         `gorm:"type:varchar(50);index" json:"environment"` // Sandbox, Production
	Status                     string         `gorm:"type:varchar(50);index" json:"status"`      // Valid, Expired, Invalid
	VerificationHash           string         `gorm:"type:varchar(255)" json:"verification_hash"`
	CreatedAt                  time.Time      `json:"created_at"`
	UpdatedAt                  time.Time      `json:"updated_at"`
	DeletedAt                  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// AppleSubscription Apple 订阅记录
type AppleSubscription struct {
	ID                     uint           `gorm:"primaryKey" json:"id"`
	UserID                 uint64         `gorm:"index" json:"user_id"`
	OriginalTransactionID  string         `gorm:"type:varchar(255);uniqueIndex" json:"original_transaction_id"`
	ProductID              string         `gorm:"type:varchar(255);index" json:"product_id"`
	PurchaseDate           time.Time      `gorm:"index" json:"purchase_date"`
	ExpiresDate            *time.Time     `gorm:"index" json:"expires_date"`
	Status                 string         `gorm:"type:varchar(50);index" json:"status"`      // Active, Expired, Canceled, Revoked
	AutoRenewStatus        string         `gorm:"type:varchar(50)" json:"auto_renew_status"` // On, Off
	IsInIntroOfferPeriod   bool           `gorm:"default:false" json:"is_in_intro_offer_period"`
	IsInGracePeriod        bool           `gorm:"default:false" json:"is_in_grace_period"`
	GracePeriodExpiresDate *time.Time     `json:"grace_period_expires_date"`
	OfferType              string         `gorm:"type:varchar(100)" json:"offer_type"`
	PriceIncreaseStatus    string         `gorm:"type:varchar(100)" json:"price_increase_status"`
	LastNotificationType   string         `gorm:"type:varchar(100)" json:"last_notification_type"`
	LastNotificationDate   *time.Time     `json:"last_notification_date"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// AppleNotification Apple 通知记录
type AppleNotification struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	NotificationID   string         `gorm:"type:varchar(255);uniqueIndex" json:"notification_id"`
	NotificationType string         `gorm:"type:varchar(100);index" json:"notification_type"`
	Subtype          string         `gorm:"type:varchar(100)" json:"subtype"`
	Version          string         `gorm:"type:varchar(50)" json:"version"`
	SignedPayload    string         `gorm:"type:text" json:"signed_payload"`
	RawData          string         `gorm:"type:text" json:"raw_data"`
	ProcessedAt      time.Time      `gorm:"index" json:"processed_at"`
	Status           string         `gorm:"type:varchar(50);index" json:"status"` // Success, Failed, Pending
	ErrorMessage     string         `gorm:"type:text" json:"error_message"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// GooglePurchase Google 购买记录
type GooglePurchase struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint64         `gorm:"index" json:"user_id"`
	PurchaseToken    string         `gorm:"type:varchar(255);uniqueIndex" json:"purchase_token"`
	ProductID        string         `gorm:"type:varchar(255);index" json:"product_id"`
	PackageName      string         `gorm:"type:varchar(255);index" json:"package_name"`
	PurchaseTime     time.Time      `gorm:"index" json:"purchase_time"`
	PurchaseState    int            `gorm:"type:smallint;index" json:"purchase_state"`    // 0: Purchased, 1: Canceled
	ConsumptionState int            `gorm:"type:smallint;index" json:"consumption_state"` // 0: Yet to be consumed, 1: Consumed
	DeveloperPayload string         `gorm:"type:text" json:"developer_payload"`
	OrderID          string         `gorm:"type:varchar(255);index" json:"order_id"`
	PurchaseType     string         `gorm:"type:varchar(50);index" json:"purchase_type"` // test, promo
	Acknowledged     bool           `gorm:"default:false" json:"acknowledged"`
	RawData          string         `gorm:"type:text" json:"raw_data"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// GoogleSubscription Google 订阅记录
type GoogleSubscription struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	UserID               uint64         `gorm:"index" json:"user_id"`
	PurchaseToken        string         `gorm:"type:varchar(255);uniqueIndex" json:"purchase_token"`
	ProductID            string         `gorm:"type:varchar(255);index" json:"product_id"`
	PackageName          string         `gorm:"type:varchar(255);index" json:"package_name"`
	StartTime            time.Time      `gorm:"index" json:"start_time"`
	ExpiryTime           time.Time      `gorm:"index" json:"expiry_time"`
	AutoRenewing         bool           `gorm:"default:true" json:"auto_renewing"`
	PriceAmountMicros    int64          `gorm:"type:bigint" json:"price_amount_micros"`
	PriceCurrencyCode    string         `gorm:"type:varchar(10)" json:"price_currency_code"`
	CountryCode          string         `gorm:"type:varchar(10)" json:"country_code"`
	PaymentState         int            `gorm:"type:smallint;index" json:"payment_state"` // 0: Payment pending, 1: Payment received, 2: Free trial, 3: Deferred
	CancelReason         int            `gorm:"type:smallint" json:"cancel_reason"`
	UserCancellationTime *time.Time     `json:"user_cancellation_time"`
	GracePeriodEndTime   *time.Time     `json:"grace_period_end_time"`
	RetryTime            *time.Time     `json:"retry_time"`
	AccountHoldTime      *time.Time     `json:"account_hold_time"`
	PauseStartTime       *time.Time     `json:"pause_start_time"`
	PauseDurationTime    *time.Time     `json:"pause_duration_time"`
	AutoResumeTime       *time.Time     `json:"auto_resume_time"`
	LastNotificationType string         `gorm:"type:varchar(100)" json:"last_notification_type"`
	LastNotificationDate *time.Time     `json:"last_notification_date"`
	RawData              string         `gorm:"type:text" json:"raw_data"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// GoogleNotification Google 通知记录
type GoogleNotification struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	NotificationID   string         `gorm:"type:varchar(255);uniqueIndex" json:"notification_id"`
	Version          string         `gorm:"type:varchar(50)" json:"version"`
	NotificationType string         `gorm:"type:varchar(100);index" json:"notification_type"`
	EventTimeMillis  int64          `gorm:"index" json:"event_time_millis"`
	SubscriptionID   string         `gorm:"type:varchar(255);index" json:"subscription_id"`
	PackageName      string         `gorm:"type:varchar(255);index" json:"package_name"`
	EventTime        time.Time      `gorm:"index" json:"event_time"`
	RawData          string         `gorm:"type:text" json:"raw_data"`
	ProcessedAt      time.Time      `gorm:"index" json:"processed_at"`
	Status           string         `gorm:"type:varchar(50);index" json:"status"` // Success, Failed, Pending
	ErrorMessage     string         `gorm:"type:text" json:"error_message"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// IAPReceiptValidation IAP 收据验证记录
type IAPReceiptValidation struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint64         `gorm:"index" json:"user_id"`
	Platform         string         `gorm:"type:varchar(50);index" json:"platform"` // apple, google
	ReceiptData      string         `gorm:"type:text" json:"receipt_data"`
	ValidationStatus string         `gorm:"type:varchar(50);index" json:"validation_status"` // success, failed, pending
	ErrorMessage     string         `gorm:"type:text" json:"error_message"`
	ResponseData     string         `gorm:"type:text" json:"response_data"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// IAPSubscriptionSync IAP 订阅同步记录
type IAPSubscriptionSync struct {
	ID             uint           `gorm:"primaryKey" json:"id"`
	UserID         uint64         `gorm:"index" json:"user_id"`
	Platform       string         `gorm:"type:varchar(50);index" json:"platform"` // apple, google
	SubscriptionID string         `gorm:"type:varchar(255);index" json:"subscription_id"`
	SyncType       string         `gorm:"type:varchar(50);index" json:"sync_type"`   // manual, auto, webhook
	SyncStatus     string         `gorm:"type:varchar(50);index" json:"sync_status"` // success, failed, pending
	ErrorMessage   string         `gorm:"type:text" json:"error_message"`
	PreviousStatus string         `gorm:"type:varchar(50)" json:"previous_status"`
	NewStatus      string         `gorm:"type:varchar(50)" json:"new_status"`
	LastSyncTime   time.Time      `gorm:"index" json:"last_sync_time"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// TableName 返回表名
func (AppleReceipt) TableName() string {
	return "apple_receipts"
}

func (AppleSubscription) TableName() string {
	return "apple_subscriptions"
}

func (AppleNotification) TableName() string {
	return "apple_notifications"
}

func (GooglePurchase) TableName() string {
	return "google_purchases"
}

func (GoogleSubscription) TableName() string {
	return "google_subscriptions"
}

func (GoogleNotification) TableName() string {
	return "google_notifications"
}

func (IAPReceiptValidation) TableName() string {
	return "iap_receipt_validations"
}

func (IAPSubscriptionSync) TableName() string {
	return "iap_subscription_syncs"
}

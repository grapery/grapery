package pay

import (
	"time"

	"gorm.io/gorm"
)

// WebPaymentStatus represents the status of a web payment
type WebPaymentStatus string

const (
	WebPaymentStatusPending    WebPaymentStatus = "pending"
	WebPaymentStatusProcessing WebPaymentStatus = "processing"
	WebPaymentStatusSucceeded  WebPaymentStatus = "succeeded"
	WebPaymentStatusFailed     WebPaymentStatus = "failed"
	WebPaymentStatusCancelled  WebPaymentStatus = "cancelled"
	WebPaymentStatusRefunded   WebPaymentStatus = "refunded"
)

// WebPaymentMethod represents the payment method used
type WebPaymentMethod string

const (
	WebPaymentMethodStripe    WebPaymentMethod = "stripe"
	WebPaymentMethodGooglePay WebPaymentMethod = "google_pay"
	WebPaymentMethodApplePay  WebPaymentMethod = "apple_pay"
	WebPaymentMethodAlipay    WebPaymentMethod = "alipay"
)

// WebPayment represents a web payment record
type WebPayment struct {
	ID                    string                 `json:"id" gorm:"primaryKey"`
	UserID                string                 `json:"userId" gorm:"index:idx_web_payments_user_id"`
	PlanID                string                 `json:"planId" gorm:"index:idx_web_payments_plan_id"`
	Amount                int                    `json:"amount"`
	Currency              string                 `json:"currency" gorm:"default:'USD'"`
	Status                WebPaymentStatus       `json:"status" gorm:"index:idx_web_payments_status"`
	Method                WebPaymentMethod       `json:"method" gorm:"index:idx_web_payments_method"`
	CreatedAt             int64                  `json:"createdAt" gorm:"index:idx_web_payments_created_at"`
	UpdatedAt             int64                  `json:"updatedAt"`
	Metadata              map[string]interface{} `json:"metadata" gorm:"type:jsonb"`
	StripePaymentIntentID string                 `json:"stripePaymentIntentId,omitempty" gorm:"uniqueIndex:idx_web_payments_stripe_pi"`
	StripeClientSecret    string                 `json:"stripeClientSecret,omitempty" gorm:"size:512"`
	AlipayOutTradeNo      string                 `json:"alipayOutTradeNo,omitempty" gorm:"uniqueIndex:idx_web_payments_alipay_trade"`
	AlipayQRCodeURL       string                 `json:"alipayQRCodeURL,omitempty" gorm:"size:512"`
	FailureReason         string                 `json:"failureReason,omitempty" gorm:"size:256"`
	FailureCode           string                 `json:"failureCode,omitempty" gorm:"size:64"`
}

// TableName specifies the table name for WebPayment
func (WebPayment) TableName() string {
	return "web_payments"
}

// WebPaymentRepository provides database operations for web payments
type WebPaymentRepository struct {
	db *gorm.DB
}

// NewWebPaymentRepository creates a new web payment repository
func NewWebPaymentRepository() *WebPaymentRepository {
	return &WebPaymentRepository{
		db: DataBase(),
	}
}

// CreatePayment creates a new web payment record
func (r *WebPaymentRepository) CreatePayment(payment *WebPayment) error {
	return r.db.Create(payment).Error
}

// GetPaymentByID retrieves a payment by ID
func (r *WebPaymentRepository) GetPaymentByID(id string) (*WebPayment, error) {
	var payment WebPayment
	err := r.db.Where("id = ?", id).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetPaymentByStripePaymentIntentID retrieves a payment by Stripe Payment Intent ID
func (r *WebPaymentRepository) GetPaymentByStripePaymentIntentID(stripePI string) (*WebPayment, error) {
	var payment WebPayment
	err := r.db.Where("stripe_payment_intent_id = ?", stripePI).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// GetPaymentByAlipayOutTradeNo retrieves a payment by Alipay out trade no
func (r *WebPaymentRepository) GetPaymentByAlipayOutTradeNo(tradeNo string) (*WebPayment, error) {
	var payment WebPayment
	err := r.db.Where("alipay_out_trade_no = ?", tradeNo).First(&payment).Error
	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// UpdatePayment updates payment fields
func (r *WebPaymentRepository) UpdatePayment(id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now().UnixMilli()
	return r.db.Model(&WebPayment{}).Where("id = ?", id).Updates(updates).Error
}

// GetUserPayments retrieves payment history for a user
func (r *WebPaymentRepository) GetUserPayments(userID string, limit, offset int, status *WebPaymentStatus, method *WebPaymentMethod) ([]*WebPayment, int64, error) {
	var payments []*WebPayment
	var total int64

	query := r.db.Model(&WebPayment{}).Where("user_id = ?", userID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	if method != nil {
		query = query.Where("method = ?", *method)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&payments).Error
	if err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// GetUserPaymentsByDateRange retrieves payments within a date range
func (r *WebPaymentRepository) GetUserPaymentsByDateRange(userID string, startDate, endDate int64, limit, offset int) ([]*WebPayment, int64, error) {
	var payments []*WebPayment
	var total int64

	query := r.db.Model(&WebPayment{}).Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startDate, endDate)

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&payments).Error
	if err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}

// AutoMigrateWebPayments auto-migrates the web_payments table
func AutoMigrateWebPayments() error {
	db := DataBase()

	// Auto migrate
	if err := db.AutoMigrate(&WebPayment{}); err != nil {
		return err
	}

	// Create indexes
	if err := createWebPaymentIndexes(db); err != nil {
		return err
	}

	return nil
}

func createWebPaymentIndexes(db *gorm.DB) error {
	// Composite indexes
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_web_payments_user_status ON web_payments(user_id, status)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_web_payments_user_created ON web_payments(user_id, created_at DESC)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_web_payments_method_status ON web_payments(method, status)").Error; err != nil {
		return err
	}
	return nil
}

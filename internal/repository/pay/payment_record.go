package pay

import (
	"context"
	"encoding/json"
	"time"
)

// PaymentStatus 支付状态
type PaymentStatus int

const (
	PaymentStatusPending         PaymentStatus = iota + 1 // 待支付
	PaymentStatusSuccess                                  // 支付成功
	PaymentStatusFailed                                   // 支付失败
	PaymentStatusCanceled                                 // 已取消
	PaymentStatusRefunded                                 // 已退款
	PaymentStatusPartialRefunded                          // 部分退款
	PaymentStatusProcessing                               // 处理中
	PaymentStatusExpired                                  // 已过期
	PaymentStatusDisputed                                 // 争议中
)

// PaymentMethod 支付方式
type PaymentMethod int

const (
	PaymentMethodApplePay  PaymentMethod = iota + 1 // Apple Pay
	PaymentMethodGooglePay                          // Google Pay
)

// PaymentRecord IAP 支付记录模型
type PaymentRecord struct {
	IDBase
	UserID          int64         `gorm:"column:user_id;not null;index" json:"user_id"`                                                                 // 用户ID
	SubscriptionID  *uint         `gorm:"column:subscription_id;index" json:"subscription_id"`                                                          // 关联的订阅ID（可选）
	Amount          int64         `gorm:"column:amount;not null" json:"amount"`                                                                         // 支付金额（分）
	Currency        string        `gorm:"column:currency;size:10;default:'USD'" json:"currency"`                                                        // 货币类型
	Status          PaymentStatus `gorm:"column:status;default:1" json:"status"`                                                                        // 支付状态
	PaymentMethod   PaymentMethod `gorm:"column:payment_method;not null" json:"payment_method"`                                                         // 支付方式（Apple Pay/Google Pay）
	PaymentProvider string        `gorm:"column:payment_provider;size:100;not null;index;comment:支付提供商（App Store/Google Play）" json:"payment_provider"` // 支付提供商（增加长度冗余）
	TransactionID   string        `gorm:"column:transaction_id;size:512;uniqueIndex;not null;comment:IAP交易ID" json:"transaction_id"`                    // IAP 交易ID（增加长度冗余）
	OriginalTxID    string        `gorm:"column:original_tx_id;size:512;index;comment:原始交易ID（用于订阅续费）" json:"original_tx_id"`                            // 原始交易ID（增加长度冗余）
	ProductID       string        `gorm:"column:product_id;size:255;not null;index;comment:产品ID" json:"product_id"`                                     // 产品ID
	PurchaseToken   string        `gorm:"column:purchase_token;type:longtext;comment:购买令牌（Google IAP）" json:"purchase_token"`                           // 购买令牌（Google IAP，使用longtext增加冗余）
	ReceiptData     string        `gorm:"column:receipt_data;type:longtext;comment:收据数据（Apple IAP）" json:"receipt_data"`                                // 收据数据（Apple IAP，使用longtext增加冗余）
	PaymentTime     *time.Time    `gorm:"column:payment_time;index;comment:支付时间" json:"payment_time"`                                                   // 支付时间
	ExpiresTime     *time.Time    `gorm:"column:expires_time;index;comment:过期时间（订阅用）" json:"expires_time"`                                              // 过期时间（订阅用）
	RefundAmount    int64         `gorm:"column:refund_amount;default:0;not null;comment:退款金额（分）" json:"refund_amount"`                                 // 退款金额（分）
	RefundTime      *time.Time    `gorm:"column:refund_time;index;comment:退款时间" json:"refund_time"`                                                     // 退款时间
	RefundReason    string        `gorm:"column:refund_reason;size:512;comment:退款原因" json:"refund_reason"`                                              // 退款原因（增加长度冗余）
	ErrorCode       string        `gorm:"column:error_code;size:100;index;comment:错误代码" json:"error_code"`                                              // 错误代码
	ErrorMessage    string        `gorm:"column:error_message;type:text;comment:错误信息" json:"error_message"`                                             // 错误信息（使用text增加冗余）
	Environment     string        `gorm:"column:environment;size:50;default:'Production';index;comment:环境（Production/Sandbox）" json:"environment"`      // 环境（增加长度冗余）
	Metadata        string        `gorm:"column:metadata;type:json;comment:元数据（JSON格式）" json:"metadata"`                                                // 元数据（JSON）
	// IAP 特定字段
	IsSubscription   bool   `gorm:"column:is_subscription;default:false" json:"is_subscription"`                   // 是否为订阅
	IsTrial          bool   `gorm:"column:is_trial;default:false" json:"is_trial"`                                 // 是否为试用期
	IsIntroOffer     bool   `gorm:"column:is_intro_offer;default:false" json:"is_intro_offer"`                     // 是否为优惠价格
	AutoRenewStatus  string `gorm:"column:auto_renew_status;size:20" json:"auto_renew_status"`                     // 自动续费状态
	LastNotification string `gorm:"column:last_notification;size:50" json:"last_notification"`                     // 最后通知类型
	VerificationHash string `gorm:"column:verification_hash;size:512;index;comment:验证哈希" json:"verification_hash"` // 验证哈希（增加长度冗余）
	IsTest           bool   `gorm:"column:is_test;default:false;index;comment:是否为测试支付" json:"is_test"`             // 是否为测试支付
	Notes            string `gorm:"column:notes;type:text;comment:备注" json:"notes"`                                // 备注
}

func (p PaymentRecord) TableName() string {
	return "payment_records"
}

// GetMetadata 获取元数据
func (p *PaymentRecord) GetMetadata() (map[string]interface{}, error) {
	if p.Metadata == "" {
		return map[string]interface{}{}, nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal([]byte(p.Metadata), &metadata)
	return metadata, err
}

// SetMetadata 设置元数据
func (p *PaymentRecord) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	p.Metadata = string(data)
	return nil
}

// IsSuccessful 判断支付是否成功
func (p *PaymentRecord) IsSuccessful() bool {
	return p.Status == PaymentStatusSuccess
}

// IsRefundable 判断是否可以退款
func (p *PaymentRecord) IsRefundable() bool {
	return p.Status == PaymentStatusSuccess && p.RefundAmount < p.Amount
}

// GetRefundableAmount 获取可退款金额
func (p *PaymentRecord) GetRefundableAmount() int64 {
	if p.Status != PaymentStatusSuccess {
		return 0
	}
	return p.Amount - p.RefundAmount
}

// CreatePaymentRecord 创建支付记录
func CreatePaymentRecord(ctx context.Context, record *PaymentRecord) error {
	return DataBase().WithContext(ctx).Create(record).Error
}

// GetPaymentRecord 获取支付记录
func GetPaymentRecord(ctx context.Context, id uint) (*PaymentRecord, error) {
	var record PaymentRecord
	err := DataBase().WithContext(ctx).Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetPaymentRecordByTransactionID 根据交易ID获取支付记录
func GetPaymentRecordByTransactionID(ctx context.Context, transactionID string) (*PaymentRecord, error) {
	var record PaymentRecord
	err := DataBase().WithContext(ctx).Where("transaction_id = ?", transactionID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetPaymentRecordByProviderOrderID 根据第三方订单ID获取支付记录
func GetPaymentRecordByProviderOrderID(ctx context.Context, providerOrderID string) (*PaymentRecord, error) {
	var record PaymentRecord
	err := DataBase().WithContext(ctx).Where("provider_order_id = ?", providerOrderID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetPaymentRecordByProviderPaymentID 根据第三方支付ID获取支付记录
func GetPaymentRecordByProviderPaymentID(ctx context.Context, providerPaymentID string) (*PaymentRecord, error) {
	var record PaymentRecord
	err := DataBase().WithContext(ctx).Where("provider_payment_id = ?", providerPaymentID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetUserPaymentRecords 获取用户支付记录
func GetUserPaymentRecords(ctx context.Context, userID int64, offset, limit int) ([]*PaymentRecord, error) {
	var records []*PaymentRecord
	err := DataBase().WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetUserPaymentRecordsByStatus 根据状态获取用户支付记录
func GetUserPaymentRecordsByStatus(ctx context.Context, userID int64, status PaymentStatus, offset, limit int) ([]*PaymentRecord, error) {
	var records []*PaymentRecord
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, status).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetUserPaymentRecordsByMethod 根据支付方式获取用户支付记录
func GetUserPaymentRecordsByMethod(ctx context.Context, userID int64, method PaymentMethod, offset, limit int) ([]*PaymentRecord, error) {
	var records []*PaymentRecord
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND payment_method = ?", userID, method).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// UpdatePaymentStatus 更新支付状态
func UpdatePaymentStatus(ctx context.Context, id uint, status PaymentStatus, paymentTime *time.Time) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if paymentTime != nil {
		updates["payment_time"] = paymentTime
	}

	return DataBase().WithContext(ctx).
		Model(&PaymentRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdatePaymentError 更新支付错误信息
func UpdatePaymentError(ctx context.Context, id uint, errorCode, errorMessage string) error {
	return DataBase().WithContext(ctx).
		Model(&PaymentRecord{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":        PaymentStatusFailed,
			"error_code":    errorCode,
			"error_message": errorMessage,
		}).Error
}

// UpdatePaymentRefund 更新退款信息
func UpdatePaymentRefund(ctx context.Context, id uint, refundAmount int64, refundReason string) error {
	now := time.Now()
	updates := map[string]interface{}{
		"refund_amount": refundAmount,
		"refund_reason": refundReason,
		"refund_time":   &now,
	}

	// 判断是否为全额退款
	record, err := GetPaymentRecord(ctx, id)
	if err != nil {
		return err
	}

	if refundAmount >= record.Amount {
		updates["status"] = PaymentStatusRefunded
	} else {
		updates["status"] = PaymentStatusPartialRefunded
	}

	return DataBase().WithContext(ctx).
		Model(&PaymentRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetPaymentRecordsBySubscriptionID 根据订阅ID获取支付记录
func GetPaymentRecordsBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*PaymentRecord, error) {
	var records []*PaymentRecord
	err := DataBase().WithContext(ctx).
		Where("subscription_id = ?", subscriptionID).
		Order("created_at DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

// GetPaymentStats 获取支付统计信息
func GetPaymentStats(ctx context.Context, userID int64) (map[string]interface{}, error) {
	var stats struct {
		TotalPayments      int64   `json:"total_payments"`
		SuccessfulPayments int64   `json:"successful_payments"`
		FailedPayments     int64   `json:"failed_payments"`
		TotalAmount        int64   `json:"total_amount"`
		TotalRefunded      int64   `json:"total_refunded"`
		SuccessRate        float64 `json:"success_rate"`
	}

	err := DataBase().WithContext(ctx).
		Model(&PaymentRecord{}).
		Where("user_id = ?", userID).
		Select(`
			COUNT(*) as total_payments,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as successful_payments,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as failed_payments,
			SUM(CASE WHEN status = ? THEN amount ELSE 0 END) as total_amount,
			SUM(refund_amount) as total_refunded
		`, PaymentStatusSuccess, PaymentStatusFailed, PaymentStatusSuccess).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	// 计算成功率
	if stats.TotalPayments > 0 {
		stats.SuccessRate = float64(stats.SuccessfulPayments) / float64(stats.TotalPayments) * 100
	}

	return map[string]interface{}{
		"total_payments":      stats.TotalPayments,
		"successful_payments": stats.SuccessfulPayments,
		"failed_payments":     stats.FailedPayments,
		"total_amount":        stats.TotalAmount,
		"total_refunded":      stats.TotalRefunded,
		"success_rate":        stats.SuccessRate,
	}, nil
}

// GetPaymentMethodStats 获取支付方式统计
func GetPaymentMethodStats(ctx context.Context, userID int64) (map[PaymentMethod]map[string]interface{}, error) {
	var stats []struct {
		PaymentMethod PaymentMethod `json:"payment_method"`
		Count         int64         `json:"count"`
		TotalAmount   int64         `json:"total_amount"`
		SuccessCount  int64         `json:"success_count"`
	}

	err := DataBase().WithContext(ctx).
		Model(&PaymentRecord{}).
		Where("user_id = ?", userID).
		Select(`
			payment_method,
			COUNT(*) as count,
			SUM(CASE WHEN status = ? THEN amount ELSE 0 END) as total_amount,
			SUM(CASE WHEN status = ? THEN 1 ELSE 0 END) as success_count
		`, PaymentStatusSuccess, PaymentStatusSuccess).
		Group("payment_method").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	result := make(map[PaymentMethod]map[string]interface{})
	for _, stat := range stats {
		successRate := 0.0
		if stat.Count > 0 {
			successRate = float64(stat.SuccessCount) / float64(stat.Count) * 100
		}

		result[stat.PaymentMethod] = map[string]interface{}{
			"count":         stat.Count,
			"total_amount":  stat.TotalAmount,
			"success_count": stat.SuccessCount,
			"success_rate":  successRate,
		}
	}

	return result, nil
}

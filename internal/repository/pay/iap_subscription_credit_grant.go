package pay

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// IAPSubscriptionCreditGrant 记录「某一 Apple/Google 订阅交易 transaction_id」是否已为会员发放当期点数，
// 用于收据校验幂等：同一 transaction_id 只重置一次配额；续费产生新 transaction_id 会再次发放。
type IAPSubscriptionCreditGrant struct {
	TransactionID string    `gorm:"column:transaction_id;primaryKey;size:255;comment:当期订阅交易ID" json:"transaction_id"`
	UserID        string    `gorm:"column:user_id;index;size:64;not null;comment:应用用户UUID" json:"user_id"`
	ProductID     string    `gorm:"column:product_id;size:255;comment:SKU" json:"product_id"`
	CreatedAt     time.Time `gorm:"column:created_at;comment:发放记录时间" json:"created_at"`
}

func (IAPSubscriptionCreditGrant) TableName() string {
	return "iap_subscription_credit_grants"
}

// IAPSubscriptionCreditRevoke 记录退款/撤销/到期等「回收」动作的幂等键。
// 与发放表分离，避免购买已占用 transaction_id 后退款无法回收额度。
type IAPSubscriptionCreditRevoke struct {
	Action        string    `gorm:"column:action;primaryKey;size:32;comment:revoked|expired" json:"action"`
	TransactionID string    `gorm:"column:transaction_id;primaryKey;size:255;comment:交易或原始交易ID" json:"transaction_id"`
	UserID        string    `gorm:"column:user_id;index;size:64;not null;comment:应用用户UUID" json:"user_id"`
	ProductID     string    `gorm:"column:product_id;size:255;comment:SKU" json:"product_id"`
	CreatedAt     time.Time `gorm:"column:created_at;comment:回收记录时间" json:"created_at"`
}

func (IAPSubscriptionCreditRevoke) TableName() string {
	return "iap_subscription_credit_revokes"
}

// IAPConsumableCreditGrant 记录消耗型 IAP 点数充值，按平台交易 ID 幂等；退款时按行 clawback。
type IAPConsumableCreditGrant struct {
	TransactionID string     `gorm:"column:transaction_id;primaryKey;size:255;comment:消耗型购买交易ID" json:"transaction_id"`
	UserID        string     `gorm:"column:user_id;index;size:64;not null;comment:应用用户UUID" json:"user_id"`
	ProductID     string     `gorm:"column:product_id;size:255;comment:SKU" json:"product_id"`
	Tokens        int        `gorm:"column:tokens;not null;comment:入账 token 数" json:"tokens"`
	CreatedAt     time.Time  `gorm:"column:created_at;comment:入账时间" json:"created_at"`
	RevokedAt     *time.Time `gorm:"column:revoked_at;index;comment:退款回收时间" json:"revoked_at"`
}

func (IAPConsumableCreditGrant) TableName() string {
	return "iap_consumable_credit_grants"
}

// TryClaimSubscriptionCreditGrant 插入发放记录；若 transaction_id 已存在（并发重复校验）则返回 claimed=false。
func TryClaimSubscriptionCreditGrant(ctx context.Context, transactionID, userID, productID string) (claimed bool, err error) {
	if transactionID == "" || userID == "" {
		return false, errors.New("transaction id and user id required")
	}
	db := DataBase().WithContext(ctx)
	row := IAPSubscriptionCreditGrant{
		TransactionID: transactionID,
		UserID:        userID,
		ProductID:     productID,
		CreatedAt:     time.Now(),
	}
	if err := db.Create(&row).Error; err != nil {
		if isDuplicateKey(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ReleaseSubscriptionCreditGrant 发放后 apply 失败时删除 claim，允许重试。
func ReleaseSubscriptionCreditGrant(ctx context.Context, transactionID string) error {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil
	}
	return DataBase().WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		Delete(&IAPSubscriptionCreditGrant{}).Error
}

// TryClaimSubscriptionCreditRevoke 退款/到期回收幂等。
func TryClaimSubscriptionCreditRevoke(ctx context.Context, action, transactionID, userID, productID string) (claimed bool, err error) {
	action = strings.ToLower(strings.TrimSpace(action))
	transactionID = strings.TrimSpace(transactionID)
	userID = strings.TrimSpace(userID)
	if action == "" || transactionID == "" || userID == "" {
		return false, errors.New("action, transaction id and user id required")
	}
	row := IAPSubscriptionCreditRevoke{
		Action:        action,
		TransactionID: transactionID,
		UserID:        userID,
		ProductID:     productID,
		CreatedAt:     time.Now(),
	}
	if err := DataBase().WithContext(ctx).Create(&row).Error; err != nil {
		if isDuplicateKey(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ReleaseSubscriptionCreditRevoke apply 失败时允许重试回收。
func ReleaseSubscriptionCreditRevoke(ctx context.Context, action, transactionID string) error {
	action = strings.ToLower(strings.TrimSpace(action))
	transactionID = strings.TrimSpace(transactionID)
	if action == "" || transactionID == "" {
		return nil
	}
	return DataBase().WithContext(ctx).
		Where("action = ? AND transaction_id = ?", action, transactionID).
		Delete(&IAPSubscriptionCreditRevoke{}).Error
}

// TryClaimConsumableCreditGrant 消耗型充值幂等；已存在返回 claimed=false。
func TryClaimConsumableCreditGrant(ctx context.Context, transactionID, userID, productID string, tokens int) (claimed bool, err error) {
	transactionID = strings.TrimSpace(transactionID)
	userID = strings.TrimSpace(userID)
	if transactionID == "" || userID == "" {
		return false, errors.New("transaction id and user id required")
	}
	if tokens <= 0 {
		return false, errors.New("tokens must be positive")
	}
	row := IAPConsumableCreditGrant{
		TransactionID: transactionID,
		UserID:        userID,
		ProductID:     productID,
		Tokens:        tokens,
		CreatedAt:     time.Now(),
	}
	if err := DataBase().WithContext(ctx).Create(&row).Error; err != nil {
		if isDuplicateKey(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ReleaseConsumableCreditGrant apply 失败时删除 claim。
func ReleaseConsumableCreditGrant(ctx context.Context, transactionID string) error {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil
	}
	return DataBase().WithContext(ctx).
		Where("transaction_id = ? AND revoked_at IS NULL", transactionID).
		Delete(&IAPConsumableCreditGrant{}).Error
}

// MarkConsumableCreditRevoked 标记消耗型充值已退款，返回应 clawback 的 token 数；已撤销或未找到返回 0。
func MarkConsumableCreditRevoked(ctx context.Context, transactionID string) (tokens int, err error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return 0, errors.New("transaction id required")
	}
	now := time.Now()
	err = DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row IAPConsumableCreditGrant
		if qErr := tx.Where("transaction_id = ?", transactionID).First(&row).Error; qErr != nil {
			if errors.Is(qErr, gorm.ErrRecordNotFound) {
				tokens = 0
				return nil
			}
			return qErr
		}
		if row.RevokedAt != nil {
			tokens = 0
			return nil
		}
		tokens = row.Tokens
		return tx.Model(&row).Updates(map[string]interface{}{
			"revoked_at": now,
		}).Error
	})
	return tokens, err
}

// SumActiveConsumableTopUpTokens 用户未撤销的消耗型充值合计。
func SumActiveConsumableTopUpTokens(ctx context.Context, userID string) (int, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return 0, nil
	}
	var sum int
	err := DataBase().WithContext(ctx).Model(&IAPConsumableCreditGrant{}).
		Select("COALESCE(SUM(tokens), 0)").
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Scan(&sum).Error
	return sum, err
}

func isDuplicateKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

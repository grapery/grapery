package pay

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
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
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

package pay

import (
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SeedGraperyIAPAppleProductsIfMissing 将 Grapery App Store 订阅 SKU 写入 iap_products（与 grapery/scripts/membership_iap_seed.sql・iOS IAP 对齐）。
// QuotaLimit 为主库 token 计量（1 展示 credit = common.CreditToTokenRatio 个 token），与 subscription_plans.token_quota（「点」展示值）对应。
// VipPay 的 GET /iap/products 读取本表：若为空则返回 product_count=0。
func SeedGraperyIAPAppleProductsIfMissing(log *zap.Logger) error {
	return seedGraperyIAPAppleProducts(DataBase(), log)
}

func seedGraperyIAPAppleProducts(db *gorm.DB, log *zap.Logger) error {
	if db == nil {
		return errors.New("database nil")
	}
	rows := defaultGraperyAppleIAPRows()
	inserted := 0
	for _, p := range rows {
		var n int64
		if err := db.Model(&IAPProduct{}).Where("product_id = ?", p.ProductID).Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			continue
		}
		if err := db.Create(&p).Error; err != nil {
			log.Warn("Failed to seed IAP product", zap.String("product_id", p.ProductID), zap.Error(err))
			continue
		}
		inserted++
		log.Info("Seeded Grapery IAP product", zap.String("product_id", p.ProductID))
	}
	if inserted == 0 {
		log.Debug("Grapery Apple IAP catalog already present (no new rows)")
	}
	return nil
}

func defaultGraperyAppleIAPRows() []IAPProduct {
	sku := func(s string) *string { return &s }

	return []IAPProduct{
		{
			ProductID:       "com.grapery.pro.monthly",
			Platform:        IAPProductPlatformApple,
			ProductType:     IAPProductTypeSubscription,
			Status:          IAPProductStatusActive,
			Name:            "Grapery Basic — Monthly",
			Description:     "Basic membership monthly (plan_pro_monthly)",
			Price:           29.9,
			Currency:        "CNY",
			Duration:        utils.StringPtr("P1M"),
			AppleSKU:        sku("com.grapery.pro.monthly"),
			AppleProductID:  sku("com.grapery.pro.monthly"),
			IsActive:        true,
			DisplayOrder:    10,
			Featured:        false,
			QuotaLimit:      100 * common.CreditToTokenRatio,
			MaxRoles:        50,
			MaxContexts:     50,
			SyncStatus:      "local_seed",
			FamilyShareable: false,
		},
		{
			ProductID:       "com.grapery.pro.yearly",
			Platform:        IAPProductPlatformApple,
			ProductType:     IAPProductTypeSubscription,
			Status:          IAPProductStatusActive,
			Name:            "Grapery Basic — Yearly",
			Description:     "Basic membership yearly (plan_pro_yearly)",
			Price:           287,
			Currency:        "CNY",
			Duration:        utils.StringPtr("P1Y"),
			AppleSKU:        sku("com.grapery.pro.yearly"),
			AppleProductID:  sku("com.grapery.pro.yearly"),
			IsActive:        true,
			DisplayOrder:    12,
			Featured:        false,
			QuotaLimit:      100 * common.CreditToTokenRatio,
			MaxRoles:        50,
			MaxContexts:     50,
			SyncStatus:      "local_seed",
			FamilyShareable: false,
		},
		{
			ProductID:       "com.grapery.prime.monthly",
			Platform:        IAPProductPlatformApple,
			ProductType:     IAPProductTypeSubscription,
			Status:          IAPProductStatusActive,
			Name:            "Grapery Premium — Monthly",
			Description:     "Premium membership monthly (plan_prime_monthly)",
			Price:           49.9,
			Currency:        "CNY",
			Duration:        utils.StringPtr("P1M"),
			AppleSKU:        sku("com.grapery.prime.monthly"),
			AppleProductID:  sku("com.grapery.prime.monthly"),
			IsActive:        true,
			DisplayOrder:    20,
			Featured:        false,
			QuotaLimit:      200 * common.CreditToTokenRatio,
			MaxRoles:        200,
			MaxContexts:     200,
			SyncStatus:      "local_seed",
			FamilyShareable: false,
		},
		{
			ProductID:       "com.grapery.prime.yearly",
			Platform:        IAPProductPlatformApple,
			ProductType:     IAPProductTypeSubscription,
			Status:          IAPProductStatusActive,
			Name:            "Grapery Premium — Yearly",
			Description:     "Premium membership yearly (plan_prime_yearly)",
			Price:           479,
			Currency:        "CNY",
			Duration:        utils.StringPtr("P1Y"),
			AppleSKU:        sku("com.grapery.prime.yearly"),
			AppleProductID:  sku("com.grapery.prime.yearly"),
			IsActive:        true,
			DisplayOrder:    22,
			Featured:        false,
			QuotaLimit:      200 * common.CreditToTokenRatio,
			MaxRoles:        200,
			MaxContexts:     200,
			SyncStatus:      "local_seed",
			FamilyShareable: false,
		},
	}
}

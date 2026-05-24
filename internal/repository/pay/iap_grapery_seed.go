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
	inserted, updated := 0, 0
	for _, p := range rows {
		var existing IAPProduct
		err := db.Where("product_id = ?", p.ProductID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.Create(&p).Error; err != nil {
				log.Warn("Failed to seed IAP product", zap.String("product_id", p.ProductID), zap.Error(err))
				continue
			}
			inserted++
			log.Info("Seeded Grapery IAP product", zap.String("product_id", p.ProductID))
			continue
		}
		if err != nil {
			return err
		}
		// 已有行仅插入时不会修正 quota_limit；启动时对齐 canonical 配置（含误存为 100/200 的历史值）。
		updates := map[string]interface{}{
			"product_type":  p.ProductType,
			"quota_limit":   p.QuotaLimit,
			"max_roles":     p.MaxRoles,
			"max_contexts":  p.MaxContexts,
			"duration":      p.Duration,
			"currency":      p.Currency,
			"is_active":     p.IsActive,
			"family_shareable": p.FamilyShareable,
		}
		if existing.QuotaLimit != p.QuotaLimit ||
			existing.ProductType != p.ProductType ||
			existing.MaxRoles != p.MaxRoles ||
			existing.MaxContexts != p.MaxContexts {
			if err := db.Model(&IAPProduct{}).Where("product_id = ?", p.ProductID).Updates(updates).Error; err != nil {
				log.Warn("Failed to sync IAP product", zap.String("product_id", p.ProductID), zap.Error(err))
				continue
			}
			updated++
			log.Info("Synced Grapery IAP product",
				zap.String("product_id", p.ProductID),
				zap.Int("old_quota_limit", existing.QuotaLimit),
				zap.Int("new_quota_limit", p.QuotaLimit),
			)
		}
	}
	if inserted == 0 && updated == 0 {
		log.Debug("Grapery Apple IAP catalog already aligned")
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

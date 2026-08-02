package pay

import (
	"context"
	"errors"

	"github.com/grapestree/fgrapery/grapery/internal/repository/migrations"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// init 自动注册 pay 包的迁移步骤
func init() {
	registry := migrations.GetRegistry()

	// ========== 支付记录表 ==========
	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_payment_records",
		Description: "Create and migrate payment_records table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&PaymentRecord{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_subscriptions",
		Description: "Create and migrate subscriptions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Subscription{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_user_subscriptions",
		Description: "Create and migrate user_subscriptions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserSubscription{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_web_payments",
		Description: "Create and migrate web_payments table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&WebPayment{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_web_payments_wechat_columns",
		Description: "Add WeChat Pay columns to web_payments",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&WebPayment{})
		},
		Required: true,
	})

	// ========== IAP (应用内购买) 表 ==========
	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_iap_products",
		Description: "Create and migrate iap_products table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&IAPProduct{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_apple_receipts",
		Description: "Create and migrate apple_receipts table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AppleReceipt{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_apple_subscriptions",
		Description: "Create and migrate apple_subscriptions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AppleSubscription{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_apple_notifications",
		Description: "Create and migrate apple_notifications table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&AppleNotification{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_iap_subscription_notices",
		Description: "Pending subscription change notices for client ACK",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&IAPSubscriptionNotice{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_iap_subscription_credit_grants",
		Description: "Idempotent grants for subscription billing periods (Apple/Google transaction_id)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&IAPSubscriptionCreditGrant{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_iap_subscription_credit_revokes",
		Description: "Idempotent revoke/expire claims separated from subscription credit grants",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&IAPSubscriptionCreditRevoke{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_iap_consumable_credit_grants",
		Description: "Idempotent consumable IAP top-ups with refund clawback",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&IAPConsumableCreditGrant{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_google_purchases",
		Description: "Create and migrate google_purchases table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GooglePurchase{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_google_subscriptions",
		Description: "Create and migrate google_subscriptions table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GoogleSubscription{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_google_notifications",
		Description: "Create and migrate google_notifications table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&GoogleNotification{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_iap_receipt_validations",
		Description: "Create and migrate iap_receipt_validations table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&IAPReceiptValidation{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_iap_subscription_syncs",
		Description: "Create and migrate iap_subscription_syncs table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&IAPSubscriptionSync{})
		},
		Required: true,
	})

	// ========== 徽章系统表 ==========
	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_badges",
		Description: "Create and migrate badges table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&Badge{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_user_badges",
		Description: "Create and migrate user_badges table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserBadge{})
		},
		Required: true,
	})

	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_user_badge_stats",
		Description: "Create and migrate user_badge_stats table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&UserBadgeStats{})
		},
		Required: true,
	})

	// ========== Token 使用表 ==========
	registry.RegisterPaymentStep(migrations.MigrationStep{
		Name:        "migrate_token_usage_logs",
		Description: "Create and migrate token_usage_logs table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return db.AutoMigrate(&TokenUsageLog{})
		},
		Required: true,
	})

	// ========== OAuth 相关表（这些表实际上映射到 mysql 包的表）==========
	// 注意：OAuthUser, OAuthUserSettings, OAuthMembership, OAuthThirdPartyLogin
	// 这些表实际上映射到 users, user_settings, memberships, third_party_logins
	// 所以不需要单独迁移，它们会在 mysql 包的迁移中处理

	// ========== 注册索引创建步骤 ==========
	registerPaymentIndexSteps(registry)

	// ========== 注册数据初始化步骤 ==========
	registerPaymentDataInitSteps(registry)
}

// registerPaymentIndexSteps 注册支付相关的索引创建步骤
func registerPaymentIndexSteps(registry *migrations.MigrationRegistry) {
	// web_payments 表的复合索引
	registry.RegisterIndexStep(migrations.MigrationStep{
		Name:        "create_web_payments_indexes",
		Description: "Create composite indexes for web_payments table",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			manager := migrations.NewMigrationManager(db, log)
			indexes := []struct {
				name    string
				columns string
			}{
				{name: "idx_web_payments_user_status", columns: "user_id, status"},
				{name: "idx_web_payments_user_created", columns: "user_id, created_at DESC"},
				{name: "idx_web_payments_method_status", columns: "method, status"},
			}

			for _, idx := range indexes {
				if err := manager.CreateIndex("web_payments", idx.name, idx.columns); err != nil {
					// 索引可能已存在，记录警告但不中断
					log.Warn("Failed to create index (may already exist)", zap.String("index", idx.name), zap.Error(err))
				}
			}
			return nil
		},
		Required: false,
	})
}

// registerPaymentDataInitSteps 注册支付相关的数据初始化步骤
func registerPaymentDataInitSteps(registry *migrations.MigrationRegistry) {
	// 初始化预定义徽章
	registry.RegisterDataInitStep(migrations.MigrationStep{
		Name:        "initialize_predefined_badges",
		Description: "Initialize predefined badges",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return initPredefinedBadges(db, log)
		},
		Required: false,
	})

	registry.RegisterDataInitStep(migrations.MigrationStep{
		Name:        "seed_grapery_iap_products",
		Description: "Insert Grapery Apple IAP rows into iap_products when absent (VipPay product list)",
		Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
			return seedGraperyIAPAppleProducts(db, log)
		},
		Required: false,
	})
}

// initPredefinedBadges 初始化预定义徽章（从 badge_repository.go 移过来）
func initPredefinedBadges(db *gorm.DB, log *zap.Logger) error {
	for _, badge := range PredefinedBadges {
		var existing Badge
		err := db.Where("code = ?", badge.Code).First(&existing).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				badge.IsActive = true
				if err := db.Create(&badge).Error; err != nil {
					log.Warn("Failed to create predefined badge", zap.String("code", badge.Code), zap.Error(err))
					// 继续处理下一个徽章
				} else {
					log.Debug("Created predefined badge", zap.String("code", badge.Code))
				}
			} else {
				log.Warn("Error checking badge existence", zap.String("code", badge.Code), zap.Error(err))
			}
		} else {
			log.Debug("Predefined badge already exists", zap.String("code", badge.Code))
		}
	}
	return nil
}

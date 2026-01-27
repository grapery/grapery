package migrations

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RunAllMigrations 统一执行所有数据库表的迁移
// 这是迁移的统一入口点，应该在应用启动时调用一次
// 它会执行所有注册的迁移步骤：
// 1. 核心业务表迁移（mysql 包）
// 2. 支付相关表迁移（pay 包）
// 3. Schema 修复（确保向后兼容）
// 4. 索引创建
// 5. 数据初始化
func RunAllMigrations(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	registry := GetRegistry()
	return registry.ExecuteAll(ctx, db, log)
}

// RunCoreMigrations 仅执行核心业务表的迁移
func RunCoreMigrations(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	registry := GetRegistry()
	return registry.ExecuteSteps(ctx, db, log, registry.GetCoreSteps(), "Core Tables")
}

// RunPaymentMigrations 仅执行支付相关表的迁移
func RunPaymentMigrations(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	registry := GetRegistry()
	return registry.ExecuteSteps(ctx, db, log, registry.GetPaymentSteps(), "Payment Tables")
}

// RunSchemaFixes 仅执行 Schema 修复
func RunSchemaFixes(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	registry := GetRegistry()
	return registry.ExecuteSteps(ctx, db, log, registry.GetSchemaFixSteps(), "Schema Fixes")
}

// RunIndexCreation 仅执行索引创建
func RunIndexCreation(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	registry := GetRegistry()
	return registry.ExecuteSteps(ctx, db, log, registry.GetIndexSteps(), "Indexes")
}

// RunDataInitialization 仅执行数据初始化
func RunDataInitialization(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	registry := GetRegistry()
	return registry.ExecuteSteps(ctx, db, log, registry.GetDataInitSteps(), "Data Initialization")
}

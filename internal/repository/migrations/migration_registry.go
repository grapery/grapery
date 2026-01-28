package migrations

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrationFunc 定义迁移函数的类型
type MigrationFunc func(ctx context.Context, db *gorm.DB, log *zap.Logger) error

// MigrationStep 定义一个迁移步骤
type MigrationStep struct {
	Name        string
	Description string
	Func        MigrationFunc
	Required    bool // 是否必需的迁移步骤
}

// MigrationRegistry 注册和管理所有迁移
type MigrationRegistry struct {
	coreSteps      []MigrationStep
	paymentSteps   []MigrationStep
	schemaFixSteps []MigrationStep
	indexSteps     []MigrationStep
	dataInitSteps  []MigrationStep
}

// NewMigrationRegistry 创建新的迁移注册表
func NewMigrationRegistry() *MigrationRegistry {
	return &MigrationRegistry{
		coreSteps:      make([]MigrationStep, 0),
		paymentSteps:   make([]MigrationStep, 0),
		schemaFixSteps: make([]MigrationStep, 0),
		indexSteps:     make([]MigrationStep, 0),
		dataInitSteps:  make([]MigrationStep, 0),
	}
}

// RegisterCoreStep 注册核心表迁移步骤
func (r *MigrationRegistry) RegisterCoreStep(step MigrationStep) {
	r.coreSteps = append(r.coreSteps, step)
}

// RegisterPaymentStep 注册支付表迁移步骤
func (r *MigrationRegistry) RegisterPaymentStep(step MigrationStep) {
	r.paymentSteps = append(r.paymentSteps, step)
}

// RegisterSchemaFixStep 注册 Schema 修复步骤
func (r *MigrationRegistry) RegisterSchemaFixStep(step MigrationStep) {
	r.schemaFixSteps = append(r.schemaFixSteps, step)
}

// RegisterIndexStep 注册索引创建步骤
func (r *MigrationRegistry) RegisterIndexStep(step MigrationStep) {
	r.indexSteps = append(r.indexSteps, step)
}

// RegisterDataInitStep 注册数据初始化步骤
func (r *MigrationRegistry) RegisterDataInitStep(step MigrationStep) {
	r.dataInitSteps = append(r.dataInitSteps, step)
}

// ExecuteAll 执行所有迁移步骤
func (r *MigrationRegistry) ExecuteAll(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
	log.Info("==================================================")
	log.Info("Starting Unified Database Migration")
	log.Info("==================================================")

	// 步骤 1: 迁移核心表
	if err := r.ExecuteSteps(ctx, db, log, r.coreSteps, "Core Tables"); err != nil {
		return fmt.Errorf("core tables migration failed: %w", err)
	}

	// 步骤 2: 迁移支付表
	if err := r.ExecuteSteps(ctx, db, log, r.paymentSteps, "Payment Tables"); err != nil {
		return fmt.Errorf("payment tables migration failed: %w", err)
	}

	// 步骤 3: 应用 Schema 修复
	if err := r.ExecuteSteps(ctx, db, log, r.schemaFixSteps, "Schema Fixes"); err != nil {
		return fmt.Errorf("schema fixes failed: %w", err)
	}

	// 步骤 4: 创建索引
	if err := r.ExecuteSteps(ctx, db, log, r.indexSteps, "Indexes"); err != nil {
		return fmt.Errorf("index creation failed: %w", err)
	}

	// 步骤 5: 初始化数据
	if err := r.ExecuteSteps(ctx, db, log, r.dataInitSteps, "Data Initialization"); err != nil {
		return fmt.Errorf("data initialization failed: %w", err)
	}

	log.Info("==================================================")
	log.Info("All Migrations Completed Successfully")
	log.Info("==================================================")

	return nil
}

// ExecuteSteps 执行一组迁移步骤
func (r *MigrationRegistry) ExecuteSteps(ctx context.Context, db *gorm.DB, log *zap.Logger, steps []MigrationStep, groupName string) error {
	log.Info("--------------------------------------------------")
	log.Info(fmt.Sprintf("Executing: %s", groupName))
	log.Info("--------------------------------------------------")

	totalSteps := len(steps)
	completed := 0
	failed := 0

	for i, step := range steps {
		log.Info(fmt.Sprintf("[%d/%d] Running: %s", i+1, totalSteps, step.Name),
			zap.String("description", step.Description))

		if err := step.Func(ctx, db, log); err != nil {
			if step.Required {
				log.Error(fmt.Sprintf("Required migration step failed: %s", step.Name), zap.Error(err))
				failed++
				return fmt.Errorf("required migration step '%s' failed: %w", step.Name, err)
			} else {
				log.Warn(fmt.Sprintf("Optional migration step failed: %s", step.Name), zap.Error(err))
				failed++
				// 继续执行下一步
			}
		} else {
			log.Debug(fmt.Sprintf("Completed: %s", step.Name))
			completed++
		}
	}

	log.Info(fmt.Sprintf("Group '%s' completed", groupName),
		zap.Int("total", totalSteps),
		zap.Int("completed", completed),
		zap.Int("failed", failed))

	return nil
}

// GetCoreSteps 获取核心表迁移步骤
func (r *MigrationRegistry) GetCoreSteps() []MigrationStep {
	return r.coreSteps
}

// GetPaymentSteps 获取支付表迁移步骤
func (r *MigrationRegistry) GetPaymentSteps() []MigrationStep {
	return r.paymentSteps
}

// GetSchemaFixSteps 获取 Schema 修复步骤
func (r *MigrationRegistry) GetSchemaFixSteps() []MigrationStep {
	return r.schemaFixSteps
}

// GetIndexSteps 获取索引创建步骤
func (r *MigrationRegistry) GetIndexSteps() []MigrationStep {
	return r.indexSteps
}

// GetDataInitSteps 获取数据初始化步骤
func (r *MigrationRegistry) GetDataInitSteps() []MigrationStep {
	return r.dataInitSteps
}

// Global migration registry instance
var globalRegistry *MigrationRegistry

// GetRegistry 获取全局迁移注册表
func GetRegistry() *MigrationRegistry {
	if globalRegistry == nil {
		globalRegistry = NewMigrationRegistry()
	}
	return globalRegistry
}

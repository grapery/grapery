package migrations

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrationManager 提供迁移相关的辅助方法
// 注意：实际的迁移执行通过 MigrationRegistry 和注册的步骤完成
type MigrationManager struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewMigrationManager 创建迁移管理器（用于辅助方法）
func NewMigrationManager(db *gorm.DB, log *zap.Logger) *MigrationManager {
	return &MigrationManager{
		db:  db,
		log: log,
	}
}

// EnsureTableExists 确保表存在（用于特定表的检查）
func (m *MigrationManager) EnsureTableExists(tableName string) (bool, error) {
	var count int64
	err := m.db.Raw(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?",
		tableName,
	).Scan(&count).Error
	return count > 0, err
}

// EnsureColumnExists 确保列存在
func (m *MigrationManager) EnsureColumnExists(tableName, columnName string) (bool, error) {
	var count int64
	err := m.db.Raw(
		"SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?",
		tableName, columnName,
	).Scan(&count).Error
	return count > 0, err
}

// EnsureIndexExists 确保索引存在
func (m *MigrationManager) EnsureIndexExists(tableName, indexName string) (bool, error) {
	var count int64
	err := m.db.Raw(
		`SELECT COUNT(*) FROM information_schema.statistics
		 WHERE table_schema = DATABASE()
		   AND table_name = ?
		   AND index_name = ?`,
		tableName, indexName,
	).Scan(&count).Error
	return count > 0, err
}

// AddColumn 添加列（如果不存在）
func (m *MigrationManager) AddColumn(tableName, columnName, definition string) error {
	exists, err := m.EnsureColumnExists(tableName, columnName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	sql := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", tableName, columnName, definition)
	return m.db.Exec(sql).Error
}

// CreateIndex 创建索引（如果不存在）
func (m *MigrationManager) CreateIndex(tableName, indexName, columns string) error {
	exists, err := m.EnsureIndexExists(tableName, indexName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	sql := fmt.Sprintf("CREATE INDEX `%s` ON `%s`(%s)", indexName, tableName, columns)
	return m.db.Exec(sql).Error
}

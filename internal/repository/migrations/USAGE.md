# 统一数据库迁移系统使用指南

## 概述

本项目使用统一的数据库迁移管理系统，将所有表的 AutoMigrate 调用集中管理，避免分散在各个文件中。

## 架构

```
internal/repository/
├── migrations/
│   ├── migration_registry.go    # 迁移注册表（核心）
│   ├── entry.go                 # 统一入口函数
│   └── USAGE.md                 # 本文档
├── mysql/
│   ├── migrations_register.go   # MySQL 包迁移步骤注册
│   ├── models.go                # MySQL 模型定义
│   └── repository.go            # MySQL Repository（已移除迁移逻辑）
└── pay/
    ├── migrations_register.go   # Pay 包迁移步骤注册
    ├── database.go              # Pay 数据库初始化（已移除迁移逻辑）
    └── ...
```

## 使用方法

### 1. 在应用启动时执行迁移

在应用的主入口文件中（如 `cmd/vippay/main.go`），添加以下代码：

```go
import (
    "context"
    "github.com/grapestree/fgrapery/grapery/internal/repository/migrations"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

func main() {
    // ... 其他初始化代码 ...
    
    // 初始化数据库连接
    db := // ... 获取数据库连接 ...
    log := // ... 获取 logger ...
    
    // 执行统一迁移
    ctx := context.Background()
    if err := migrations.RunAllMigrations(ctx, db, log); err != nil {
        log.Fatal("Failed to run database migrations", zap.Error(err))
    }
    
    // ... 其他启动代码 ...
}
```

### 2. 迁移步骤注册

迁移步骤通过 `init()` 函数自动注册：

- **MySQL 包**：在 `mysql/migrations_register.go` 中注册核心业务表的迁移步骤
- **Pay 包**：在 `pay/migrations_register.go` 中注册支付相关表的迁移步骤

### 3. 迁移执行顺序

迁移按以下顺序执行：

1. **核心业务表迁移**（Core Tables）
   - 所有 mysql 包中定义的表
   - 包括：users, stories, characters, groups 等

2. **支付相关表迁移**（Payment Tables）
   - 所有 pay 包中定义的表
   - 包括：payment_records, subscriptions, badges 等

3. **Schema 修复**（Schema Fixes）
   - 确保向后兼容性的 Schema 修复
   - 包括：添加新列、修改列类型、删除旧索引等

4. **索引创建**（Indexes）
   - 创建额外的复合索引
   - 包括：web_payments 表的复合索引等

5. **数据初始化**（Data Initialization）
   - 初始化必要的数据
   - 包括：预定义徽章等

## 迁移步骤类型

### 必需的迁移步骤（Required: true）

如果迁移失败，会中断整个迁移过程。

### 可选的迁移步骤（Required: false）

如果迁移失败，会记录警告但继续执行后续步骤。

## 添加新的迁移步骤

### 1. 添加新表迁移

在对应的 `migrations_register.go` 文件中添加：

```go
registry.RegisterCoreStep(migrations.MigrationStep{
    Name:        "migrate_new_table",
    Description: "Create and migrate new_table",
    Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
        return db.AutoMigrate(&NewTable{})
    },
    Required: true,
})
```

### 2. 添加 Schema 修复

在 `mysql/migrations_register.go` 的 `registerSchemaFixSteps` 函数中添加：

```go
registry.RegisterSchemaFixStep(migrations.MigrationStep{
    Name:        "ensure_new_column",
    Description: "Ensure table has new_column",
    Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
        repo := &Repository{db: db, log: log}
        return repo.ensureNewColumn()
    },
    Required: false,
})
```

### 3. 添加索引创建

在对应的 `registerIndexSteps` 函数中添加：

```go
registry.RegisterIndexStep(migrations.MigrationStep{
    Name:        "create_new_index",
    Description: "Create index on new_table",
    Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
        return db.Exec("CREATE INDEX IF NOT EXISTS idx_new_table_column ON new_table(column)").Error
    },
    Required: false,
})
```

### 4. 添加数据初始化

在对应的 `registerDataInitSteps` 函数中添加：

```go
registry.RegisterDataInitStep(migrations.MigrationStep{
    Name:        "initialize_new_data",
    Description: "Initialize new data",
    Func: func(ctx context.Context, db *gorm.DB, log *zap.Logger) error {
        // 初始化逻辑
        return nil
    },
    Required: false,
})
```

## 注意事项

1. **避免循环依赖**：迁移注册文件通过 `init()` 函数自动注册，避免直接导入模型包

2. **幂等性**：所有迁移步骤应该是幂等的，可以安全地多次执行

3. **向后兼容**：Schema 修复步骤应该确保向后兼容，不会破坏现有数据

4. **错误处理**：必需的迁移步骤失败会中断整个迁移过程，可选的迁移步骤失败只会记录警告

5. **日志记录**：所有迁移步骤都会记录详细的日志，便于追踪和调试

## 已移除的迁移调用

以下文件中的 AutoMigrate 调用已被移除，改为使用统一迁移系统：

- `mysql/repository.go` - `migrate()` 方法已废弃
- `mysql/models.go` - `migrate()` 函数已废弃
- `pay/database.go` - 所有 AutoMigrate 调用已移除
- `pay/badge_repository.go` - AutoMigrate 调用已移除
- `pay/oauth_repository.go` - AutoMigrate 调用已移除
- `pay/web_payment_models.go` - `AutoMigrateWebPayments()` 已废弃

## 迁移历史

迁移系统会记录每个步骤的执行情况，包括：
- 步骤名称和描述
- 执行进度（当前步骤/总步骤数）
- 执行结果（成功/失败）
- 错误信息（如果有）

## 故障排查

如果迁移失败，检查：

1. 数据库连接是否正常
2. 数据库用户是否有足够的权限
3. 是否有表或索引冲突
4. 查看日志中的详细错误信息

## 示例

完整的使用示例请参考 `cmd/vippay/main.go`（需要更新以使用新的迁移系统）。

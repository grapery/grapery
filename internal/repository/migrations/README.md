# Database Migrations

## 概述

本项目使用统一的数据库迁移管理系统，将所有表的 AutoMigrate 调用集中管理，避免分散在各个文件中。

## 架构说明

### 迁移管理结构

```
internal/repository/
├── migrations/
│   ├── migration_registry.go    # 迁移注册表（核心）
│   ├── entry.go                 # 统一入口函数
│   ├── migration_manager.go     # 辅助方法（表/列/索引检查）
│   ├── README.md                # 本文档
│   ├── USAGE.md                 # 使用指南
│   └── CHANGELOG.md             # 变更日志
├── mysql/
│   ├── migrations_register.go   # MySQL 包迁移步骤注册
│   ├── models.go                # MySQL 模型定义（核心业务表）
│   ├── repository.go             # MySQL Repository 实现
│   └── migrations.go            # 特定 Schema 修复方法
└── pay/
    ├── migrations_register.go   # Pay 包迁移步骤注册
    ├── database.go             # Pay 包的数据库初始化
    ├── badge_repository.go     # 徽章系统
    └── ...
```

## 使用方式

### 在应用启动时执行迁移

在应用的主入口文件中（如 `cmd/vippay/main.go`），添加以下代码：

```go
import (
    "context"
    "github.com/grapestree/fgrapery/grapery/internal/repository/migrations"
    "go.uber.org/zap"
    "gorm.io/gorm"
)

func main() {
    // ... 配置加载和数据库连接 ...
    
    // 执行统一迁移
    ctx := context.Background()
    if err := migrations.RunAllMigrations(ctx, db, log); err != nil {
        log.Fatal("Failed to run database migrations", zap.Error(err))
    }
    
    // ... 其他初始化 ...
}
```

## 迁移策略

### 1. 统一迁移系统

- **核心业务表**：在 `mysql/migrations_register.go` 中注册迁移步骤
- **支付相关表**：在 `pay/migrations_register.go` 中注册迁移步骤
- **自动注册**：通过 `init()` 函数自动注册所有迁移步骤
- **统一执行**：通过 `migrations.RunAllMigrations()` 统一执行所有迁移

### 2. Schema 修复方法

以下方法确保向后兼容性：

- `ensureStoryboardVideoGenerationSchema()` - 确保 storyboard_video_generations 有新字段
- `ensureStoryboardScenesSchema()` - 确保 storyboard_scenes 有新字段
- `ensureStoriesStyleSchema()` - 确保 stories.style 可以存储 JSON (TEXT)
- `ensureAIGenerationRecordsSchema()` - 确保 ai_generation_records prompt 字段支持 Unicode
- `ensureGroupMembersSchema()` - 确保 group_members 有 role_id 列
- `ensureGroupFollowsSchema()` - 确保 group_follows 表存在
- `ensureUserDevicesSchema()` - 确保 user_devices 表存在
- `ensureStoryboardImageGenerationSchema()` - 确保 storyboard_image_generations 有 prompt_details_json
- `ensureStoryboardVideoGenerationPromptDetailsSchema()` - 确保 storyboard_video_generations 有 prompt_details_json
- `ensureCharacterPortraitSchema()` - 确保 characters 有 portrait 相关列
- `ensureIsCollaborationOpenColumn()` - 确保 stories 有 is_collaboration_open 列
- `ensureUserGroupCountColumns()` - 确保 users 有 groups_count 和 groups_created 列

### 3. 索引管理

额外索引在 `createIndexes()` 方法中创建，包括：

- `idx_web_payments_user_status` - web_payments 表的复合索引
- `idx_web_payments_user_created` - web_payments 表的复合索引
- `idx_web_payments_method_status` - web_payments 表的复合索引

## 表分类

### 核心业务表 (mysql 包)

1. **用户相关**
   - users
   - user_login_records
   - user_settings
   - user_devices
   - user_statistics

2. **故事相关**
   - stories
   - panels
   - storyboards
   - story_contributors

3. **角色相关**
   - characters
   - character_posters
   - character_analytics

4. **场景相关**
   - story_scenes
   - storyboard_scenes
   - storyboard_character_links
   - storyboard_scene_links

5. **AI 生成**
   - storyboard_content_generations
   - storyboard_scene_generations
   - storyboard_image_generations
   - storyboard_video_generations
   - ai_generation_records

6. **评论和聊天**
   - comments
   - chat_threads
   - chat_messages

7. **标签和搜索**
   - tags
   - story_tags
   - character_tags
   - search_histories
   - view_histories

8. **Agent 系统**
   - agents
   - agent_skills
   - agent_skill_usages
   - agent_interactions
   - agent_memories

9. **任务系统**
    - ai_tasks
    - render_tasks
    - story_publications

10. **通知和活动**
    - notifications
    - user_activities

11. **第三方登录**
    - third_party_logins

12. **其他**
    - assets
    - reports
    - story_compositions
    - story_participants
    - style_configs
    - invitation_codes
    - storyboard_chat_sessions
    - storyboard_chat_messages

### 支付相关表 (pay 包)

1. **支付记录**
   - payment_records
   - subscriptions
   - user_subscriptions
   - web_payments

2. **IAP (应用内购买)**
   - iap_products
   - apple_receipts
   - apple_subscriptions
   - apple_notifications
   - google_purchases
   - google_subscriptions
   - google_notifications
   - iap_receipt_validations
   - iap_subscription_syncs

3. **徽章系统**
   - badges
   - user_badges
   - user_badge_stats

4. **Token 使用**
   - token_usage_logs

## 最佳实践

### 1. 添加新表

1. 在 `mysql/models.go` 或 `pay/models.go` 中定义模型
2. 在对应的 `migrations_register.go` 中注册迁移步骤：
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

### 2. 修改现有表

1. 在 `mysql/migrations.go` 中添加 `ensure*` 方法检查列是否存在
2. 在 `mysql/migrations_register.go` 的 `registerSchemaFixSteps()` 中注册：
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

### 3. 创建索引

1. 在模型定义中使用 `gorm:"index"` 创建简单索引
2. 对于复合索引，在对应的 `registerIndexSteps()` 函数中注册：
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

### 4. 数据初始化

1. 在对应的 `registerDataInitSteps()` 函数中注册：
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

## 故障排查

### 1. AutoMigrate 失败

```
Error: BLOB/TEXT column 'style' used in key specification without a key length
```

**解决方案**：先删除索引，再修改列类型（已在 `ensureStoriesStyleSchema` 中处理）

### 2. 列已存在错误

```
Error: Duplicate column name 'is_collaboration_open'
```

**解决方案**：使用 `EnsureColumnExists` 检查后再添加

### 3. 索引已存在错误

```
Error: Duplicate key name 'idx_web_payments_user_status'
```

**解决方案**：使用 `IF NOT EXISTS` 语法，或者先检查索引是否存在

## 注意事项

1. **避免循环依赖**：迁移注册通过 `init()` 函数自动执行，避免直接导入模型包
2. **向后兼容**：所有 Schema 修改都要确保向后兼容
3. **原子性**：每个迁移步骤应该是独立的，可以单独执行
4. **幂等性**：迁移步骤应该可以多次执行而不会出错
5. **日志记录**：所有迁移步骤都会记录详细的日志，便于追踪和调试
6. **必需 vs 可选**：使用 `Required` 标志区分必需和可选的迁移步骤

## 迁移历史

- **2024-01**: 初始迁移系统建立
- **2024-02**: 添加 storyboard 视频细分支持
- **2024-03**: 添加 stories.style TEXT 支持
- **2024-04**: 添加 Unicode (utf8mb4) 支持
- **2024-05**: 添加群组角色系统
- **2024-06**: 添加群组关注功能
- **2024-07**: 添加推送通知支持
- **2024-08**: 添加角色形象生成系统
- **2024-09**: 添加协作开放功能
- **2024-10**: 统一迁移管理系统

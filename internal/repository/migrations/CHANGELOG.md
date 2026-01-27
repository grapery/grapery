# 数据库迁移系统重构变更日志

## 概述

本次重构统一了所有数据库表的迁移管理，将所有分散在各处的 AutoMigrate 调用集中到 `migrations` 包中统一管理。

## 变更内容

### 1. 新增文件

#### migrations 包
- `migration_registry.go` - 迁移注册表核心实现
- `entry.go` - 统一迁移入口函数
- `USAGE.md` - 使用指南
- `CHANGELOG.md` - 本文档

#### mysql 包
- `migrations_register.go` - MySQL 包迁移步骤注册（846行）

#### pay 包
- `migrations_register.go` - Pay 包迁移步骤注册

### 2. 修改文件

#### mysql/repository.go
- **移除**：`migrate()` 方法中的实际迁移逻辑
- **标记为废弃**：`migrate()` 方法现在只返回 nil，实际迁移由 migrations 包处理
- **移除**：`NewRepository()` 中对 `migrate()` 的调用

#### mysql/models.go
- **标记为废弃**：`migrate()` 函数，实际迁移由 migrations 包处理

#### pay/database.go
- **移除**：所有 `AutoMigrate()` 调用（共14个）
- **添加注释**：说明迁移现在由 migrations 包统一管理

#### pay/badge_repository.go
- **移除**：`NewBadgeRepository()` 中的 `AutoMigrate()` 调用
- **标记为废弃**：`initPredefinedBadges()` 函数（实际初始化在 migrations 包中）

#### pay/oauth_repository.go
- **移除**：`NewOAuthRepository()` 中的 `AutoMigrate()` 调用
- **添加注释**：说明 OAuth 表映射到 mysql 包的表

#### pay/web_payment_models.go
- **标记为废弃**：`AutoMigrateWebPayments()` 函数
- **添加注释**：说明迁移现在由 migrations 包统一管理

### 3. 迁移步骤统计

#### 核心业务表（mysql 包）
共注册了 **80+** 个核心表的迁移步骤，包括：
- 用户相关：users, user_login_records, user_settings, user_devices 等
- 故事相关：stories, panels, story_contributors 等
- Storyboard 相关：storyboards, storyboard_content_generations 等
- 角色相关：characters, character_posters 等
- 群组相关：groups, group_members, group_roles 等
- 其他：comments, notifications, agents 等

#### 支付相关表（pay 包）
共注册了 **16** 个支付表的迁移步骤，包括：
- 支付记录：payment_records, subscriptions, user_subscriptions, web_payments
- IAP 相关：iap_products, apple_receipts, google_purchases 等
- 徽章系统：badges, user_badges, user_badge_stats
- Token 使用：token_usage_logs

#### Schema 修复步骤
共注册了 **12** 个 Schema 修复步骤，包括：
- ensureStoryboardVideoGenerationSchema
- ensureStoryboardScenesSchema
- ensureStoriesStyleSchema
- ensureAIGenerationRecordsSchema
- ensureGroupMembersSchema
- ensureGroupFollowsSchema
- ensureUserDevicesSchema
- ensureStoryboardImageGenerationSchema
- ensureStoryboardVideoGenerationPromptDetailsSchema
- ensureCharacterPortraitSchema
- ensureIsCollaborationOpenColumn
- ensureUserGroupCountColumns

#### 索引创建步骤
共注册了 **5** 个索引创建步骤，包括：
- web_payments 表的3个复合索引
- characters 表的 portrait_generation_status 索引
- stories 表的 is_collaboration_open 索引

#### 数据初始化步骤
共注册了 **1** 个数据初始化步骤：
- 初始化预定义徽章

## 迁移执行流程

1. **应用启动时**调用 `migrations.RunAllMigrations(ctx, db, log)`
2. **自动注册**：通过 `init()` 函数自动注册所有迁移步骤
3. **顺序执行**：
   - 核心业务表迁移
   - 支付相关表迁移
   - Schema 修复
   - 索引创建
   - 数据初始化

## 优势

1. **集中管理**：所有迁移逻辑集中在一个地方，易于维护
2. **避免重复**：不再有分散的 AutoMigrate 调用
3. **统一日志**：所有迁移步骤都有统一的日志记录
4. **错误处理**：区分必需和可选的迁移步骤
5. **易于扩展**：添加新表只需在对应的 register 文件中添加一步

## 向后兼容性

- 所有旧的迁移函数都标记为废弃，但保留代码以避免编译错误
- 旧的 AutoMigrate 调用已移除，但功能由新的迁移系统提供
- 数据初始化逻辑已迁移到新的迁移系统中

## 下一步

1. 在应用启动代码中调用 `migrations.RunAllMigrations()`
2. 测试迁移系统是否正常工作
3. 监控迁移日志，确保所有步骤成功执行
4. 根据实际使用情况调整迁移步骤的 Required 标志

## 注意事项

1. **首次迁移**：确保在应用启动时调用迁移函数
2. **数据库权限**：确保数据库用户有足够的权限执行迁移
3. **备份数据**：在生产环境执行迁移前，建议备份数据库
4. **测试环境**：先在测试环境验证迁移系统

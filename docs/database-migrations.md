# Database Migrations

本文档记录 Grapery 后端数据库的所有 Migration 信息。

## Migration 007: Add follows, likes tables and update user_settings (2026-02-01)

### 概述

本 Migration 创建了多态关联的 follows 和 likes 表，并为 user_settings、stories 和 characters 表添加了新字段。

### 新表

#### follows
存储关注关系，支持多态关联（story, user, character）。

| Field | Type | Description |
|-------|------|-------------|
| id | VARCHAR(36) | 主键 |
| follower_id | VARCHAR(36) | 关注者用户ID |
| followable_type | VARCHAR(50) | 被关注对象类型: story, user, character |
| followable_id | VARCHAR(36) | 被关注对象ID |
| notifications_enabled | BOOLEAN | 是否接收通知 |
| created_at | BIGINT | 创建时间戳 |

**索引:**
- UNIQUE: (follower_id, followable_type, followable_id)
- INDEX: follower_id
- INDEX: (followable_type, followable_id)
- INDEX: created_at

#### likes
存储点赞关系，支持多态关联（story, character, storyboard_node, fragment, character_poster）。

| Field | Type | Description |
|-------|------|-------------|
| id | VARCHAR(36) | 主键 |
| user_id | VARCHAR(36) | 点赞用户ID |
| likeable_type | VARCHAR(50) | 被点赞对象类型: story, character, storyboard_node, fragment, character_poster |
| likeable_id | VARCHAR(36) | 被点赞对象ID |
| created_at | BIGINT | 创建时间戳 |

**索引:**
- UNIQUE: (user_id, likeable_type, likeable_id)
- INDEX: user_id
- INDEX: (likeable_type, likeable_id)
- INDEX: created_at

### 更新的表

#### user_settings
新增字段:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| font_size | VARCHAR(20) | 'medium' | 字体大小: small, medium, large |
| data_saver | BOOLEAN | FALSE | 省流量模式 |
| default_story_visibility | VARCHAR(50) | 'public' | 故事默认可见性: public, unlisted, private |
| default_fragment_visibility | VARCHAR(50) | 'public' | 碎片默认可见性: public, followers_only, private |
| allow_follow_from | VARCHAR(50) | 'everyone' | 谁可以关注我: everyone, followers_only, followers_of_followers, no_one |
| allow_comments_from | VARCHAR(50) | 'everyone' | 谁可以评论我: everyone, followers_only, no_one |
| allow_messages_from | VARCHAR(50) | 'followers_only' | 谁可以私信我: everyone, followers_only, no_one |
| show_read_receipts | BOOLEAN | TRUE | 显示已读回执 |
| ai_enabled | BOOLEAN | TRUE | 启用AI功能 |
| ai_data_sharing | BOOLEAN | TRUE | 允许使用数据改进AI |

#### stories
新增字段:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| ai_enabled | BOOLEAN | TRUE | 是否允许AI辅助 |

#### characters
新增字段:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| poster_creation_permission | VARCHAR(50) | 'creator_only' | 海报创建权限: creator_only, anyone |

### 相关文件

- SQL Migration: `internal/repository/mysql/migrations/007_add_follows_likes_settings.sql`
- Schema Fix: `internal/repository/mysql/migrations.go` (ensureStoriesAIEnabledColumn, ensureCharactersPosterCreationPermissionColumn)
- Model Update: `internal/repository/mysql/models.go` (Story, Character, UserSettings)
- Migration Registration: `internal/repository/mysql/migrations_register.go`

### 执行方式

1. **自动迁移**: 应用启动时，GORM AutoMigrate 会自动创建 follows 和 likes 表，以及 UserSettings 模型的新字段
2. **Schema Fix**: 对于 stories 和 characters 表的新字段，通过 `registerSchemaFixSteps` 中注册的 schema fix 步骤添加
3. **手动执行**: 如需手动执行 SQL，可运行：
   ```bash
   mysql -u username -p database_name < internal/repository/mysql/migrations/007_add_follows_likes_settings.sql
   ```

---

## 历史 Migrations

### Migration 001-006

之前的 migrations 已整合到 GORM AutoMigrate 系统中，主要包括：

- 核心实体表：users, stories, characters, storyboards, panels
- 关系表：user_follows, story_likes, story_follows, character_follows
- AI 生成记录：storyboard_content_generations, storyboard_image_generations, storyboard_video_generations
- Fragment 系统：fragments, fragment_likes, fragment_comments

详细的 schema 定义请参考 `internal/repository/mysql/models.go`。

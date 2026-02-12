# 数据库迁移计划 - 域模型重构

**文档版本**: 1.0
**创建日期**: 2026-02-11
**目标**: 为数据模型重构准备数据库迁移计划

---

## 概述

本文档列出了为支持域模型重构所需的所有数据库迁移，按照优先级排序。每个迁移包含：
- 迁移前状态描述
- 迁移 SQL 脚本
- 迁移后状态描述
- 回滚 SQL 脚本
- 潜在风险标记

---

## 迁移优先级说明

- **P0 - 关键**: 阻塞性迁移，必须优先执行
- **P1 - 高**: 重要迁移，影响核心功能
- **P2 - 中**: 中等重要性，影响次要功能
- **P3 - 低**: 低优先级，可选优化

---

## 迁移列表

### MIGRATION-001: 重命名 comments.author_id → user_id (P0)

**影响范围**: 评论表
**破坏性**: 是 (需要更新所有查询代码)

#### 迁移前状态
```sql
CREATE TABLE comments (
    id VARCHAR(36) PRIMARY KEY,
    author_id VARCHAR(36) NOT NULL,  -- 旧字段名
    content TEXT NOT NULL,
    target_type VARCHAR(20) NOT NULL,
    target_id VARCHAR(36) NOT NULL,
    ...
    INDEX idx_author_id (author_id)
);
```

#### 迁移 SQL
```sql
-- 步骤1: 删除旧索引
DROP INDEX idx_author_id ON comments;

-- 步骤2: 重命名列
ALTER TABLE comments
    CHANGE COLUMN author_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- 步骤3: 创建新索引
CREATE INDEX idx_user_id ON comments(user_id);

-- 步骤4: 更新外键约束（如果存在）
-- 首先删除旧外键
ALTER TABLE comments DROP FOREIGN KEY fk_comment_author;

-- 添加新外键
ALTER TABLE comments
    ADD CONSTRAINT fk_comment_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 迁移后状态
```sql
CREATE TABLE comments (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,  -- 新字段名
    content TEXT NOT NULL,
    target_type VARCHAR(20) NOT NULL,
    target_id VARCHAR(36) NOT NULL,
    ...
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_comment_user FOREIGN KEY (user_id) REFERENCES users(id)
);
```

#### 回滚 SQL
```sql
-- 回滚步骤（按相反顺序）
ALTER TABLE comments DROP FOREIGN KEY fk_comment_user;
DROP INDEX idx_user_id ON comments;
ALTER TABLE comments
    CHANGE COLUMN user_id author_id VARCHAR(36) NOT NULL COMMENT '作者ID';
CREATE INDEX idx_author_id ON comments(author_id);
ALTER TABLE comments
    ADD CONSTRAINT fk_comment_author
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 风险说明
- ⚠️ **需要停机**: 所有依赖 author_id 的查询代码需要同步更新
- ⚠️ **数据完整性**: 确保外键约束正确迁移
- ⚠️ **索引重建**: 大表重建索引可能耗时较长

---

### MIGRATION-002: 重命名 character_posters.author_id → user_id (P0)

**影响范围**: 角色海报表
**破坏性**: 是 (需要更新所有查询代码)

#### 迁移前状态
```sql
CREATE TABLE character_posters (
    id VARCHAR(36) PRIMARY KEY,
    character_id VARCHAR(36) NOT NULL,
    author_id VARCHAR(36) NOT NULL,  -- 旧字段名
    ...
    INDEX idx_author_id (author_id)
);
```

#### 迁移 SQL
```sql
-- 步骤1: 删除旧索引
DROP INDEX idx_author_id ON character_posters;

-- 步骤2: 重命名列
ALTER TABLE character_posters
    CHANGE COLUMN author_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- 步骤3: 创建新索引
CREATE INDEX idx_user_id ON character_posters(user_id);

-- 步骤4: 更新外键约束
ALTER TABLE character_posters DROP FOREIGN KEY fk_poster_author;
ALTER TABLE character_posters
    ADD CONSTRAINT fk_poster_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 迁移后状态
```sql
CREATE TABLE character_posters (
    id VARCHAR(36) PRIMARY KEY,
    character_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,  -- 新字段名
    ...
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_poster_user FOREIGN KEY (user_id) REFERENCES users(id)
);
```

#### 回滚 SQL
```sql
ALTER TABLE character_posters DROP FOREIGN KEY fk_poster_user;
DROP INDEX idx_user_id ON character_posters;
ALTER TABLE character_posters
    CHANGE COLUMN user_id author_id VARCHAR(36) NOT NULL COMMENT '作者ID';
CREATE INDEX idx_author_id ON character_posters(author_id);
ALTER TABLE character_posters
    ADD CONSTRAINT fk_poster_author
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 风险说明
- ⚠️ **需要停机**: 所有依赖 author_id 的查询代码需要同步更新
- ⚠️ **关联查询**: 检查所有 JOIN 查询中的字段引用

---

### MIGRATION-003: 重命名 stories.author_id → user_id (P0)

**影响范围**: 故事表
**破坏性**: 是 (核心表，高影响)

#### 迁移前状态
```sql
CREATE TABLE stories (
    id VARCHAR(36) PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    author_id VARCHAR(36) NOT NULL,  -- 旧字段名
    ...
    INDEX idx_author_id (author_id),
    CONSTRAINT fk_story_author FOREIGN KEY (author_id) REFERENCES users(id)
);
```

#### 迁移 SQL
```sql
-- 步骤1: 备份数据（可选）
-- CREATE TABLE stories_backup AS SELECT * FROM stories;

-- 步骤2: 删除旧索引
DROP INDEX idx_author_id ON stories;

-- 步骤3: 删除旧外键
ALTER TABLE stories DROP FOREIGN KEY fk_story_author;

-- 步骤4: 重命名列
ALTER TABLE stories
    CHANGE COLUMN author_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- 步骤5: 创建新索引
CREATE INDEX idx_user_id ON stories(user_id);

-- 步骤6: 添加新外键
ALTER TABLE stories
    ADD CONSTRAINT fk_story_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 迁移后状态
```sql
CREATE TABLE stories (
    id VARCHAR(36) PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    user_id VARCHAR(36) NOT NULL,  -- 新字段名
    ...
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_story_user FOREIGN KEY (user_id) REFERENCES users(id)
);
```

#### 回滚 SQL
```sql
ALTER TABLE stories DROP FOREIGN KEY fk_story_user;
DROP INDEX idx_user_id ON stories;
ALTER TABLE stories
    CHANGE COLUMN user_id author_id VARCHAR(36) NOT NULL COMMENT '作者ID';
CREATE INDEX idx_author_id ON stories(author_id);
ALTER TABLE stories
    ADD CONSTRAINT fk_story_author
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 风险说明
- 🔴 **高风险**: 这是核心表，影响广泛
- ⚠️ **需要停机**: 必须在低峰期执行
- ⚠️ **级联影响**: 需要同时更新所有子表的外键引用
- ⚠️ **关联表**: 以下表可能需要更新引用
  - story_contributors
  - story_likes
  - story_follows
  - story_publications
  - panels
  - storyboards

---

### MIGRATION-004: 重命名 characters.author_id → user_id (P0)

**影响范围**: 角色表
**破坏性**: 是

#### 迁移前状态
```sql
CREATE TABLE characters (
    id VARCHAR(36) PRIMARY KEY,
    story_id VARCHAR(36) NOT NULL,
    author_id VARCHAR(36) NOT NULL,  -- 旧字段名
    ...
    INDEX idx_author_id (author_id)
);
```

#### 迁移 SQL
```sql
-- 步骤1: 删除旧索引
DROP INDEX idx_author_id ON characters;

-- 步骤2: 重命名列
ALTER TABLE characters
    CHANGE COLUMN author_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- 步骤3: 创建新索引
CREATE INDEX idx_user_id ON characters(user_id);

-- 步骤4: 更新外键
ALTER TABLE characters DROP FOREIGN KEY fk_character_author;
ALTER TABLE characters
    ADD CONSTRAINT fk_character_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 迁移后状态
```sql
CREATE TABLE characters (
    id VARCHAR(36) PRIMARY KEY,
    story_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,  -- 新字段名
    ...
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_character_user FOREIGN KEY (user_id) REFERENCES users(id)
);
```

#### 回滚 SQL
```sql
ALTER TABLE characters DROP FOREIGN KEY fk_character_user;
DROP INDEX idx_user_id ON characters;
ALTER TABLE characters
    CHANGE COLUMN user_id author_id VARCHAR(36) NOT NULL COMMENT '作者ID';
CREATE INDEX idx_author_id ON characters(author_id);
ALTER TABLE characters
    ADD CONSTRAINT fk_character_author
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 风险说明
- ⚠️ **需要停机**: 角色表查询较多
- ⚠️ **关联表检查**: 以下表可能需要更新
  - character_follows
  - character_posters (已单独处理)
  - character_analytics
  - agents

---

### MIGRATION-005: 重命名 storyboards.creator_id → user_id (P1)

**影响范围**: 故事板表
**破坏性**: 是

#### 迁移前状态
```sql
CREATE TABLE storyboards (
    id VARCHAR(36) PRIMARY KEY,
    story_id VARCHAR(36) NOT NULL,
    creator_id VARCHAR(36) NOT NULL,  -- 旧字段名
    ...
    INDEX idx_creator_id (creator_id)
);
```

#### 迁移 SQL
```sql
-- 步骤1: 删除旧索引
DROP INDEX idx_creator_id ON storyboards;

-- 步骤2: 重命名列
ALTER TABLE storyboards
    CHANGE COLUMN creator_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- 步骤3: 创建新索引
CREATE INDEX idx_user_id ON storyboards(user_id);

-- 步骤4: 更新外键
ALTER TABLE storyboards DROP FOREIGN KEY fk_storyboard_creator;
ALTER TABLE storyboards
    ADD CONSTRAINT fk_storyboard_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 迁移后状态
```sql
CREATE TABLE storyboards (
    id VARCHAR(36) PRIMARY KEY,
    story_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,  -- 新字段名
    ...
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_storyboard_user FOREIGN KEY (user_id) REFERENCES users(id)
);
```

#### 回滚 SQL
```sql
ALTER TABLE storyboards DROP FOREIGN KEY fk_storyboard_user;
DROP INDEX idx_user_id ON storyboards;
ALTER TABLE storyboards
    CHANGE COLUMN user_id creator_id VARCHAR(36) NOT NULL COMMENT '创建者ID';
CREATE INDEX idx_creator_id ON storyboards(creator_id);
ALTER TABLE storyboards
    ADD CONSTRAINT fk_storyboard_creator
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 风险说明
- ⚠️ **需要停机**: 故事板表是高频访问表
- ⚠️ **关联表检查**:
  - storyboard_content_generations
  - storyboard_scene_generations
  - storyboard_image_generations
  - storyboard_video_generations
  - storyboard_likes

---

### MIGRATION-006: 重命名 fragments.creator_id → user_id (P1)

**影响范围**: 碎片表
**破坏性**: 是

#### 迁移前状态
```sql
CREATE TABLE fragments (
    id VARCHAR(36) PRIMARY KEY,
    creator_id VARCHAR(36) NOT NULL,  -- 旧字段名
    content TEXT,
    ...
    INDEX idx_fragment_creator (creator_id)
);
```

#### 迁移 SQL
```sql
-- 步骤1: 删除旧索引
DROP INDEX idx_fragment_creator ON fragments;

-- 步骤2: 重命名列
ALTER TABLE fragments
    CHANGE COLUMN creator_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- 步骤3: 创建新索引
CREATE INDEX idx_user_id ON fragments(user_id);

-- 步骤4: 更新外键
ALTER TABLE fragments DROP FOREIGN KEY fk_fragment_creator;
ALTER TABLE fragments
    ADD CONSTRAINT fk_fragment_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 迁移后状态
```sql
CREATE TABLE fragments (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,  -- 新字段名
    content TEXT,
    ...
    INDEX idx_user_id (user_id),
    CONSTRAINT fk_fragment_user FOREIGN KEY (user_id) REFERENCES users(id)
);
```

#### 回滚 SQL
```sql
ALTER TABLE fragments DROP FOREIGN KEY fk_fragment_user;
DROP INDEX idx_user_id ON fragments;
ALTER TABLE fragments
    CHANGE COLUMN user_id creator_id VARCHAR(36) NOT NULL COMMENT '创建者ID';
CREATE INDEX idx_fragment_creator ON fragments(creator_id);
ALTER TABLE fragments
    ADD CONSTRAINT fk_fragment_creator
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE;
```

#### 风险说明
- ⚠️ **需要停机**: 碎片表有外键关联
- ⚠️ **关联表检查**:
  - fragment_likes
  - fragment_comments
  - fragment_shares

---

### MIGRATION-007: 更新 story_scenes.created_by 字段 (P2)

**影响范围**: 故事场景表
**破坏性**: 是

**说明**: `created_by` 和 `last_edited_by` 字段保持不变，但需确保一致性。此迁移主要是检查和优化，不进行重命名。

#### 检查 SQL
```sql
-- 检查是否有孤儿记录（引用不存在用户）
SELECT id, title, created_by
FROM story_scenes
WHERE created_by NOT IN (SELECT id FROM users);

-- 如果有孤儿记录，考虑设置为系统用户或 NULL
-- UPDATE story_scenes SET created_by = 'SYSTEM_USER_ID'
-- WHERE created_by NOT IN (SELECT id FROM users);
```

#### 优化 SQL
```sql
-- 确保 created_by 字段有索引
CREATE INDEX IF NOT EXISTS idx_created_by ON story_scenes(created_by);

-- 确保 last_edited_by 字段有索引
CREATE INDEX IF NOT EXISTS idx_last_edited_by ON story_scenes(last_edited_by);
```

#### 风险说明
- ℹ️ **低风险**: 主要是优化性迁移
- ℹ️ **可在线执行**: 不需要停机

---

### MIGRATION-008: 更新 notifications.actor_id 字段 (P2)

**影响范围**: 通知表
**破坏性**: 否

**说明**: `actor_id` 字段保持不变，但需确保索引优化。

#### 迁移 SQL
```sql
-- 确保 actor_id 有索引（如果尚未存在）
CREATE INDEX IF NOT EXISTS idx_actor_id ON notifications(actor_id);

-- 检查是否有孤儿记录
SELECT id, type, actor_id
FROM notifications
WHERE actor_id IS NOT NULL
  AND actor_id != ''
  AND actor_id NOT IN (SELECT id FROM users);
```

#### 风险说明
- ℹ️ **低风险**: 仅索引优化
- ℹ️ **可在线执行**: 不需要停机

---

### MIGRATION-009: 统一状态字段类型 (P2)

**影响范围**: 多个表
**破坏性**: 是

**说明**: 确保所有状态字段使用一致的 VARCHAR 长度和类型。

#### 需要检查的表和字段

| 表名 | 字段名 | 当前类型 | 目标类型 |
|------|--------|----------|----------|
| stories | status | VARCHAR(20) | VARCHAR(20) ✓ |
| storyboards | workflow_status | VARCHAR(20) | VARCHAR(20) ✓ |
| characters | portrait_generation_status | VARCHAR(20) | VARCHAR(20) ✓ |
| ai_tasks | status | VARCHAR(20) | VARCHAR(20) ✓ |
| ai_generation_records | status | VARCHAR(20) | VARCHAR(20) ✓ |
| render_tasks | status | VARCHAR(20) | VARCHAR(20) ✓ |
| character_posters | status | VARCHAR(20) | VARCHAR(20) ✓ |
| subscription_orders | status | VARCHAR(20) | VARCHAR(20) ✓ |
| reports | status | VARCHAR(20) | VARCHAR(20) ✓ |

#### 验证 SQL
```sql
-- 验证所有状态字段类型一致性
SELECT
    TABLE_NAME,
    COLUMN_NAME,
    DATA_TYPE,
    CHARACTER_MAXIMUM_LENGTH
FROM INFORMATION_SCHEMA.COLUMNS
WHERE COLUMN_NAME = 'status'
  AND TABLE_SCHEMA = DATABASE()
ORDER BY TABLE_NAME;
```

#### 风险说明
- ℹ️ **低风险**: 当前状态字段类型已基本一致
- ℹ️ **可在线执行**: 仅需验证，无需修改

---

### MIGRATION-010: 索引优化和清理 (P3)

**影响范围**: 所有表
**破坏性**: 否

**说明**: 清理未使用的索引，添加缺失的复合索引。

#### 索引检查 SQL
```sql
-- 检查重复或未使用的索引
SELECT
    TABLE_NAME,
    INDEX_NAME,
    GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) as columns
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
GROUP BY TABLE_NAME, INDEX_NAME
HAVING COUNT(*) > 1;
```

#### 建议添加的复合索引

```sql
-- comments 表：优化按目标和父评论的查询
CREATE INDEX IF NOT EXISTS idx_target ON comments(target_type, target_id);

-- storyboards 表：优化树状结构查询
CREATE INDEX IF NOT EXISTS idx_tree ON storyboards(story_id, parent_id);

-- fragments 表：优化按创建者和可见性查询
CREATE INDEX IF NOT EXISTS idx_user_visibility ON fragments(user_id, visibility);

-- likes 表：优化多态查询
CREATE INDEX IF NOT EXISTS idx_likeable ON likes(likeable_type, likeable_id);

-- follows 表：优化多态查询
CREATE INDEX IF NOT EXISTS idx_followable ON follows(followable_type, followable_id);
```

#### 风险说明
- ⚠️ **中等风险**: 添加索引可能影响写入性能
- ℹ️ **可在线执行**: 可以在低峰期执行
- ℹ️ **需要监控**: 执行后监控查询性能变化

---

## 执行计划

### 阶段 1: 准备阶段（执行前 1 周）

1. **代码审查**
   - [ ] 识别所有使用旧字段名的代码位置
   - [ ] 准备代码更新分支
   - [ ] 编写单元测试覆盖迁移场景

2. **数据验证**
   - [ ] 备份生产数据库
   - [ ] 在测试环境执行所有迁移
   - [ ] 验证外键约束完整性
   - [ ] 检查孤儿记录

3. **性能评估**
   - [ ] 评估各表数据量
   - [ ] 估算迁移时间
   - [ ] 准备回滚方案

### 阶段 2: 执行阶段（维护窗口）

#### 执行顺序（按依赖关系）

1. **MIGRATION-007**: story_scenes 优化（可先执行，无依赖）
2. **MIGRATION-008**: notifications 优化（可先执行，无依赖）
3. **MIGRATION-001**: comments 表
4. **MIGRATION-002**: character_posters 表
5. **MIGRATION-003**: stories 表（注意：此表影响最广）
6. **MIGRATION-004**: characters 表
7. **MIGRATION-005**: storyboards 表
8. **MIGRATION-006**: fragments 表
9. **MIGRATION-009**: 状态字段验证
10. **MIGRATION-010**: 索引优化（可在系统运行时执行）

#### 时间估算（假设 100万 条记录）

| 迁移 ID | 表名 | 估算时间 | 说明 |
|---------|------|----------|------|
| 001 | comments | 5-10 分钟 | 小表 |
| 002 | character_posters | 5-10 分钟 | 小表 |
| 003 | stories | 30-60 分钟 | 核心表，需要更长停机时间 |
| 004 | characters | 15-30 分钟 | 中等表 |
| 005 | storyboards | 20-40 分钟 | 大表 |
| 006 | fragments | 10-20 分钟 | 中小表 |
| 007-010 | - | 10-20 分钟 | 优化类迁移 |

**总停机时间**: 约 2-3 小时（包含验证时间）

### 阶段 3: 验证阶段（执行后）

1. **数据完整性检查**
   ```sql
   -- 检查外键约束
   SELECT TABLE_NAME, CONSTRAINT_NAME
   FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
   WHERE TABLE_SCHEMA = DATABASE()
   AND REFERENCED_TABLE_NAME = 'users';

   -- 检查索引
   SHOW INDEX FROM comments;
   SHOW INDEX FROM stories;
   SHOW INDEX FROM characters;
   -- ... 其他表
   ```

2. **功能测试**
   - [ ] 测试评论功能
   - [ ] 测试故事创建/编辑
   - [ ] 测试角色管理
   - [ ] 测试故事板操作
   - [ ] 测试碎片功能

3. **性能监控**
   - [ ] 监控慢查询日志
   - [ ] 检查索引使用情况
   - [ ] 验证查询响应时间

---

## 回滚计划

如果迁移失败，按以下顺序回滚：

1. 停止应用服务
2. 按迁移相反顺序执行回滚 SQL
3. 从备份恢复数据（如果 SQL 回滚失败）
4. 重启应用服务
5. 执行健康检查

### 回滚决策点

- 如果任何迁移失败超过 30 分钟，立即回滚
- 如果数据完整性检查失败，立即回滚
- 如果应用启动失败，立即回滚

---

## 注意事项

### 关键风险

1. **外键约束**: 所有带外键的表需要先删除约束，重命名后再重建
2. **索引依赖**: 复合索引可能受字段重命名影响
3. **触发器**: 检查是否有触发器依赖旧字段名
4. **视图**: 检查是否有视图引用旧字段名
5. **存储过程**: 检查存储过程和函数中的字段引用

### 最佳实践

1. **分步执行**: 不要一次性执行所有迁移
2. **充分测试**: 在测试环境完整验证
3. **保留备份**: 确保有完整的数据库备份
4. **监控日志**: 执行过程中监控数据库日志
5. **团队协作**: 确保开发和运维团队同步

### 检查清单

#### 迁移前
- [ ] 代码已更新并合并到主分支
- [ ] 测试环境验证通过
- [ ] 数据库备份已完成
- [ ] 回滚计划已准备
- [ ] 团队已通知
- [ ] 监控系统已就绪

#### 迁移中
- [ ] 按顺序执行迁移
- [ ] 记录每个迁移的执行时间
- [ ] 监控数据库性能指标
- [ ] 检查错误日志

#### 迁移后
- [ ] 数据完整性验证
- [ ] 功能测试通过
- [ ] 性能监控正常
- [ ] 文档已更新
- [ ] 团队已通知完成

---

## 附录

### A. 受影响的代码模块

以下模块可能需要更新以适应字段重命名：

1. **Repository 层**
   - CommentRepository
   - StoryRepository
   - CharacterRepository
   - StoryboardRepository
   - FragmentRepository
   - CharacterPosterRepository

2. **Service 层**
   - CommentService
   - StoryService
   - CharacterService
   - StoryboardService
   - FragmentService

3. **API 层**
   - 所有返回 DTO/Response 对象
   - 请求参数验证

### B. 相关文档

- [GORM 迁移指南](https://gorm.io/docs/migration.html)
- [MySQL ALTER TABLE 语法](https://dev.mysql.com/doc/refman/8.0/en/alter-table.html)
- 项目数据库设计文档

### C. 联系人

- **数据库负责人**: [待填写]
- **后端负责人**: [待填写]
- **运维负责人**: [待填写]

---

**文档结束**

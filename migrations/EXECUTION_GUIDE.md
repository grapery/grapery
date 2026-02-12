# 数据库迁移执行指南

**文档版本**: 1.0
**创建日期**: 2026-02-11
**适用范围**: 生产环境数据库迁移

---

## 概述

本指南提供了执行数据库迁移的详细步骤和注意事项。在执行任何迁移之前，请确保已经：

1. 阅读并理解 `REFACTORING_PLAN.md`
2. 在测试环境验证所有迁移
3. 完成代码更新以匹配新的数据库结构
4. 通知所有相关团队成员

---

## 迁移文件清单

| 文件名 | 版本 | 描述 | 优先级 | 破坏性 |
|--------|------|------|--------|--------|
| 009_rename_author_id_to_user_id_comments.sql | 009 | 重命名 comments.author_id | P0 | 是 |
| 010_rename_author_id_to_user_id_character_posters.sql | 010 | 重命名 character_posters.author_id | P0 | 是 |
| 011_rename_author_id_to_user_id_stories.sql | 011 | 重命名 stories.author_id | P0 | 是 |
| 012_rename_author_id_to_user_id_characters.sql | 012 | 重命名 characters.author_id | P0 | 是 |
| 013_rename_creator_id_to_user_id_storyboards.sql | 013 | 重命名 storyboards.creator_id | P1 | 是 |
| 014_rename_creator_id_to_user_id_fragments.sql | 014 | 重命名 fragments.creator_id | P1 | 是 |
| 015_add_composite_indexes.sql | 015 | 添加复合索引 | P3 | 否 |

---

## 执行前准备

### 1. 环境检查

```bash
# 检查 MySQL 版本
mysql --version

# 检查数据库大小
mysql -u root -p -e "
SELECT
    TABLE_NAME,
    ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024), 2) AS 'Size in MB',
    TABLE_ROWS
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = 'your_database_name'
ORDER BY DATA_LENGTH DESC;
"
```

### 2. 备份数据库

```bash
# 完整备份
mysqldump -u root -p --single-transaction --routines --triggers --events \
    your_database_name > backup_$(date +%Y%m%d_%H%M%S).sql

# 或使用 Percona XtraBackup（推荐用于大型数据库）
# innobackupex --user=root --password=xxx /backup/dir
```

### 3. 准备回滚脚本

```bash
# 将所有迁移文件中的 ROLLBACK 部分提取到单独的文件
# grep -A 20 "MIGRATION DOWN" migrations/*.sql > rollback_$(date +%Y%m%d).sql
```

### 4. 停止应用服务

```bash
# 停止应用服务（根据你的部署方式调整）
systemctl stop grapery-api
# 或
docker-compose down
```

---

## 执行步骤

### 方式 1: 使用迁移系统（推荐）

如果你的项目使用迁移系统：

```bash
# 进入项目目录
cd /Users/grapestree/Desktop/work/fgrapery/grapery

# 注册迁移（如果需要）
# 更新 migrations_register.go 添加新迁移

# 执行迁移
go run cmd/migrate/main.go up

# 或使用 Make 命令
make migrate-up
```

### 方式 2: 手动执行 SQL

如果需要手动执行：

```bash
# 1. 连接到数据库
mysql -u root -p your_database_name

# 2. 设置安全模式（可选，防止误操作）
SET sql_safe_updates = 1;

# 3. 按顺序执行迁移文件
source /Users/grapestree/Desktop/work/fgrapery/grapery/migrations/009_rename_author_id_to_user_id_comments.sql
source /Users/grapestree/Desktop/work/fgrapery/grapery/migrations/010_rename_author_id_to_user_id_character_posters.sql
source /Users/grapestree/Desktop/work/fgrapery/grapery/migrations/011_rename_author_id_to_user_id_stories.sql
source /Users/grapestree/Desktop/work/fgrapery/grapery/migrations/012_rename_author_id_to_user_id_characters.sql
source /Users/grapestree/Desktop/work/fgrapery/grapery/migrations/013_rename_creator_id_to_user_id_storyboards.sql
source /Users/grapestree/Desktop/work/fgrapery/grapery/migrations/014_rename_creator_id_to_user_id_fragments.sql

# 4. 执行索引优化（可在系统运行后执行）
source /Users/grapestree/Desktop/work/fgrapery/grapery/migrations/015_add_composite_indexes.sql
```

---

## 执行时间估算

基于 100 万条记录的表：

| 迁移文件 | 表名 | 预估时间 | 备注 |
|----------|------|----------|------|
| 009 | comments | 5-10 分钟 | |
| 010 | character_posters | 5-10 分钟 | |
| 011 | stories | 30-60 分钟 | 核心表，可能需要更长时间 |
| 012 | characters | 15-30 分钟 | |
| 013 | storyboards | 20-40 分钟 | |
| 014 | fragments | 10-20 分钟 | |
| 015 | (多个表) | 10-20 分钟 | 可在系统运行时执行 |

**总计**: 约 2-3 小时（包括验证时间）

---

## 验证检查

### 执行后立即验证

```sql
-- 1. 验证所有列重命名成功
SELECT TABLE_NAME, COLUMN_NAME
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
AND COLUMN_NAME = 'user_id'
AND TABLE_NAME IN ('comments', 'character_posters', 'stories', 'characters', 'storyboards', 'fragments');

-- 2. 验证所有外键约束存在
SELECT
    TABLE_NAME,
    CONSTRAINT_NAME,
    REFERENCED_TABLE_NAME
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = DATABASE()
AND REFERENCED_TABLE_NAME = 'users'
AND TABLE_NAME IN ('comments', 'character_posters', 'stories', 'characters', 'storyboards', 'fragments');

-- 3. 验证所有索引存在
SELECT
    TABLE_NAME,
    INDEX_NAME,
    GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) as columns
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
AND INDEX_NAME IN ('idx_user_id', 'idx_target', 'idx_tree', 'idx_user_visibility')
GROUP BY TABLE_NAME, INDEX_NAME;
```

### 应用启动验证

```bash
# 1. 启动应用
systemctl start grapery-api

# 2. 检查日志
tail -f /var/log/grapery/app.log

# 3. 检查错误
grep -i error /var/log/grapery/app.log
```

### 功能验证清单

- [ ] 用户可以创建评论
- [ ] 用户可以查看故事
- [ ] 用户可以创建角色
- [ ] 用户可以创建故事板
- [ ] 用户可以发布碎片
- [ ] 通知系统正常工作
- [ ] 搜索功能正常
- [ ] API 响应时间正常

---

## 性能监控

### 监控慢查询

```sql
-- 启用慢查询日志
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 2; -- 记录执行时间超过 2 秒的查询

-- 查看慢查询
SELECT * FROM mysql.slow_log ORDER BY query_time DESC LIMIT 20;
```

### 监控索引使用

```sql
-- 查看索引使用统计
SELECT
    TABLE_NAME,
    INDEX_NAME,
    ROWS_READ,
    ROWS_REQUESTED
FROM performance_schema.table_io_waits_summary_by_index_usage
WHERE TABLE_SCHEMA = DATABASE()
ORDER BY ROWS_READ DESC;
```

### 监控连接数

```sql
-- 查看当前连接
SHOW PROCESSLIST;

-- 查看连接统计
SHOW STATUS LIKE 'Threads_connected';
SHOW STATUS LIKE 'Max_used_connections';
```

---

## 回滚程序

### 判断回滚条件

- 迁移执行时间超过预期 2 倍
- 数据完整性检查失败
- 应用无法启动
- 严重性能下降

### 回滚步骤

```bash
# 1. 停止应用
systemctl stop grapery-api

# 2. 执行回滚 SQL（按相反顺序）
mysql -u root -p your_database_name < rollback_20260211.sql

# 3. 如果 SQL 回滚失败，从备份恢复
mysql -u root -p your_database_name < backup_20260211_hhmmss.sql

# 4. 重启应用
systemctl start grapery-api

# 5. 验证应用正常
curl http://localhost:8080/health
```

---

## 故障排除

### 常见问题

#### 1. 外键约束错误

**错误信息**: `Cannot delete or update a parent row: a foreign key constraint fails`

**解决方案**:
```sql
-- 临时禁用外键检查
SET FOREIGN_KEY_CHECKS = 0;

-- 执行迁移

-- 重新启用外键检查
SET FOREIGN_KEY_CHECKS = 1;
```

#### 2. 索引创建超时

**错误信息**: `Lock wait timeout exceeded`

**解决方案**:
```sql
-- 增加锁等待超时时间
SET SESSION innodb_lock_wait_timeout = 300;

-- 或在低峰期执行
```

#### 3. 表太大导致 ALTER TABLE 超时

**解决方案**:
```bash
# 使用 pt-online-schema-change（Percona Toolkit）
pt-online-schema-change \
  --alter "CHANGE COLUMN author_id user_id VARCHAR(36) NOT NULL" \
  --user=root \
  --password=xxx \
  D=your_database_name,t=stories \
  --execute
```

---

## 代码更新清单

完成数据库迁移后，需要更新以下代码：

### Go 代码

```go
// 更新模型定义
// /internal/domain/comment_models.go
type Comment struct {
    ID       string `json:"id"`
    UserID   string `json:"userId"`  // 旧: AuthorID
    // ...
}

// /internal/repository/mysql/models.go
type Comment struct {
    ID     string `gorm:"primaryKey;size:36"`
    UserID string `gorm:"size:36;not null;index"`  // 旧: AuthorID
    User   User   `gorm:"foreignKey:UserID"`  // 旧: AuthorID
    // ...
}
```

### 需要更新的文件

- [ ] `/internal/domain/comment_models.go`
- [ ] `/internal/domain/character_models.go`
- [ ] `/internal/domain/story_models.go`
- [ ] `/internal/domain/storyboard_models.go`
- [ ] `/internal/domain/fragment_models.go`
- [ ] `/internal/repository/mysql/models.go`
- [ ] 所有 Repository 层文件
- [ ] 所有 Service 层文件
- [ ] API 请求/响应结构

---

## 联系信息

| 角色 | 姓名 | 联系方式 |
|------|------|----------|
| 数据库负责人 | [待填写] | [待填写] |
| 后端负责人 | [待填写] | [待填写] |
| 运维负责人 | [待填写] | [待填写] |
| 值班电话 | [待填写] | [待填写] |

---

## 附录

### A. 有用的 MySQL 命令

```sql
-- 查看表结构
SHOW CREATE TABLE table_name;

-- 查看索引
SHOW INDEX FROM table_name;

-- 查看外键
SELECT * FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_NAME = 'table_name';

-- 查看表大小
SELECT
    TABLE_NAME,
    ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024), 2) AS 'Size in MB'
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = DATABASE();
```

### B. 恢复测试

在非生产环境测试恢复流程：

```bash
# 1. 创建测试数据库
mysql -u root -p -e "CREATE DATABASE test_migration;"

# 2. 导入备份
mysql -u root -p test_migration < backup_20260211_hhmmss.sql

# 3. 在测试环境执行迁移
# ... 执行迁移步骤 ...

# 4. 验证结果
# ... 验证步骤 ...
```

---

**文档结束**

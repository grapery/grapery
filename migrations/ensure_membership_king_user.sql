-- 为 King 测试用户 6492a794-5c56-40a6-9979-b5aeafc4b2f3 写入/补齐 memberships 行
-- 用法: mysql -u... -p... grapery < migrations/ensure_membership_king_user.sql
-- 依赖: users 表中已存在该 user_id（可先执行 king_mock_data.sql 第一节）

SET NAMES utf8mb4 COLLATE utf8mb4_0900_ai_ci;

SET @king_id = '6492a794-5c56-40a6-9979-b5aeafc4b2f3';

INSERT INTO memberships (
  id, user_id, tier, status, start_date, end_date, auto_renew,
  token_quota, token_used, storage_quota, storage_used,
  created_at, updated_at, deleted_at
) VALUES (
  UUID(),
  @king_id,
  'free',
  'active',
  FROM_UNIXTIME(UNIX_TIMESTAMP()),
  NULL,
  0,
  25000,
  0,
  104857600,
  0,
  FROM_UNIXTIME(UNIX_TIMESTAMP()),
  FROM_UNIXTIME(UNIX_TIMESTAMP()),
  NULL
)
ON DUPLICATE KEY UPDATE
  deleted_at = NULL,
  status = 'active',
  token_quota = GREATEST(memberships.token_quota, 25000),
  storage_quota = GREATEST(memberships.storage_quota, 104857600),
  updated_at = FROM_UNIXTIME(UNIX_TIMESTAMP());

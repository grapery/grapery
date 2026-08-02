-- 第三方登录相关表结构
-- 用于存储第三方登录信息，建立第三方用户ID与系统用户ID的映射关系

-- 第三方登录表
CREATE TABLE IF NOT EXISTS `third_party_logins` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `user_id` bigint(20) NOT NULL COMMENT '系统用户ID',
  `provider` varchar(50) NOT NULL COMMENT '第三方提供商（apple, google, facebook, wechat, alipay）',
  `provider_user_id` varchar(255) NOT NULL COMMENT '第三方用户ID',
  `provider_email` varchar(255) DEFAULT NULL COMMENT '第三方用户邮箱',
  `provider_user_name` varchar(255) DEFAULT NULL COMMENT '第三方用户名称',
  `provider_user_info` json DEFAULT NULL COMMENT '第三方用户信息JSON（扩展信息）',
  `access_token` varchar(500) DEFAULT NULL COMMENT '第三方访问令牌',
  `refresh_token` varchar(500) DEFAULT NULL COMMENT '第三方刷新令牌',
  `token_expire_time` datetime DEFAULT NULL COMMENT '令牌过期时间',
  `status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '状态：1-正常，2-禁用',
  `create_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(4) NOT NULL DEFAULT '0' COMMENT '删除标记：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_provider_user_id` (`provider`,`provider_user_id`) COMMENT '第三方提供商和用户ID的唯一约束',
  KEY `idx_user_id` (`user_id`) COMMENT '用户ID索引',
  KEY `idx_provider` (`provider`) COMMENT '提供商索引',
  KEY `idx_provider_email` (`provider_email`) COMMENT '第三方邮箱索引',
  KEY `idx_status` (`status`) COMMENT '状态索引',
  KEY `idx_create_at` (`create_at`) COMMENT '创建时间索引'
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='第三方登录信息表';

-- 添加外键约束（如果需要）
-- ALTER TABLE `third_party_logins` ADD CONSTRAINT `fk_third_party_logins_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE ON UPDATE CASCADE;

-- 插入示例数据（可选）
-- INSERT INTO `third_party_logins` (`user_id`, `provider`, `provider_user_id`, `provider_email`, `provider_user_name`, `provider_user_info`, `status`) VALUES
-- (1000001, 'apple', '001234.abc123def456.1234', 'user@example.com', 'John Doe', '{"email_verified":true,"is_private_email":false}', 1),
-- (1000002, 'google', 'google_user_123', 'user2@gmail.com', 'Jane Smith', '{"email_verified":true,"avatar_url":"https://example.com/avatar.jpg"}', 1);

-- 查询示例
-- 通过第三方登录信息查找系统用户ID
-- SELECT user_id FROM third_party_logins WHERE provider = 'apple' AND provider_user_id = '001234.abc123def456.1234' AND deleted = 0;

-- 通过邮箱查找第三方登录记录
-- SELECT * FROM third_party_logins WHERE provider_email = 'user@example.com' AND deleted = 0;

-- 查询用户的所有第三方登录方式
-- SELECT * FROM third_party_logins WHERE user_id = 1000001 AND deleted = 0 ORDER BY create_at DESC;

-- 查询第三方登录统计
-- SELECT provider, COUNT(*) as count FROM third_party_logins WHERE deleted = 0 GROUP BY provider;

-- 查询特定提供商的用户列表
-- SELECT provider_user_name, provider_email FROM third_party_logins WHERE provider = 'apple' AND deleted = 0;

-- 查询用户的基本第三方信息（不包含敏感信息）
-- SELECT provider, provider_user_name, provider_email, create_at FROM third_party_logins WHERE user_id = 1000001 AND deleted = 0;

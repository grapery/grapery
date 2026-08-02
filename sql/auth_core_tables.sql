-- 认证系统核心表创建SQL（简化版）
-- 创建时间: 2024-01-01
-- 描述: 用户认证系统的核心表结构

-- 设置字符集和排序规则
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- =============================================
-- 1. 用户基础信息表 (users)
-- =============================================
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '用户ID',
  `name` varchar(100) NOT NULL DEFAULT '' COMMENT '用户名',
  `email` varchar(255) DEFAULT NULL COMMENT '邮箱',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `gender` tinyint NOT NULL DEFAULT '0' COMMENT '性别：0-未知，1-男，2-女',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '用户状态：1-正常，2-禁用',
  `avatar` varchar(500) DEFAULT NULL COMMENT '头像URL',
  `short_desc` text COMMENT '简短描述',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除',
  PRIMARY KEY (`id`),
  KEY `idx_email` (`email`),
  KEY `idx_phone` (`phone`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户基础信息表';

-- =============================================
-- 2. 用户认证信息表 (user_auth)
-- =============================================
CREATE TABLE `user_auth` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '认证记录ID',
  `uid` bigint NOT NULL COMMENT '用户ID',
  `email` varchar(255) DEFAULT NULL COMMENT '邮箱',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `password` varchar(255) NOT NULL COMMENT '密码（bcrypt加密）',
  `token` varchar(500) DEFAULT NULL COMMENT '认证令牌',
  `is_valid` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否有效',
  `auth_type` tinyint NOT NULL DEFAULT '1' COMMENT '认证类型：1-邮箱，2-手机号',
  `expired` bigint NOT NULL DEFAULT '0' COMMENT '过期时间戳',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uid` (`uid`),
  UNIQUE KEY `uk_email` (`email`),
  UNIQUE KEY `uk_phone` (`phone`),
  KEY `idx_is_valid` (`is_valid`),
  KEY `idx_expired` (`expired`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户认证信息表';

-- =============================================
-- 3. 用户档案表 (user_profiles)
-- =============================================
CREATE TABLE `user_profiles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '档案ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `background` text COMMENT '背景信息',
  `num_group` int NOT NULL DEFAULT '0' COMMENT '加入群组数',
  `used_tokens` int NOT NULL DEFAULT '0' COMMENT '已用token数量',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态',
  `created_group_num` int NOT NULL DEFAULT '0' COMMENT '创建群组数',
  `created_story_num` int NOT NULL DEFAULT '0' COMMENT '创建故事数',
  `created_role_num` int NOT NULL DEFAULT '0' COMMENT '创建角色数',
  `followers_num` int NOT NULL DEFAULT '0' COMMENT '粉丝数',
  `following_num` int NOT NULL DEFAULT '0' COMMENT '关注数',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户档案表';

-- =============================================
-- 4. 用户会话表 (user_sessions)
-- =============================================
CREATE TABLE `user_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '会话ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `session_id` varchar(255) NOT NULL COMMENT '会话标识',
  `token` varchar(500) NOT NULL COMMENT '认证令牌',
  `ip_address` varchar(45) DEFAULT NULL COMMENT 'IP地址',
  `user_agent` varchar(500) DEFAULT NULL COMMENT '用户代理',
  `login_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
  `last_active_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后活跃时间',
  `expire_time` timestamp NOT NULL COMMENT '过期时间',
  `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否活跃',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session_id` (`session_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_token` (`token`),
  KEY `idx_expire_time` (`expire_time`),
  KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户会话表';

-- =============================================
-- 5. 密码重置表 (password_resets)
-- =============================================
CREATE TABLE `password_resets` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '重置记录ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `email` varchar(255) NOT NULL COMMENT '邮箱',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `reset_token` varchar(255) NOT NULL COMMENT '重置令牌',
  `old_password` varchar(255) DEFAULT NULL COMMENT '旧密码',
  `new_password` varchar(255) DEFAULT NULL COMMENT '新密码',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '状态：0-待处理，1-成功，2-失败，3-已过期',
  `expire_time` timestamp NOT NULL COMMENT '过期时间',
  `ip_address` varchar(45) DEFAULT NULL COMMENT 'IP地址',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_reset_token` (`reset_token`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_email` (`email`),
  KEY `idx_status` (`status`),
  KEY `idx_expire_time` (`expire_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='密码重置表';

-- =============================================
-- 复合索引优化
-- =============================================
ALTER TABLE `users` ADD INDEX `idx_email_status` (`email`, `status`);
ALTER TABLE `users` ADD INDEX `idx_phone_status` (`phone`, `status`);
ALTER TABLE `user_auth` ADD INDEX `idx_email_is_valid` (`email`, `is_valid`);
ALTER TABLE `user_auth` ADD INDEX `idx_phone_is_valid` (`phone`, `is_valid`);
ALTER TABLE `user_sessions` ADD INDEX `idx_user_id_is_active` (`user_id`, `is_active`);

-- =============================================
-- 触发器
-- =============================================

-- 用户创建时自动创建档案
DELIMITER $$
CREATE TRIGGER `tr_users_after_insert` 
AFTER INSERT ON `users` 
FOR EACH ROW
BEGIN
    INSERT INTO `user_profiles` (user_id, status, create_at, update_at)
    VALUES (NEW.id, 1, NOW(), NOW());
END$$
DELIMITER ;

-- =============================================
-- 完成
-- =============================================

SET FOREIGN_KEY_CHECKS = 1;

-- 显示创建的表信息
SHOW TABLES LIKE '%user%';
SHOW TABLES LIKE '%auth%';

-- 显示表结构
DESCRIBE users;
DESCRIBE user_auth;
DESCRIBE user_profiles;
DESCRIBE user_sessions;
DESCRIBE password_resets; 
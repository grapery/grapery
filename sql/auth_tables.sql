-- 认证系统数据库表创建SQL
-- 创建时间: 2024-01-01
-- 描述: 用户认证、用户信息、用户档案等相关表结构

-- 设置字符集和排序规则
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- =============================================
-- 1. 用户基础信息表 (users)
-- =============================================
CREATE TABLE `users` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '用户ID，自增从1000000开始',
  `name` varchar(100) NOT NULL DEFAULT '' COMMENT '用户名',
  `email` varchar(255) DEFAULT NULL COMMENT '邮箱',
  `phone` varchar(20) DEFAULT NULL COMMENT '手机号',
  `gender` tinyint NOT NULL DEFAULT '0' COMMENT '性别：0-未知，1-男，2-女',
  `bio_id` varchar(100) DEFAULT NULL COMMENT '简介ID',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '用户状态：1-正常，2-禁用，3-删除',
  `location` varchar(255) DEFAULT NULL COMMENT '位置信息',
  `avatar` varchar(500) DEFAULT NULL COMMENT '头像URL',
  `short_desc` text COMMENT '简短描述',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  KEY `idx_name` (`name`),
  KEY `idx_email` (`email`),
  KEY `idx_phone` (`phone`),
  KEY `idx_status` (`status`),
  KEY `idx_create_at` (`create_at`)
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
  `salt` varchar(100) DEFAULT NULL COMMENT '密码盐值',
  `is_valid` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否有效：0-无效，1-有效',
  `auth_type` tinyint NOT NULL DEFAULT '1' COMMENT '认证类型：1-邮箱，2-手机号，3-第三方',
  `expired` bigint NOT NULL DEFAULT '0' COMMENT '过期时间戳',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_uid` (`uid`),
  UNIQUE KEY `uk_email` (`email`),
  UNIQUE KEY `uk_phone` (`phone`),
  KEY `idx_auth_type` (`auth_type`),
  KEY `idx_is_valid` (`is_valid`),
  KEY `idx_expired` (`expired`),
  KEY `idx_create_at` (`create_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户认证信息表';

-- =============================================
-- 3. 用户档案表 (user_profiles)
-- =============================================
CREATE TABLE `user_profiles` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '档案ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `background` text COMMENT '背景信息',
  `num_group` int NOT NULL DEFAULT '0' COMMENT '加入群组数',
  `default_group_id` bigint NOT NULL DEFAULT '0' COMMENT '默认群组ID',
  `min_same_group` int NOT NULL DEFAULT '0' COMMENT '最小同群组数',
  `limit` int NOT NULL DEFAULT '0' COMMENT '限制数量',
  `used_tokens` int NOT NULL DEFAULT '0' COMMENT '已用token数量',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1-正常，2-禁用',
  `created_group_num` int NOT NULL DEFAULT '0' COMMENT '创建群组数',
  `created_story_num` int NOT NULL DEFAULT '0' COMMENT '创建故事数',
  `created_role_num` int NOT NULL DEFAULT '0' COMMENT '创建角色数',
  `created_board_num` int NOT NULL DEFAULT '0' COMMENT '创建故事板数',
  `created_gen_num` int NOT NULL DEFAULT '0' COMMENT '创建生成数',
  `watching_story_num` int NOT NULL DEFAULT '0' COMMENT '关注故事数',
  `watching_group_num` int NOT NULL DEFAULT '0' COMMENT '关注群组数',
  `watching_story_role_num` int NOT NULL DEFAULT '0' COMMENT '关注角色数',
  `contribut_story_num` int NOT NULL DEFAULT '0' COMMENT '贡献故事数',
  `contribut_role_num` int NOT NULL DEFAULT '0' COMMENT '贡献角色数',
  `liked_story_num` int NOT NULL DEFAULT '0' COMMENT '点赞故事数',
  `liked_role_num` int NOT NULL DEFAULT '0' COMMENT '点赞角色数',
  `followers_num` int NOT NULL DEFAULT '0' COMMENT '粉丝数',
  `following_num` int NOT NULL DEFAULT '0' COMMENT '关注数',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_id` (`user_id`),
  KEY `idx_status` (`status`),
  KEY `idx_create_at` (`create_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户档案表';

-- =============================================
-- 4. 用户会话表 (user_sessions)
-- =============================================
CREATE TABLE `user_sessions` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '会话ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `session_id` varchar(255) NOT NULL COMMENT '会话标识',
  `token` varchar(500) NOT NULL COMMENT '认证令牌',
  `device_info` varchar(500) DEFAULT NULL COMMENT '设备信息',
  `ip_address` varchar(45) DEFAULT NULL COMMENT 'IP地址',
  `user_agent` varchar(500) DEFAULT NULL COMMENT '用户代理',
  `login_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '登录时间',
  `last_active_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '最后活跃时间',
  `expire_time` timestamp NOT NULL COMMENT '过期时间',
  `is_active` tinyint(1) NOT NULL DEFAULT '1' COMMENT '是否活跃：0-不活跃，1-活跃',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_session_id` (`session_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_token` (`token`),
  KEY `idx_login_time` (`login_time`),
  KEY `idx_expire_time` (`expire_time`),
  KEY `idx_is_active` (`is_active`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户会话表';

-- =============================================
-- 5. 用户活动记录表 (user_activities)
-- =============================================
CREATE TABLE `user_activities` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '活动ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `activity_type` varchar(50) NOT NULL COMMENT '活动类型：login, logout, register, reset_password',
  `activity_data` json DEFAULT NULL COMMENT '活动数据',
  `ip_address` varchar(45) DEFAULT NULL COMMENT 'IP地址',
  `user_agent` varchar(500) DEFAULT NULL COMMENT '用户代理',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1-成功，2-失败',
  `error_message` text COMMENT '错误信息',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_activity_type` (`activity_type`),
  KEY `idx_status` (`status`),
  KEY `idx_create_at` (`create_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户活动记录表';

-- =============================================
-- 6. 密码重置表 (password_resets)
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
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_reset_token` (`reset_token`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_email` (`email`),
  KEY `idx_phone` (`phone`),
  KEY `idx_status` (`status`),
  KEY `idx_expire_time` (`expire_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='密码重置表';

-- =============================================
-- 7. 邮箱验证表 (email_verifications)
-- =============================================
CREATE TABLE `email_verifications` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '验证记录ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `email` varchar(255) NOT NULL COMMENT '邮箱',
  `verification_token` varchar(255) NOT NULL COMMENT '验证令牌',
  `status` tinyint NOT NULL DEFAULT '0' COMMENT '状态：0-待验证，1-已验证，2-已过期',
  `expire_time` timestamp NOT NULL COMMENT '过期时间',
  `verified_at` timestamp NULL DEFAULT NULL COMMENT '验证时间',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_verification_token` (`verification_token`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_email` (`email`),
  KEY `idx_status` (`status`),
  KEY `idx_expire_time` (`expire_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='邮箱验证表';

-- =============================================
-- 8. 第三方登录表 (third_party_logins)
-- =============================================
CREATE TABLE `third_party_logins` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '第三方登录记录ID',
  `user_id` bigint NOT NULL COMMENT '用户ID',
  `provider` varchar(50) NOT NULL COMMENT '提供商：google, facebook, wechat, alipay',
  `provider_user_id` varchar(255) NOT NULL COMMENT '第三方用户ID',
  `provider_user_info` json DEFAULT NULL COMMENT '第三方用户信息',
  `access_token` varchar(500) DEFAULT NULL COMMENT '访问令牌',
  `refresh_token` varchar(500) DEFAULT NULL COMMENT '刷新令牌',
  `token_expire_time` timestamp NULL DEFAULT NULL COMMENT '令牌过期时间',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1-正常，2-禁用',
  `create_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted` tinyint(1) NOT NULL DEFAULT '0' COMMENT '是否删除：0-未删除，1-已删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_provider_user_id` (`provider`, `provider_user_id`),
  KEY `idx_user_id` (`user_id`),
  KEY `idx_provider` (`provider`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='第三方登录表';

-- =============================================
-- 索引优化
-- =============================================

-- 为常用查询添加复合索引
ALTER TABLE `users` ADD INDEX `idx_email_status` (`email`, `status`);
ALTER TABLE `users` ADD INDEX `idx_phone_status` (`phone`, `status`);
ALTER TABLE `user_auth` ADD INDEX `idx_email_is_valid` (`email`, `is_valid`);
ALTER TABLE `user_auth` ADD INDEX `idx_phone_is_valid` (`phone`, `is_valid`);
ALTER TABLE `user_sessions` ADD INDEX `idx_user_id_is_active` (`user_id`, `is_active`);
ALTER TABLE `user_activities` ADD INDEX `idx_user_id_activity_type` (`user_id`, `activity_type`);

-- =============================================
-- 外键约束（可选，根据业务需求决定是否启用）
-- =============================================

-- 启用外键约束
-- SET FOREIGN_KEY_CHECKS = 1;

-- 添加外键约束（如果需要）
-- ALTER TABLE `user_auth` ADD CONSTRAINT `fk_user_auth_uid` FOREIGN KEY (`uid`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `user_profiles` ADD CONSTRAINT `fk_user_profiles_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `user_sessions` ADD CONSTRAINT `fk_user_sessions_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `user_activities` ADD CONSTRAINT `fk_user_activities_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `password_resets` ADD CONSTRAINT `fk_password_resets_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `email_verifications` ADD CONSTRAINT `fk_email_verifications_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;
-- ALTER TABLE `third_party_logins` ADD CONSTRAINT `fk_third_party_logins_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE;

-- =============================================
-- 初始数据（可选）
-- =============================================

-- 插入默认管理员用户（如果需要）
-- INSERT INTO `users` (`id`, `name`, `email`, `status`, `create_at`, `update_at`) 
-- VALUES (1000000, 'admin', 'admin@grapery.com', 1, NOW(), NOW());

-- =============================================
-- 视图创建（可选）
-- =============================================

-- 创建用户完整信息视图
CREATE OR REPLACE VIEW `v_user_complete_info` AS
SELECT 
    u.id,
    u.name,
    u.email,
    u.phone,
    u.gender,
    u.status as user_status,
    u.avatar,
    u.short_desc,
    ua.auth_type,
    ua.is_valid as auth_valid,
    up.num_group,
    up.followers_num,
    up.following_num,
    up.created_story_num,
    up.created_role_num,
    u.create_at,
    u.update_at
FROM `users` u
LEFT JOIN `user_auth` ua ON u.id = ua.uid
LEFT JOIN `user_profiles` up ON u.id = up.user_id
WHERE u.deleted = 0 AND ua.deleted = 0;

-- 创建活跃用户视图
CREATE OR REPLACE VIEW `v_active_users` AS
SELECT 
    u.id,
    u.name,
    u.email,
    u.avatar,
    up.followers_num,
    up.created_story_num,
    u.create_at
FROM `users` u
LEFT JOIN `user_profiles` up ON u.id = up.user_id
WHERE u.deleted = 0 AND u.status = 1
ORDER BY up.followers_num DESC, up.created_story_num DESC;

-- =============================================
-- 存储过程（可选）
-- =============================================

DELIMITER $$

-- 用户注册存储过程
CREATE PROCEDURE `sp_register_user`(
    IN p_name VARCHAR(100),
    IN p_email VARCHAR(255),
    IN p_phone VARCHAR(20),
    IN p_password VARCHAR(255),
    IN p_auth_type TINYINT,
    OUT p_user_id BIGINT,
    OUT p_result_code INT,
    OUT p_result_message VARCHAR(255)
)
BEGIN
    DECLARE v_user_id BIGINT DEFAULT 0;
    DECLARE v_auth_id BIGINT DEFAULT 0;
    DECLARE v_profile_id BIGINT DEFAULT 0;
    DECLARE EXIT HANDLER FOR SQLEXCEPTION
    BEGIN
        ROLLBACK;
        SET p_result_code = -1;
        SET p_result_message = '注册失败，请稍后重试';
    END;
    
    START TRANSACTION;
    
    -- 检查邮箱或手机号是否已存在
    IF EXISTS(SELECT 1 FROM user_auth WHERE (email = p_email OR phone = p_phone) AND deleted = 0) THEN
        SET p_result_code = 1003; -- USER_ALREADY_EXISTS
        SET p_result_message = '用户已存在';
        ROLLBACK;
    ELSE
        -- 创建用户
        INSERT INTO users (name, email, phone, status, create_at, update_at)
        VALUES (p_name, p_email, p_phone, 1, NOW(), NOW());
        SET v_user_id = LAST_INSERT_ID();
        
        -- 创建认证信息
        INSERT INTO user_auth (uid, email, phone, password, auth_type, expired, create_at, update_at)
        VALUES (v_user_id, p_email, p_phone, p_password, p_auth_type, UNIX_TIMESTAMP() + 3600*72, NOW(), NOW());
        SET v_auth_id = LAST_INSERT_ID();
        
        -- 创建用户档案
        INSERT INTO user_profiles (user_id, status, create_at, update_at)
        VALUES (v_user_id, 1, NOW(), NOW());
        SET v_profile_id = LAST_INSERT_ID();
        
        SET p_user_id = v_user_id;
        SET p_result_code = 0; -- OK
        SET p_result_message = '注册成功';
        
        COMMIT;
    END IF;
END$$

-- 用户登录验证存储过程
CREATE PROCEDURE `sp_verify_login`(
    IN p_account VARCHAR(255),
    IN p_password VARCHAR(255),
    OUT p_user_id BIGINT,
    OUT p_result_code INT,
    OUT p_result_message VARCHAR(255)
)
BEGIN
    DECLARE v_user_id BIGINT DEFAULT 0;
    DECLARE v_stored_password VARCHAR(255);
    DECLARE v_is_valid TINYINT DEFAULT 0;
    
    -- 查找用户认证信息
    SELECT uid, password, is_valid INTO v_user_id, v_stored_password, v_is_valid
    FROM user_auth 
    WHERE (email = p_account OR phone = p_account) AND deleted = 0
    LIMIT 1;
    
    IF v_user_id = 0 THEN
        SET p_result_code = 1001; -- ACCOUNT_NOT_FOUND
        SET p_result_message = '账号不存在';
    ELSEIF v_is_valid = 0 THEN
        SET p_result_code = 1004; -- ACCOUNT_EXPIRED
        SET p_result_message = '账号已过期';
    ELSEIF v_stored_password != p_password THEN
        SET p_result_code = 1002; -- WRONG_PASSWORD
        SET p_result_message = '密码错误';
    ELSE
        SET p_result_code = 0; -- OK
        SET p_result_message = '登录成功';
    END IF;
    
    SET p_user_id = v_user_id;
END$$

DELIMITER ;

-- =============================================
-- 触发器（可选）
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

-- 用户删除时软删除相关记录
DELIMITER $$
CREATE TRIGGER `tr_users_after_update` 
AFTER UPDATE ON `users` 
FOR EACH ROW
BEGIN
    IF NEW.deleted = 1 AND OLD.deleted = 0 THEN
        UPDATE `user_auth` SET deleted = 1 WHERE uid = NEW.id;
        UPDATE `user_profiles` SET deleted = 1 WHERE user_id = NEW.id;
        UPDATE `user_sessions` SET deleted = 1 WHERE user_id = NEW.id;
    END IF;
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
DESCRIBE user_activities;
DESCRIBE password_resets;
DESCRIBE email_verifications;
DESCRIBE third_party_logins; 
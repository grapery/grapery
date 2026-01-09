-- Token 用量日志表
-- 用于记录每条 token 使用记录的详细日志，关联到具体的业务实体

CREATE TABLE IF NOT EXISTS `token_usage_log` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT NOT NULL COMMENT '用户ID',
    `entity_type` VARCHAR(50) NOT NULL COMMENT '业务实体类型: story, storyboard, character, poster等',
    `entity_id` VARCHAR(36) NOT NULL COMMENT '业务实体ID',
    `operation_type` VARCHAR(50) NOT NULL COMMENT '操作类型: create, update, generate_image, generate_video等',
    `usage_type` INT NOT NULL COMMENT 'Token使用类型: 1=chat, 2=image_gen, 3=video_gen等',
    
    -- Token 使用详情
    `input_tokens` INT NOT NULL DEFAULT 0 COMMENT '输入Token数',
    `output_tokens` INT NOT NULL DEFAULT 0 COMMENT '输出Token数',
    `total_tokens` INT NOT NULL DEFAULT 0 COMMENT '总Token数',
    
    -- 模型和功能信息
    `model_name` VARCHAR(100) DEFAULT NULL COMMENT '使用的模型名称',
    `provider` VARCHAR(50) DEFAULT NULL COMMENT '提供商: gemini, hailuo等',
    `feature_name` VARCHAR(100) DEFAULT NULL COMMENT '功能名称',
    
    -- 关联信息
    `task_id` VARCHAR(36) DEFAULT NULL COMMENT '关联的AI任务ID',
    `story_id` VARCHAR(36) DEFAULT NULL COMMENT '关联的故事ID（如果适用）',
    
    -- 成本和计费
    `cost_amount` DECIMAL(10,4) DEFAULT 0 COMMENT '成本金额',
    `currency` VARCHAR(10) DEFAULT 'USD' COMMENT '货币类型',
    `is_billed` BOOLEAN DEFAULT FALSE COMMENT '是否已计费',
    `billing_id` VARCHAR(36) DEFAULT NULL COMMENT '计费记录ID',
    
    -- 元数据
    `metadata` JSON DEFAULT NULL COMMENT '扩展元数据（JSON格式）',
    
    -- 时间戳
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `deleted_at` TIMESTAMP NULL DEFAULT NULL COMMENT '删除时间',
    
    PRIMARY KEY (`id`),
    INDEX `idx_user_id` (`user_id`),
    INDEX `idx_entity` (`entity_type`, `entity_id`),
    INDEX `idx_created_at` (`created_at`),
    INDEX `idx_user_entity` (`user_id`, `entity_type`, `entity_id`),
    INDEX `idx_task_id` (`task_id`),
    INDEX `idx_story_id` (`story_id`),
    INDEX `idx_is_billed` (`is_billed`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Token用量日志表';

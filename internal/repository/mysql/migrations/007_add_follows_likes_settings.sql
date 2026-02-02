-- Migration: Add follows, likes tables and update user_settings
-- Date: 2026-02-01
-- Description: Create polymorphic follows/likes tables and add new fields to user_settings, stories, and characters

-- =============================================
-- follows 表: 关注关系（多态关联）
-- =============================================
CREATE TABLE IF NOT EXISTS follows (
    id VARCHAR(36) PRIMARY KEY,
    follower_id VARCHAR(36) NOT NULL COMMENT '关注者用户ID',
    followable_type VARCHAR(50) NOT NULL COMMENT '被关注对象类型: story, user, group, character',
    followable_id VARCHAR(36) NOT NULL COMMENT '被关注对象ID',
    notifications_enabled BOOLEAN DEFAULT TRUE COMMENT '是否接收通知',
    created_at BIGINT NOT NULL COMMENT '创建时间戳',
    
    -- 唯一约束：一个用户对同一对象只能关注一次
    UNIQUE KEY uk_follow_unique (follower_id, followable_type, followable_id),
    
    -- 索引
    INDEX idx_follower_id (follower_id),
    INDEX idx_followable (followable_type, followable_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='关注关系表';

-- =============================================
-- likes 表: 点赞关系（多态关联）
-- =============================================
CREATE TABLE IF NOT EXISTS likes (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL COMMENT '点赞用户ID',
    likeable_type VARCHAR(50) NOT NULL COMMENT '被点赞对象类型: story, character, storyboard_node, fragment, character_poster',
    likeable_id VARCHAR(36) NOT NULL COMMENT '被点赞对象ID',
    created_at BIGINT NOT NULL COMMENT '创建时间戳',
    
    -- 唯一约束：一个用户对同一对象只能点赞一次
    UNIQUE KEY uk_like_unique (user_id, likeable_type, likeable_id),
    
    -- 索引
    INDEX idx_user_id (user_id),
    INDEX idx_likeable (likeable_type, likeable_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='点赞关系表';

-- =============================================
-- user_settings 表: 更新字段
-- =============================================

-- 添加字体大小字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS font_size VARCHAR(20) DEFAULT 'medium' COMMENT '字体大小: small, medium, large';

-- 添加省流量模式字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS data_saver BOOLEAN DEFAULT FALSE COMMENT '省流量模式';

-- 添加故事默认可见性字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS default_story_visibility VARCHAR(50) DEFAULT 'public' COMMENT '故事默认可见性: public, unlisted, private';

-- 添加碎片默认可见性字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS default_fragment_visibility VARCHAR(50) DEFAULT 'public' COMMENT '碎片默认可见性: public, followers_only, private';

-- 添加谁可以关注我字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS allow_follow_from VARCHAR(50) DEFAULT 'everyone' COMMENT '谁可以关注我: everyone, followers_only, followers_of_followers, no_one';

-- 添加谁可以评论我字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS allow_comments_from VARCHAR(50) DEFAULT 'everyone' COMMENT '谁可以评论我: everyone, followers_only, no_one';

-- 添加谁可以私信我字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS allow_messages_from VARCHAR(50) DEFAULT 'followers_only' COMMENT '谁可以私信我: everyone, followers_only, no_one';

-- 添加显示已读回执字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS show_read_receipts BOOLEAN DEFAULT TRUE COMMENT '显示已读回执';

-- 添加启用AI功能字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS ai_enabled BOOLEAN DEFAULT TRUE COMMENT '启用AI功能';

-- 添加允许使用数据改进AI字段
ALTER TABLE user_settings 
    ADD COLUMN IF NOT EXISTS ai_data_sharing BOOLEAN DEFAULT TRUE COMMENT '允许使用数据改进AI';

-- =============================================
-- stories 表: 添加 ai_enabled 字段
-- =============================================
ALTER TABLE stories 
    ADD COLUMN IF NOT EXISTS ai_enabled BOOLEAN DEFAULT TRUE COMMENT '是否允许AI辅助';

-- =============================================
-- characters 表: 添加 poster_creation_permission 字段
-- =============================================
ALTER TABLE characters 
    ADD COLUMN IF NOT EXISTS poster_creation_permission VARCHAR(50) DEFAULT 'creator_only' COMMENT '海报创建权限: creator_only, group_members, anyone';

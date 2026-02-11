-- Migration: Add Story Universe Fields for Parallel Universe System
-- Date: 2026-02-10
-- Description: Add fate_snapshot and context_snapshot for character state tracking across parallel universes
--              Add character cameo support for cross-story character references

-- =============================================
-- storyboards 表: 添加命运快照字段
-- =============================================
-- fate_snapshot 存储分叉时刻所有角色的状态快照（JSON格式）
-- 格式: {"character_id": {"name": "...", "personality": "...", "relationships": {...}, ...}}
ALTER TABLE storyboards
    ADD COLUMN IF NOT EXISTS fate_snapshot JSON COMMENT '分叉时刻所有角色的状态快照，用于平行宇宙角色状态恢复',
    ADD COLUMN IF NOT EXISTS fate_snapshot_hash VARCHAR(64) COMMENT '命运快照哈希值，用于检测状态变化';

-- 索引优化：为 fate_snapshot_hash 添加索引
ALTER TABLE storyboards
    ADD INDEX IF NOT EXISTS idx_fate_snapshot_hash (fate_snapshot_hash);

-- =============================================
-- storyboard_scenes 表: 添加上下文快照字段
-- =============================================
-- context_snapshot 存储该场景结束后的角色状态增量（JSON格式）
-- 格式: {"character_id": {"health": 100, "mood": "happy", "location": "castle", ...}}
ALTER TABLE storyboard_scenes
    ADD COLUMN IF NOT EXISTS context_snapshot JSON COMMENT '该场景结束后的角色状态增量，用于状态追溯';

-- =============================================
-- characters 表: 添加客串支持字段
-- =============================================
-- origin_story_id 标识角色来源故事，用于客串角色追踪
-- is_cameo 标识是否为客串角色
ALTER TABLE characters
    ADD COLUMN IF NOT EXISTS origin_story_id VARCHAR(36) COMMENT '原始故事ID，用于客串角色',
    ADD COLUMN IF NOT EXISTS is_cameo BOOLEAN DEFAULT FALSE COMMENT '是否为客串角色';

-- 添加外键约束（可选，根据实际数据完整性要求）
-- ALTER TABLE characters
--     ADD CONSTRAINT fk_character_origin_story
--     FOREIGN KEY (origin_story_id) REFERENCES stories(id) ON DELETE SET NULL;

-- 索引优化：为客串查询添加索引
ALTER TABLE characters
    ADD INDEX IF NOT EXISTS idx_origin_story_id (origin_story_id),
    ADD INDEX IF NOT EXISTS idx_is_cameo (is_cameo);

-- =============================================
-- character_cameos 表: 角色客串关系表
-- =============================================
-- 用于管理角色跨故事客串关系
CREATE TABLE IF NOT EXISTS character_cameos (
    id VARCHAR(36) PRIMARY KEY COMMENT '客串关系ID',
    character_id VARCHAR(36) NOT NULL COMMENT '客串角色ID',
    target_story_id VARCHAR(36) NOT NULL COMMENT '目标故事ID',
    cameo_role VARCHAR(100) COMMENT '客串角色定位（如：主角、配角、NPC等）',
    adaptation_notes TEXT COMMENT '角色适配说明，记录如何在目标故事中调整角色',
    created_at BIGINT NOT NULL COMMENT '创建时间戳',
    updated_at BIGINT COMMENT '更新时间戳',

    -- 唯一约束：同一角色对同一故事只能有一条客串记录
    UNIQUE KEY uk_character_target (character_id, target_story_id),

    -- 外键约束
    CONSTRAINT fk_cameo_character FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE,
    CONSTRAINT fk_cameo_target_story FOREIGN KEY (target_story_id) REFERENCES stories(id) ON DELETE CASCADE,

    -- 索引
    INDEX idx_character_id (character_id),
    INDEX idx_target_story_id (target_story_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='角色客串关系表';

-- =============================================
-- 数据迁移说明
-- =============================================
-- 1. 对于现有的 storyboards，fate_snapshot 默认为 NULL
--    首次分叉时会自动生成初始快照
--
-- 2. 对于现有的 storyboard_scenes，context_snapshot 默认为 NULL
--    新生成的场景会自动创建状态增量
--
-- 3. 对于现有的 characters，is_cameo 默认为 FALSE
--    origin_story_id 默认为 NULL（表示原生角色）
--
-- 4. character_cameos 表为空，需要通过 API 添加客串关系

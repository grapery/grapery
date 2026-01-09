-- Add portrait-related fields to characters table
-- Migration: Add portrait, needs_portrait, reference_image, and portrait_generation_status fields

ALTER TABLE characters 
ADD COLUMN portrait VARCHAR(500) DEFAULT NULL COMMENT '完整角色形象图URL（AI生成）',
ADD COLUMN needs_portrait BOOLEAN DEFAULT FALSE COMMENT '是否需要生成形象',
ADD COLUMN reference_image VARCHAR(500) DEFAULT NULL COMMENT '参考图URL',
ADD COLUMN portrait_generation_status VARCHAR(20) DEFAULT 'none' COMMENT '形象生成状态: none/pending/generating/generated/failed';

-- Add index for portrait_generation_status to support queries filtering by status
CREATE INDEX idx_characters_portrait_status ON characters(portrait_generation_status);

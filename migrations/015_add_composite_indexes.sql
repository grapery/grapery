-- Migration: Add Composite Indexes for Query Optimization
-- Version: 015
-- Date: 2026-02-11
-- Author: Database Migration Team
-- Description: Add composite indexes to optimize common query patterns
-- Priority: P3 - Low (Performance optimization)
-- Breaking Change: No
-- Downtime Required: No (Can be executed during live operations)

-- =====================================================
-- PRE-MIGRATION ANALYSIS
-- =====================================================

-- Check current index usage
-- Run this before migration to identify which indexes will be most beneficial
-- SELECT * FROM sys.schema_unused_indexes WHERE object_schema = DATABASE();

-- =====================================================
-- MIGRATION UP
-- =====================================================

BEGIN;

-- Index 1: comments table - Optimize queries filtering by target_type and target_id
-- Pattern: SELECT * FROM comments WHERE target_type = ? AND target_id = ?
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'comments'
    AND INDEX_NAME = 'idx_target'
);

SET @create_index_sql = IF(@index_exists = 0,
    'CREATE INDEX idx_target ON comments(target_type, target_id)',
    'SELECT ''Index idx_target already exists, skipping'' AS message'
);

PREPARE stmt FROM @create_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Index 2: storyboards table - Optimize tree structure queries
-- Pattern: SELECT * FROM storyboards WHERE story_id = ? AND parent_id = ?
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'storyboards'
    AND INDEX_NAME = 'idx_tree'
);

SET @create_index_sql = IF(@index_exists = 0,
    'CREATE INDEX idx_tree ON storyboards(story_id, parent_id)',
    'SELECT ''Index idx_tree already exists, skipping'' AS message'
);

PREPARE stmt FROM @create_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Index 3: fragments table - Optimize queries by user and visibility
-- Pattern: SELECT * FROM fragments WHERE user_id = ? AND visibility = ?
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'fragments'
    AND INDEX_NAME = 'idx_user_visibility'
);

SET @create_index_sql = IF(@index_exists = 0,
    'CREATE INDEX idx_user_visibility ON fragments(user_id, visibility)',
    'SELECT ''Index idx_user_visibility already exists, skipping'' AS message'
);

PREPARE stmt FROM @create_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Index 4: likes table - Optimize polymorphic queries
-- Pattern: SELECT * FROM likes WHERE likeable_type = ? AND likeable_id = ?
-- Note: idx_likeable may already exist from migration 007
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'likes'
    AND INDEX_NAME = 'idx_likeable'
);

SET @create_index_sql = IF(@index_exists = 0,
    'CREATE INDEX idx_likeable ON likes(likeable_type, likeable_id)',
    'SELECT ''Index idx_likeable already exists, skipping'' AS message'
);

PREPARE stmt FROM @create_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Index 5: follows table - Optimize polymorphic queries
-- Pattern: SELECT * FROM follows WHERE followable_type = ? AND followable_id = ?
-- Note: idx_followable may already exist from migration 007
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'follows'
    AND INDEX_NAME = 'idx_followable'
);

SET @create_index_sql = IF(@index_exists = 0,
    'CREATE INDEX idx_followable ON follows(followable_type, followable_id)',
    'SELECT ''Index idx_followable already exists, skipping'' AS message'
);

PREPARE stmt FROM @create_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Index 6: notifications table - Optimize user notification queries
-- Pattern: SELECT * FROM notifications WHERE user_id = ? AND read = ?
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'notifications'
    AND INDEX_NAME = 'idx_user_read'
);

SET @create_index_sql = IF(@index_exists = 0,
    'CREATE INDEX idx_user_read ON notifications(user_id, read)',
    'SELECT ''Index idx_user_read already exists, skipping'' AS message'
);

PREPARE stmt FROM @create_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

COMMIT;

-- =====================================================
-- VERIFICATION QUERIES
-- =====================================================

-- Verify all new indexes were created
SELECT
    TABLE_NAME,
    INDEX_NAME,
    GROUP_CONCAT(COLUMN_NAME ORDER BY SEQ_IN_INDEX) as columns
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = DATABASE()
AND INDEX_NAME IN (
    'idx_target',
    'idx_tree',
    'idx_user_visibility',
    'idx_likeable',
    'idx_followable',
    'idx_user_read'
)
GROUP BY TABLE_NAME, INDEX_NAME
ORDER BY TABLE_NAME, INDEX_NAME;

-- =====================================================
-- MIGRATION DOWN (ROLLBACK)
-- =====================================================

/*
BEGIN;

-- Rollback: Drop the new indexes
DROP INDEX IF EXISTS idx_target ON comments;
DROP INDEX IF EXISTS idx_tree ON storyboards;
DROP INDEX IF EXISTS idx_user_visibility ON fragments;
DROP INDEX IF EXISTS idx_likeable ON likes;
DROP INDEX IF EXISTS idx_followable ON follows;
DROP INDEX IF EXISTS idx_user_read ON notifications;

COMMIT;
*/

-- =====================================================
-- POST-MIGRATION MONITORING
-- =====================================================

-- After running this migration, monitor the following:

-- 1. Check query execution plans for queries that should use these indexes
-- EXPLAIN SELECT * FROM comments WHERE target_type = 'story' AND target_id = '...';

-- 2. Monitor index usage over time
-- SELECT * FROM sys.schema_index_statistics WHERE table_schema = DATABASE();

-- 3. Check for unused indexes (can be removed after monitoring period)
-- SELECT * FROM sys.schema_unused_indexes WHERE object_schema = DATABASE();

-- 4. Monitor write performance (indexes add overhead to INSERT/UPDATE/DELETE)
-- SHOW GLOBAL STATUS LIKE 'Handler_write%';

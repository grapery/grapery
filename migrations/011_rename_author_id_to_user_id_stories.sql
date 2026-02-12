-- Migration: Rename author_id to user_id in stories table
-- Version: 011
-- Date: 2026-02-11
-- Author: Database Migration Team
-- Description: Rename author_id column to user_id for consistency across the schema
--              THIS IS A CORE TABLE - HIGH IMPACT MIGRATION
-- Priority: P0 - Critical
-- Breaking Change: Yes
-- Downtime Required: Yes

-- =====================================================
-- PRE-MIGRATION CHECKS
-- =====================================================

-- Check table size (run before migration)
SELECT
    TABLE_NAME,
    ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024), 2) AS 'Size in MB',
    TABLE_ROWS
FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = DATABASE()
AND TABLE_NAME = 'stories';

-- Check for orphaned records
SELECT COUNT(*) as orphaned_count
FROM stories s
LEFT JOIN users u ON s.author_id = u.id
WHERE u.id IS NULL;

-- =====================================================
-- MIGRATION UP
-- =====================================================

BEGIN;

-- Step 1: Optional backup (uncomment if needed)
-- CREATE TABLE stories_backup_20260211 AS SELECT * FROM stories;

-- Step 2: Drop old index if exists
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'stories'
    AND INDEX_NAME = 'idx_author_id'
);

SET @drop_index_sql = IF(@index_exists > 0,
    'DROP INDEX idx_author_id ON stories',
    'SELECT ''Index idx_author_id does not exist, skipping'' AS message'
);

PREPARE stmt FROM @drop_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 3: Drop old foreign key if exists
SET @fk_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'stories'
    AND CONSTRAINT_NAME = 'fk_story_author'
    AND REFERENCED_TABLE_NAME = 'users'
);

SET @drop_fk_sql = IF(@fk_exists > 0,
    'ALTER TABLE stories DROP FOREIGN KEY fk_story_author',
    'SELECT ''Foreign key fk_story_author does not exist, skipping'' AS message'
);

PREPARE stmt FROM @drop_fk_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 4: Rename the column
-- This may take several minutes on large tables
ALTER TABLE stories
    CHANGE COLUMN author_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- Step 5: Create new index
CREATE INDEX idx_user_id ON stories(user_id);

-- Step 6: Add new foreign key constraint
ALTER TABLE stories
    ADD CONSTRAINT fk_story_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

COMMIT;

-- =====================================================
-- VERIFICATION QUERIES
-- =====================================================

-- Verify column rename
SELECT
    COLUMN_NAME,
    DATA_TYPE,
    IS_NULLABLE,
    COLUMN_COMMENT
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
AND TABLE_NAME = 'stories'
AND COLUMN_NAME = 'user_id';

-- Verify foreign key exists
SELECT
    CONSTRAINT_NAME,
    COLUMN_NAME,
    REFERENCED_TABLE_NAME,
    REFERENCED_COLUMN_NAME
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = DATABASE()
AND TABLE_NAME = 'stories'
AND CONSTRAINT_NAME = 'fk_story_user';

-- =====================================================
-- POST-MIGRATION CHECKS FOR RELATED TABLES
-- =====================================================

-- Check related tables that might reference stories.author_id
-- These may need code updates but not database changes

-- Story contributors table (references story_id, not author_id)
SELECT TABLE_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = DATABASE()
AND REFERENCED_TABLE_NAME = 'stories';

-- =====================================================
-- MIGRATION DOWN (ROLLBACK)
-- =====================================================

/*
BEGIN;

-- Rollback Step 1: Drop new foreign key
ALTER TABLE stories DROP FOREIGN KEY fk_story_user;

-- Rollback Step 2: Drop new index
DROP INDEX idx_user_id ON stories;

-- Rollback Step 3: Rename column back
ALTER TABLE stories
    CHANGE COLUMN user_id author_id VARCHAR(36) NOT NULL COMMENT '作者ID';

-- Rollback Step 4: Recreate old index
CREATE INDEX idx_author_id ON stories(author_id);

-- Rollback Step 5: Recreate old foreign key
ALTER TABLE stories
    ADD CONSTRAINT fk_story_author
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;

-- Optional: Restore from backup if needed
-- DROP TABLE stories;
-- RENAME TABLE stories_backup_20260211 TO stories;

COMMIT;
*/

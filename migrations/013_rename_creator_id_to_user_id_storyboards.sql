-- Migration: Rename creator_id to user_id in storyboards table
-- Version: 013
-- Date: 2026-02-11
-- Author: Database Migration Team
-- Description: Rename creator_id column to user_id for consistency across the schema
-- Priority: P1 - High
-- Breaking Change: Yes

-- =====================================================
-- MIGRATION UP
-- =====================================================

BEGIN;

-- Step 1: Drop old index if exists
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'storyboards'
    AND INDEX_NAME = 'idx_creator_id'
);

SET @drop_index_sql = IF(@index_exists > 0,
    'DROP INDEX idx_creator_id ON storyboards',
    'SELECT ''Index idx_creator_id does not exist, skipping'' AS message'
);

PREPARE stmt FROM @drop_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 2: Drop old foreign key if exists
SET @fk_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'storyboards'
    AND CONSTRAINT_NAME = 'fk_storyboard_creator'
    AND REFERENCED_TABLE_NAME = 'users'
);

SET @drop_fk_sql = IF(@fk_exists > 0,
    'ALTER TABLE storyboards DROP FOREIGN KEY fk_storyboard_creator',
    'SELECT ''Foreign key fk_storyboard_creator does not exist, skipping'' AS message'
);

PREPARE stmt FROM @drop_fk_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 3: Rename the column
ALTER TABLE storyboards
    CHANGE COLUMN creator_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- Step 4: Create new index
CREATE INDEX idx_user_id ON storyboards(user_id);

-- Step 5: Add new foreign key constraint
ALTER TABLE storyboards
    ADD CONSTRAINT fk_storyboard_user
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
AND TABLE_NAME = 'storyboards'
AND COLUMN_NAME = 'user_id';

-- =====================================================
-- MIGRATION DOWN (ROLLBACK)
-- =====================================================

/*
BEGIN;

-- Rollback Step 1: Drop new foreign key
ALTER TABLE storyboards DROP FOREIGN KEY fk_storyboard_user;

-- Rollback Step 2: Drop new index
DROP INDEX idx_user_id ON storyboards;

-- Rollback Step 3: Rename column back
ALTER TABLE storyboards
    CHANGE COLUMN user_id creator_id VARCHAR(36) NOT NULL COMMENT '创建者ID';

-- Rollback Step 4: Recreate old index
CREATE INDEX idx_creator_id ON storyboards(creator_id);

-- Rollback Step 5: Recreate old foreign key
ALTER TABLE storyboards
    ADD CONSTRAINT fk_storyboard_creator
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE;

COMMIT;
*/

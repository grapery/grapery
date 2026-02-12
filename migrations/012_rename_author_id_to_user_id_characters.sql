-- Migration: Rename author_id to user_id in characters table
-- Version: 012
-- Date: 2026-02-11
-- Author: Database Migration Team
-- Description: Rename author_id column to user_id for consistency across the schema
-- Priority: P0 - Critical
-- Breaking Change: Yes

-- =====================================================
-- MIGRATION UP
-- =====================================================

BEGIN;

-- Step 1: Drop old index if exists
SET @index_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'characters'
    AND INDEX_NAME = 'idx_author_id'
);

SET @drop_index_sql = IF(@index_exists > 0,
    'DROP INDEX idx_author_id ON characters',
    'SELECT ''Index idx_author_id does not exist, skipping'' AS message'
);

PREPARE stmt FROM @drop_index_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 2: Drop old foreign key if exists
SET @fk_exists = (
    SELECT COUNT(*) FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'characters'
    AND CONSTRAINT_NAME = 'fk_character_author'
    AND REFERENCED_TABLE_NAME = 'users'
);

SET @drop_fk_sql = IF(@fk_exists > 0,
    'ALTER TABLE characters DROP FOREIGN KEY fk_character_author',
    'SELECT ''Foreign key fk_character_author does not exist, skipping'' AS message'
);

PREPARE stmt FROM @drop_fk_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Step 3: Rename the column
ALTER TABLE characters
    CHANGE COLUMN author_id user_id VARCHAR(36) NOT NULL COMMENT '用户ID';

-- Step 4: Create new index
CREATE INDEX idx_user_id ON characters(user_id);

-- Step 5: Add new foreign key constraint
ALTER TABLE characters
    ADD CONSTRAINT fk_character_user
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
AND TABLE_NAME = 'characters'
AND COLUMN_NAME = 'user_id';

-- =====================================================
-- MIGRATION DOWN (ROLLBACK)
-- =====================================================

/*
BEGIN;

-- Rollback Step 1: Drop new foreign key
ALTER TABLE characters DROP FOREIGN KEY fk_character_user;

-- Rollback Step 2: Drop new index
DROP INDEX idx_user_id ON characters;

-- Rollback Step 3: Rename column back
ALTER TABLE characters
    CHANGE COLUMN user_id author_id VARCHAR(36) NOT NULL COMMENT '作者ID';

-- Rollback Step 4: Recreate old index
CREATE INDEX idx_author_id ON characters(author_id);

-- Rollback Step 5: Recreate old foreign key
ALTER TABLE characters
    ADD CONSTRAINT fk_character_author
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;

COMMIT;
*/

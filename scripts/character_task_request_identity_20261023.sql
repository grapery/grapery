-- Character creation request identity, MySQL 8. Apply via the registered
-- character_task_request_identity_20261023 migration, or this equivalent script.
-- Legacy rows remain NULL (multiple NULLs are allowed by the unique index).
-- Keys are SHA-256(user ID + separator + explicit retry identity), or a full
-- canonical manual-intent hash fallback for older clients without a retry key. This
-- migration does not invent identities or restart previously created tasks.
SET @character_request_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'character_generation_tasks' AND COLUMN_NAME = 'request_key') = 0,
  'ALTER TABLE character_generation_tasks ADD COLUMN request_key VARCHAR(64) NULL', 'SELECT 1');
PREPARE character_request_stmt FROM @character_request_ddl;
EXECUTE character_request_stmt;
DEALLOCATE PREPARE character_request_stmt;

SET @character_request_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'character_generation_tasks' AND COLUMN_NAME = 'intent_hash') = 0,
  'ALTER TABLE character_generation_tasks ADD COLUMN intent_hash VARCHAR(64) NULL', 'SELECT 1');
PREPARE character_request_stmt FROM @character_request_ddl;
EXECUTE character_request_stmt;
DEALLOCATE PREPARE character_request_stmt;

SET @character_request_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'character_generation_tasks' AND INDEX_NAME = 'idx_character_task_request_key') = 0,
  'CREATE UNIQUE INDEX idx_character_task_request_key ON character_generation_tasks(request_key)', 'SELECT 1');
PREPARE character_request_stmt FROM @character_request_ddl;
EXECUTE character_request_stmt;
DEALLOCATE PREPARE character_request_stmt;

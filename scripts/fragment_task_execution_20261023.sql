-- Manual pre-deploy schema preparation; never executed by application tests.
-- Idempotent on MySQL 8. Existing rows remain execution_version=0 and are
-- recoverable; new application inserts version=1 and explicitly marks readiness.
-- Select the Grapery database before running. DDL commits implicitly in MySQL.
SET @fragment_execution_schema = DATABASE();

SET @fragment_execution_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = @fragment_execution_schema AND TABLE_NAME = 'fragment_generation_tasks' AND COLUMN_NAME = 'lease_owner') = 0,
  'ALTER TABLE fragment_generation_tasks ADD COLUMN lease_owner VARCHAR(64) NOT NULL DEFAULT ''''', 'SELECT 1');
PREPARE fragment_execution_stmt FROM @fragment_execution_ddl;
EXECUTE fragment_execution_stmt;
DEALLOCATE PREPARE fragment_execution_stmt;
SET @fragment_execution_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = @fragment_execution_schema AND TABLE_NAME = 'fragment_generation_tasks' AND COLUMN_NAME = 'lease_fence') = 0,
  'ALTER TABLE fragment_generation_tasks ADD COLUMN lease_fence BIGINT NOT NULL DEFAULT 0', 'SELECT 1');
PREPARE fragment_execution_stmt FROM @fragment_execution_ddl;
EXECUTE fragment_execution_stmt;
DEALLOCATE PREPARE fragment_execution_stmt;
SET @fragment_execution_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = @fragment_execution_schema AND TABLE_NAME = 'fragment_generation_tasks' AND COLUMN_NAME = 'lease_expires_at') = 0,
  'ALTER TABLE fragment_generation_tasks ADD COLUMN lease_expires_at BIGINT NULL', 'SELECT 1');
PREPARE fragment_execution_stmt FROM @fragment_execution_ddl;
EXECUTE fragment_execution_stmt;
DEALLOCATE PREPARE fragment_execution_stmt;

SET @fragment_execution_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = @fragment_execution_schema AND TABLE_NAME = 'fragment_generation_tasks' AND COLUMN_NAME = 'checkpoint_json') = 0,
  'ALTER TABLE fragment_generation_tasks ADD COLUMN checkpoint_json LONGTEXT NULL', 'SELECT 1');
PREPARE fragment_execution_stmt FROM @fragment_execution_ddl;
EXECUTE fragment_execution_stmt;
DEALLOCATE PREPARE fragment_execution_stmt;

SET @fragment_execution_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = @fragment_execution_schema AND TABLE_NAME = 'fragment_generation_tasks' AND COLUMN_NAME = 'execution_version') = 0,
  'ALTER TABLE fragment_generation_tasks ADD COLUMN execution_version INT NOT NULL DEFAULT 0', 'SELECT 1');
PREPARE fragment_execution_stmt FROM @fragment_execution_ddl;
EXECUTE fragment_execution_stmt;
DEALLOCATE PREPARE fragment_execution_stmt;

SET @fragment_execution_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.COLUMNS
   WHERE TABLE_SCHEMA = @fragment_execution_schema AND TABLE_NAME = 'fragment_generation_tasks' AND COLUMN_NAME = 'ready_at') = 0,
  'ALTER TABLE fragment_generation_tasks ADD COLUMN ready_at BIGINT NULL', 'SELECT 1');
PREPARE fragment_execution_stmt FROM @fragment_execution_ddl;
EXECUTE fragment_execution_stmt;
DEALLOCATE PREPARE fragment_execution_stmt;

SET @fragment_execution_ddl = IF(
  (SELECT COUNT(*) FROM information_schema.STATISTICS
   WHERE TABLE_SCHEMA = @fragment_execution_schema AND TABLE_NAME = 'fragment_generation_tasks' AND INDEX_NAME = 'idx_fragment_gen_lease_expiry') = 0,
  'CREATE INDEX idx_fragment_gen_lease_expiry ON fragment_generation_tasks (lease_expires_at)', 'SELECT 1');
PREPARE fragment_execution_stmt FROM @fragment_execution_ddl;
EXECUTE fragment_execution_stmt;
DEALLOCATE PREPARE fragment_execution_stmt;

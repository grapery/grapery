-- Apply once before deploying the durable storyboard creation-idempotency code.
-- The application model migration also declares this column. This script is
-- intentionally not run by tests or development tooling.
ALTER TABLE storyboards
  ADD COLUMN creation_request_hash CHAR(64) NOT NULL DEFAULT '';
-- Existing rows keep an empty hash: an old idempotency mapping cannot be reused
-- as verified until it has an immutable original-request fingerprint. Never
-- infer that fingerprint from later edited/generated content.

-- Migration: Ensure is_collaboration_open field exists in stories table
-- Description: This field controls who can edit story
--   - true: All users can edit (open collaboration)
--   - false: Only story author and group members can edit (restricted collaboration)

-- Add is_collaboration_open column if it doesn't exist
ALTER TABLE stories
ADD COLUMN IF NOT EXISTS is_collaboration_open BOOLEAN
DEFAULT FALSE
NOT NULL
COMMENT 'Whether collaboration is open: true=anyone can edit, false=only author and group members can edit';

-- Create index for query performance if it doesn't exist
CREATE INDEX IF NOT EXISTS idx_stories_is_collaboration_open
ON stories(is_collaboration_open);

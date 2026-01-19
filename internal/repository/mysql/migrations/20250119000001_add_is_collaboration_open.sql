-- Migration: Add is_collaboration_open field to stories table
-- Description: This field controls who can edit the story
--   - true: All users can edit (open collaboration)
--   - false: Only story author and group members can edit (restricted collaboration)

-- Add is_collaboration_open column
ALTER TABLE stories 
ADD COLUMN is_collaboration_open BOOLEAN 
DEFAULT FALSE 
NOT NULL 
COMMENT 'Whether collaboration is open: true=anyone can edit, false=only author and group members can edit';

-- Create index for query performance
CREATE INDEX idx_stories_is_collaboration_open 
ON stories(is_collaboration_open);

-- Update existing stories to have restricted collaboration (default behavior)
-- Existing stories will have is_collaboration_open = FALSE (the default)
-- This is a conservative default to maintain current behavior

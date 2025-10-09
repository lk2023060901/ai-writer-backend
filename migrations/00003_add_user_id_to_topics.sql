-- +goose Up
-- Migration: Add user_id to topics table for direct user-topic relationship
-- This improves query performance by avoiding JOIN with agents table

-- Add user_id column (nullable first to allow existing data)
ALTER TABLE topics ADD COLUMN user_id UUID;

-- Backfill user_id from agents table for existing topics
UPDATE topics t
SET user_id = a.owner_id
FROM agents a
WHERE t.assistant_id = a.id AND t.user_id IS NULL;

-- Make user_id NOT NULL after backfilling
ALTER TABLE topics ALTER COLUMN user_id SET NOT NULL;

-- Add foreign key constraint
ALTER TABLE topics ADD CONSTRAINT fk_topics_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Add index for querying topics by user_id
CREATE INDEX idx_topics_user_id ON topics(user_id, updated_at DESC) WHERE deleted_at IS NULL;

-- Update comment
COMMENT ON COLUMN topics.user_id IS '对话主题所属用户 ID（外键到 users 表）';

-- +goose Down
-- Remove user_id column and related constraints
DROP INDEX IF EXISTS idx_topics_user_id;
ALTER TABLE topics DROP CONSTRAINT IF EXISTS fk_topics_user;
ALTER TABLE topics DROP COLUMN IF EXISTS user_id;

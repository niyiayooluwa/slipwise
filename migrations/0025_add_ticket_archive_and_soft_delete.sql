-- +goose Up
ALTER TABLE user_tickets
ADD COLUMN is_archived BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN archived_at TIMESTAMPTZ,
ADD COLUMN deleted_at TIMESTAMPTZ;

-- Partial composite index for active and archived user ticket history lookups
CREATE INDEX idx_user_tickets_active_lookup 
ON user_tickets (user_id, is_archived, created_at DESC) 
WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_user_tickets_active_lookup;

ALTER TABLE user_tickets
DROP COLUMN IF EXISTS deleted_at,
DROP COLUMN IF EXISTS archived_at,
DROP COLUMN IF EXISTS is_archived;

-- +goose Up
-- Allow users to not have a password if they sign up via OAuth
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- Track OAuth connections
CREATE TABLE oauth_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider TEXT NOT NULL, -- e.g., 'google', 'apple'
    provider_user_id TEXT NOT NULL, -- The unique ID from the provider
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_user_id)
);

CREATE INDEX idx_oauth_user ON oauth_connections (user_id);

-- +goose Down
DROP TABLE oauth_connections;
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;

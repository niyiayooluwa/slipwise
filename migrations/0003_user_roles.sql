-- +goose Up
ALTER TABLE users 
ADD COLUMN is_admin BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN is_punter BOOLEAN NOT NULL DEFAULT false,
ADD COLUMN is_suspended BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users 
DROP COLUMN is_admin,
DROP COLUMN is_punter,
DROP COLUMN is_suspended;

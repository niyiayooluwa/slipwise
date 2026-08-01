-- +goose Up
ALTER TABLE user_tickets ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE user_tickets DROP COLUMN description;

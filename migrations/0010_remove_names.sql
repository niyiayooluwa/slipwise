-- +goose Up
ALTER TABLE users DROP COLUMN first_name;
ALTER TABLE users DROP COLUMN last_name;

-- +goose Down
ALTER TABLE users ADD COLUMN first_name VARCHAR(100);
ALTER TABLE users ADD COLUMN last_name VARCHAR(100);

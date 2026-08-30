-- +goose Up
ALTER TABLE booking_codes ADD COLUMN created_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- +goose Down
ALTER TABLE booking_codes DROP COLUMN created_at;

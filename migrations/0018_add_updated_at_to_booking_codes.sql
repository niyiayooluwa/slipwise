-- +goose Up
ALTER TABLE booking_codes ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- +goose Down
ALTER TABLE booking_codes DROP COLUMN updated_at;

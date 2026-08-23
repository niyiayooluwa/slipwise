-- +goose Up
ALTER TABLE booking_codes DROP CONSTRAINT IF EXISTS booking_codes_provider_code_key;

-- +goose Down
ALTER TABLE booking_codes ADD CONSTRAINT booking_codes_provider_code_key UNIQUE(provider, code);

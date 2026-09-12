-- +goose Up
ALTER TABLE booking_codes 
    ADD COLUMN notified_status TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_booking_codes_notified_status 
ON booking_codes (id, status, notified_status);

-- +goose Down
DROP INDEX IF EXISTS idx_booking_codes_notified_status;

ALTER TABLE booking_codes 
    DROP COLUMN IF EXISTS notified_status;

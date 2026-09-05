-- +goose Up
CREATE INDEX IF NOT EXISTS idx_booking_selections_code_status 
ON booking_selections (booking_code_id, status);

-- +goose Down
DROP INDEX IF EXISTS idx_booking_selections_code_status;

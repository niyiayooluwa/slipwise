-- +goose Up
ALTER TABLE user_tickets DROP CONSTRAINT IF EXISTS user_tickets_user_id_booking_code_id_key;

-- +goose Down
ALTER TABLE user_tickets ADD CONSTRAINT user_tickets_user_id_booking_code_id_key UNIQUE(user_id, booking_code_id);

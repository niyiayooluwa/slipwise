-- +goose Up
DROP INDEX IF EXISTS idx_fast_settle;
CREATE INDEX idx_fast_settle ON booking_selections (match_id, market_type, selection, market_spec) WHERE status = 'PENDING';

-- +goose Down
DROP INDEX IF EXISTS idx_fast_settle;
CREATE INDEX idx_fast_settle ON booking_selections (match_id, market_type, selection) WHERE status = 'PENDING';

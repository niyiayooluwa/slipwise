-- +goose Up
ALTER TABLE matches ADD COLUMN provider TEXT NOT NULL DEFAULT '';
ALTER TABLE matches ADD COLUMN provider_id TEXT NOT NULL DEFAULT '';
ALTER TABLE matches ADD CONSTRAINT matches_provider_provider_id_key UNIQUE(provider, provider_id);

ALTER TABLE booking_selections DROP COLUMN IF EXISTS provider;
ALTER TABLE booking_selections DROP COLUMN IF EXISTS external_match_id;

DROP INDEX IF EXISTS idx_fast_settle;
CREATE INDEX idx_fast_settle ON booking_selections (match_id, market_type, selection) WHERE status = 'PENDING';

-- +goose Down
ALTER TABLE booking_selections ADD COLUMN provider TEXT NOT NULL DEFAULT '';
ALTER TABLE booking_selections ADD COLUMN external_match_id TEXT NOT NULL DEFAULT '';

DROP INDEX IF EXISTS idx_fast_settle;
CREATE INDEX idx_fast_settle ON booking_selections (provider, external_match_id) WHERE status = 'PENDING';

ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_provider_provider_id_key;
ALTER TABLE matches DROP COLUMN IF EXISTS provider_id;
ALTER TABLE matches DROP COLUMN IF EXISTS provider;

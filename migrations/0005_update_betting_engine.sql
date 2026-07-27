-- +goose Up
-- Update matches table
ALTER TABLE matches ADD COLUMN start_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now();
ALTER TABLE matches ALTER COLUMN status SET DEFAULT 'PENDING';
UPDATE matches SET status = 'PENDING' WHERE status = 'NOT_STARTED';

-- Update booking_codes table
ALTER TABLE booking_codes RENAME COLUMN bookie TO provider;
ALTER TABLE booking_codes ADD COLUMN total_odds DECIMAL(10, 2);
ALTER TABLE booking_codes DROP CONSTRAINT IF EXISTS booking_codes_code_key;
ALTER TABLE booking_codes ADD CONSTRAINT booking_codes_provider_code_key UNIQUE(provider, code);

-- Update booking_selections table
ALTER TABLE booking_selections DROP CONSTRAINT IF EXISTS booking_selections_match_id_fkey;
ALTER TABLE booking_selections ADD CONSTRAINT booking_selections_match_id_fkey FOREIGN KEY (match_id) REFERENCES matches(id) ON DELETE CASCADE;

ALTER TABLE booking_selections ADD COLUMN provider TEXT NOT NULL DEFAULT '';
ALTER TABLE booking_selections ADD COLUMN external_match_id TEXT NOT NULL DEFAULT '';
ALTER TABLE booking_selections ADD COLUMN market_spec TEXT;
ALTER TABLE booking_selections ADD COLUMN odds DECIMAL(10, 2);

DROP INDEX IF EXISTS idx_booking_buckets;
CREATE INDEX idx_fast_settle ON booking_selections (provider, external_match_id) WHERE status = 'PENDING';

-- Update user_tickets table
ALTER TABLE user_tickets DROP CONSTRAINT IF EXISTS user_tickets_user_id_fkey;
ALTER TABLE user_tickets ADD CONSTRAINT user_tickets_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE user_tickets DROP CONSTRAINT IF EXISTS user_tickets_booking_code_id_fkey;
ALTER TABLE user_tickets ADD CONSTRAINT user_tickets_booking_code_id_fkey FOREIGN KEY (booking_code_id) REFERENCES booking_codes(id) ON DELETE CASCADE;

ALTER TABLE user_tickets ADD COLUMN stake DECIMAL(10, 2);
ALTER TABLE user_tickets ADD COLUMN created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
ALTER TABLE user_tickets ADD CONSTRAINT user_tickets_user_id_booking_code_id_key UNIQUE(user_id, booking_code_id);

-- +goose Down
-- Revert user_tickets
ALTER TABLE user_tickets DROP CONSTRAINT IF EXISTS user_tickets_user_id_booking_code_id_key;
ALTER TABLE user_tickets DROP COLUMN IF EXISTS created_at;
ALTER TABLE user_tickets DROP COLUMN IF EXISTS stake;

ALTER TABLE user_tickets DROP CONSTRAINT IF EXISTS user_tickets_booking_code_id_fkey;
ALTER TABLE user_tickets ADD CONSTRAINT user_tickets_booking_code_id_fkey FOREIGN KEY (booking_code_id) REFERENCES booking_codes(id);

ALTER TABLE user_tickets DROP CONSTRAINT IF EXISTS user_tickets_user_id_fkey;
ALTER TABLE user_tickets ADD CONSTRAINT user_tickets_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id);

-- Revert booking_selections
DROP INDEX IF EXISTS idx_fast_settle;
CREATE INDEX idx_booking_buckets ON booking_selections (match_id, market_type, selection) WHERE status = 'PENDING';

ALTER TABLE booking_selections DROP COLUMN IF EXISTS odds;
ALTER TABLE booking_selections DROP COLUMN IF EXISTS market_spec;
ALTER TABLE booking_selections DROP COLUMN IF EXISTS external_match_id;
ALTER TABLE booking_selections DROP COLUMN IF EXISTS provider;

ALTER TABLE booking_selections DROP CONSTRAINT IF EXISTS booking_selections_match_id_fkey;
ALTER TABLE booking_selections ADD CONSTRAINT booking_selections_match_id_fkey FOREIGN KEY (match_id) REFERENCES matches(id);

-- Revert booking_codes
ALTER TABLE booking_codes DROP CONSTRAINT IF EXISTS booking_codes_provider_code_key;
ALTER TABLE booking_codes ADD CONSTRAINT booking_codes_code_key UNIQUE(code);
ALTER TABLE booking_codes DROP COLUMN IF EXISTS total_odds;
ALTER TABLE booking_codes RENAME COLUMN provider TO bookie;

-- Revert matches
UPDATE matches SET status = 'NOT_STARTED' WHERE status = 'PENDING';
ALTER TABLE matches ALTER COLUMN status SET DEFAULT 'NOT_STARTED';
ALTER TABLE matches DROP COLUMN IF EXISTS start_time;

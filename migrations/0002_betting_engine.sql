-- +goose Up
CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    home_team TEXT NOT NULL,
    away_team TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'NOT_STARTED'
);

CREATE TABLE booking_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bookie TEXT NOT NULL, 
    code TEXT UNIQUE NOT NULL, 
    status TEXT NOT NULL DEFAULT 'PENDING'
);

CREATE TABLE booking_selections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_code_id UUID NOT NULL REFERENCES booking_codes(id) ON DELETE CASCADE,
    match_id UUID NOT NULL REFERENCES matches(id),
    market_type TEXT NOT NULL, 
    selection TEXT NOT NULL,   
    status TEXT NOT NULL DEFAULT 'PENDING'
);

CREATE INDEX idx_booking_buckets ON booking_selections (match_id, market_type, selection) 
WHERE status = 'PENDING';

CREATE TABLE user_tickets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    booking_code_id UUID NOT NULL REFERENCES booking_codes(id)
);

-- +goose Down
DROP TABLE user_tickets;
DROP TABLE booking_selections;
DROP TABLE booking_codes;
DROP TABLE matches;

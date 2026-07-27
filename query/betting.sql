-- name: CreateMatch :one
INSERT INTO matches (home_team, away_team, start_time, status) 
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: CreateBookingCode :one
INSERT INTO booking_codes (provider, code, total_odds, status) 
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: CreateBookingSelection :one
INSERT INTO booking_selections (booking_code_id, match_id, provider, external_match_id, market_type, market_spec, selection, odds, status) 
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *;

-- name: CreateUserTicket :one
INSERT INTO user_tickets (user_id, booking_code_id, stake) 
VALUES ($1, $2, $3) RETURNING *;

-- name: GetActiveBucketsByProvider :many
-- Used by the Background Poller to find out what matches to fetch
SELECT DISTINCT external_match_id 
FROM booking_selections 
WHERE provider = $1 AND status = 'PENDING';

-- name: UpdateSelectionStatus :many
-- The Fast Settlement query!
UPDATE booking_selections 
SET status = $1 
WHERE provider = $2 
  AND external_match_id = $3 
  AND market_type = $4 
  AND selection = $5 
  AND status = 'PENDING'
RETURNING booking_code_id;

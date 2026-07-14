-- name: CreateMatch :one
INSERT INTO matches (home_team, away_team, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateBookingCode :one
INSERT INTO booking_codes (bookie, code, status)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetBookingCodeByCode :one
SELECT * FROM booking_codes WHERE code = $1 LIMIT 1;

-- name: CreateBookingSelection :one
INSERT INTO booking_selections (booking_code_id, match_id, market_type, selection, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateUserTicket :one
INSERT INTO user_tickets (user_id, booking_code_id)
VALUES ($1, $2)
RETURNING *;

-- name: EvaluateBucket :many
UPDATE booking_selections 
SET status = $4 
WHERE match_id = $1 
  AND market_type = $2 
  AND selection = $3 
  AND status = 'PENDING'
RETURNING booking_code_id;

-- name: GetUsersForBookingCodes :many
SELECT user_id 
FROM user_tickets 
WHERE booking_code_id = ANY($1::uuid[]);

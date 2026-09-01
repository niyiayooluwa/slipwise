-- name: CreateMatch :one
INSERT INTO matches (home_team, away_team, start_time, status, provider, provider_id) 
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (provider, provider_id) DO UPDATE SET status = EXCLUDED.status
RETURNING *;

-- name: CreateBookingCode :one
INSERT INTO booking_codes (provider, code, total_odds, status) 
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateBookingSelection :one
INSERT INTO booking_selections (booking_code_id, match_id, market_type, market_spec, selection, odds, status) 
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: CreateUserTicket :one
INSERT INTO user_tickets (user_id, booking_code_id, stake, description) 
VALUES ($1, $2, $3, $4) RETURNING *;

-- name: UpsertUserTrack :one
INSERT INTO user_tickets (user_id, booking_code_id, stake, description) 
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetActiveBucketsByProvider :many
-- Used by the Background Poller to find out what matches to fetch
SELECT DISTINCT m.id, m.provider_id 
FROM booking_selections bs
JOIN matches m ON bs.match_id = m.id
WHERE m.provider = $1 AND bs.status = 'PENDING';

-- name: GetPendingBucketsForMatch :many
SELECT DISTINCT market_type, market_spec, selection 
FROM booking_selections 
WHERE match_id = $1 AND status = 'PENDING';

-- name: UpdateSelectionStatus :many
-- The Fast Settlement query!
UPDATE booking_selections 
SET status = $1 
WHERE match_id = $2 
  AND market_type = $3 
  AND selection = $4 
  AND status = 'PENDING'
RETURNING booking_code_id;

-- name: DeleteUserTicket :exec
DELETE FROM user_tickets 
WHERE id = $1 AND user_id = $2;

-- name: GetUserHistory :many
SELECT 
    ut.id AS ticket_id,
    ut.stake,
    ut.description,
    ut.created_at AS tracked_at,
    bc.provider,
    bc.code,
    bc.total_odds,
    bc.status AS overall_status,
    bc.updated_at
FROM user_tickets ut
JOIN booking_codes bc ON ut.booking_code_id = bc.id
WHERE ut.user_id = $1
  AND (sqlc.narg('status')::text IS NULL OR bc.status = sqlc.narg('status')::text)
  AND (sqlc.narg('since')::timestamptz IS NULL OR bc.updated_at > sqlc.narg('since')::timestamptz)
ORDER BY ut.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUserHistory :one
SELECT COUNT(*) 
FROM user_tickets ut
JOIN booking_codes bc ON ut.booking_code_id = bc.id
WHERE ut.user_id = $1
  AND (sqlc.narg('status')::text IS NULL OR bc.status = sqlc.narg('status')::text);

-- name: UpdateUserTicket :one
UPDATE user_tickets
SET stake = COALESCE(sqlc.narg('stake'), stake),
    description = COALESCE(sqlc.narg('description'), description)
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: GetTicketDetails :many
SELECT 
    bs.id AS selection_id,
    bs.market_type,
    bs.market_spec,
    bs.selection,
    bs.odds,
    bs.status AS selection_status,
    m.home_team,
    m.away_team,
    m.start_time,
    m.status AS match_status,
    m.home_score,
    m.away_score,
    m.live_time
FROM booking_selections bs
JOIN matches m ON bs.match_id = m.id
JOIN user_tickets ut ON ut.booking_code_id = bs.booking_code_id
WHERE ut.id = $1 AND ut.user_id = $2;

-- name: CleanupOrphanedBookingCodes :exec
DELETE FROM booking_codes
WHERE id NOT IN (SELECT booking_code_id FROM user_tickets)
AND created_at < NOW() - INTERVAL '7 days';

-- name: UpdateMatchState :exec
UPDATE matches
SET home_score = $1, away_score = $2, status = $3, live_time = $4
WHERE id = $5;

-- name: GetStuckMatches :many
-- Returns matches that started 3+ hours ago but are still showing as LIVE/NOT_STARTED.
-- These are matches that disappeared from the SportyBet live firehose (i.e., they ended)
-- but our evaluator never received an ENDED signal for them.
-- We include the last known scores so the sweeper can force-settle without hitting Cloudflare.
SELECT id, provider_id, home_score, away_score
FROM matches 
WHERE status IN ('NOT_STARTED', 'LIVE')
  AND start_time < NOW() - INTERVAL '3 hours'
LIMIT 5;

-- name: GetAdminDashboardStats :one
SELECT 
    (SELECT COUNT(*) FROM users) AS total_users,
    (SELECT COUNT(*) FROM booking_codes) AS total_booking_codes,
    (SELECT COUNT(*) FROM user_tickets) AS total_tracked_tickets,
    (SELECT COUNT(*) FROM user_tickets ut JOIN booking_codes bc ON ut.booking_code_id = bc.id WHERE bc.status = 'WON') AS won_tickets,
    (SELECT COUNT(*) FROM user_tickets ut JOIN booking_codes bc ON ut.booking_code_id = bc.id WHERE bc.status = 'LOST') AS lost_tickets,
    (SELECT COUNT(*) FROM user_tickets ut JOIN booking_codes bc ON ut.booking_code_id = bc.id WHERE bc.status = 'PENDING') AS pending_tickets;

-- name: GetAdminUsers :many
SELECT 
    id, email, username, email_verified_at, is_admin, is_punter, is_suspended, created_at, updated_at
FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAdminUsersCount :one
SELECT COUNT(*) FROM users;

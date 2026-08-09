-- name: CreateUser :one
INSERT INTO users (first_name, last_name, email, password_hash)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: MarkEmailVerified :exec
UPDATE users SET email_verified_at = now(), updated_at = now() WHERE id = $1;

-- name: CreateOTP :one
INSERT INTO otp_codes (email, code_hash, purpose, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetLatestOTP :one
SELECT * FROM otp_codes
WHERE email = $1 AND purpose = $2 AND used_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: IncrementOTPAttempts :exec
UPDATE otp_codes SET attempt_count = attempt_count + 1 WHERE id = $1;

-- name: MarkOTPUsed :exec
UPDATE otp_codes SET used_at = now() WHERE id = $1;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token_hash = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $1, updated_at = now() WHERE email = $2;

-- name: UpdateUserProfile :one
UPDATE users
SET 
  first_name = COALESCE(sqlc.narg('first_name'), first_name),
  last_name = COALESCE(sqlc.narg('last_name'), last_name),
  username = COALESCE(sqlc.narg('username'), username),
  updated_at = now()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: CheckUsernameExists :one
SELECT EXISTS(SELECT 1 FROM users WHERE username = $1);

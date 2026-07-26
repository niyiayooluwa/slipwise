-- name: CreateOAuthConnection :exec
INSERT INTO oauth_connections (user_id, provider, provider_user_id)
VALUES ($1, $2, $3);

-- name: GetUserByOAuthProvider :one
SELECT u.* 
FROM users u
JOIN oauth_connections o ON u.id = o.user_id
WHERE o.provider = $1 AND o.provider_user_id = $2;

-- name: GetOAuthProvidersForUser :many
SELECT provider FROM oauth_connections WHERE user_id = $1;

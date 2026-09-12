-- name: UpsertDeviceToken :exec
INSERT INTO user_devices (user_id, fcm_token, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (fcm_token) 
DO UPDATE SET user_id = EXCLUDED.user_id, updated_at = NOW();

-- name: DeleteDeviceToken :exec
DELETE FROM user_devices
WHERE user_id = $1 AND fcm_token = $2;

-- name: GetUserDeviceTokens :many
SELECT fcm_token
FROM user_devices
WHERE user_id = $1;

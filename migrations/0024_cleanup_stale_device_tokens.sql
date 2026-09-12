-- +goose Up
-- Keep only the most recently updated device record per fcm_token
DELETE FROM user_devices ud1
USING user_devices ud2
WHERE ud1.fcm_token = ud2.fcm_token
  AND ud1.updated_at < ud2.updated_at;

-- Enforce unique fcm_token globally so a token is never duplicated
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_devices_fcm_token 
ON user_devices (fcm_token);

-- +goose Down
DROP INDEX IF EXISTS idx_user_devices_fcm_token;

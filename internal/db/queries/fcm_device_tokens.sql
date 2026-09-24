-- name: DeleteOldestDeviceTokensBeyondLimit :exec
DELETE FROM fcm_device_tokens
WHERE user_id = $1
  AND id NOT IN (
      SELECT id FROM fcm_device_tokens
      WHERE user_id = $1
      ORDER BY created_at DESC
      LIMIT $2
  );
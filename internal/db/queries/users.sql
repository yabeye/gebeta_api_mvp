-- =============================================================================
-- USERS
-- =============================================================================

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 AND deleted_at IS NULL;

-- name: UpdateUserPhone :one
UPDATE users
SET phone = $2
WHERE id = $1
RETURNING *;

-- name: UpdateUserStatus :one
UPDATE users
SET status = $2
WHERE id = $1
RETURNING *;

-- name: SoftDeleteUser :exec
UPDATE users
SET status = 'deleted', deleted_at = now()
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- =============================================================================
-- USER PROFILES (1:1 with users)
-- =============================================================================

-- name: GetUserProfile :one
SELECT * FROM user_profiles
WHERE user_id = $1;

-- name: UpsertUserProfile :one
INSERT INTO user_profiles (user_id, first_name, last_name, profile_image_url, birth_date)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET
    first_name        = EXCLUDED.first_name,
    last_name         = EXCLUDED.last_name,
    profile_image_url = EXCLUDED.profile_image_url,
    birth_date        = EXCLUDED.birth_date,
    updated_at        = now()
RETURNING *;

-- name: GetUserWithProfile :one
SELECT
    u.id, u.email, u.phone, u.status, u.created_at, u.updated_at,
    p.first_name, p.last_name, p.profile_image_url, p.birth_date
FROM users u
LEFT JOIN user_profiles p ON p.user_id = u.id
WHERE u.id = $1 AND u.deleted_at IS NULL;

-- =============================================================================
-- USER ROLES (1:1 with users)
-- =============================================================================

-- name: GetUserRole :one
SELECT * FROM user_roles
WHERE user_id = $1;

-- name: SetUserRole :one
INSERT INTO user_roles (user_id, role)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET
    role = EXCLUDED.role
RETURNING *;

-- name: ListUsersByRole :many
SELECT u.*
FROM users u
JOIN user_roles r ON r.user_id = u.id
WHERE r.role = $1 AND u.deleted_at IS NULL
ORDER BY u.created_at DESC
LIMIT $2 OFFSET $3;

-- =============================================================================
-- FCM DEVICE TOKENS (1:many with users)
-- =============================================================================

-- name: UpsertFCMDeviceToken :one
INSERT INTO fcm_device_tokens (user_id, token, platform)
VALUES ($1, $2, $3)
ON CONFLICT (token) DO UPDATE SET
    user_id  = EXCLUDED.user_id,
    platform = EXCLUDED.platform
RETURNING *;

-- name: ListDeviceTokensByUser :many
SELECT * FROM fcm_device_tokens
WHERE user_id = $1;

-- name: DeleteDeviceToken :exec
DELETE FROM fcm_device_tokens
WHERE token = $1 AND user_id = $2;

-- =============================================================================
-- NOTIFICATIONS (1:many with users)
-- =============================================================================

-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, title, description, image_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListNotificationsByUser :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkNotificationRead :exec
UPDATE notifications
SET is_read = true
WHERE id = $1 AND user_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications
SET is_read = true
WHERE user_id = $1 AND is_read = false;

-- name: CountUnreadNotifications :one
SELECT count(*) FROM notifications
WHERE user_id = $1 AND is_read = false;

-- =============================================================================
-- RIDERS (1:1 with users)
-- =============================================================================

-- name: GetRiderByUserID :one
SELECT
    user_id, status, is_available, assigned_area,
    ST_Y(current_location::geometry) AS lat,
    ST_X(current_location::geometry) AS lng,
    updated_at
FROM riders
WHERE user_id = $1;

-- name: UpsertRider :one
INSERT INTO riders (user_id, status, is_available, assigned_area, current_location)
VALUES (
    $1, $2, $3, $4,
    ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float8, sqlc.arg(lat)::float8), 4326)::geography
)
ON CONFLICT (user_id) DO UPDATE SET
    status           = EXCLUDED.status,
    is_available     = EXCLUDED.is_available,
    assigned_area    = EXCLUDED.assigned_area,
    current_location = EXCLUDED.current_location,
    updated_at       = now()
RETURNING
    user_id, status, is_available, assigned_area,
    ST_Y(current_location::geometry) AS lat,
    ST_X(current_location::geometry) AS lng,
    updated_at;

-- name: UpdateRiderLocation :exec
UPDATE riders
SET
    current_location = ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float8, sqlc.arg(lat)::float8), 4326)::geography,
    updated_at = now()
WHERE user_id = $1;

-- name: ListAvailableRidersNearby :many
SELECT
    user_id, status, is_available, assigned_area,
    ST_Y(current_location::geometry) AS lat,
    ST_X(current_location::geometry) AS lng,
    ST_Distance(current_location, ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float8, sqlc.arg(lat)::float8), 4326)::geography) AS distance_meters
FROM riders
WHERE is_available = true
  AND ST_DWithin(
        current_location,
        ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float8, sqlc.arg(lat)::float8), 4326)::geography,
        sqlc.arg(radius_meters)::float8
      )
ORDER BY distance_meters ASC
LIMIT $1;
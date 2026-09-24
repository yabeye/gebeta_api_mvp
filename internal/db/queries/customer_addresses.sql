-- name: CreateCustomerAddress :one
INSERT INTO customer_addresses (user_id, label, address, location, delivery_note, is_default)
VALUES (
    sqlc.arg(user_id), sqlc.arg(label), sqlc.arg(address),
    ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float8, sqlc.arg(lat)::float8), 4326)::geography,
    sqlc.arg(delivery_note), sqlc.arg(is_default)
)
RETURNING
    id, user_id, label, address,
    ST_Y(location::geometry)::float8 AS lat,
    ST_X(location::geometry)::float8 AS lng,
    delivery_note, is_default, created_at, updated_at;

-- name: UpdateCustomerAddress :one
UPDATE customer_addresses
SET
    label = sqlc.arg(label), address = sqlc.arg(address),
    location = ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float8, sqlc.arg(lat)::float8), 4326)::geography,
    delivery_note = sqlc.arg(delivery_note), is_default = sqlc.arg(is_default),
    updated_at = now()
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING
    id, user_id, label, address,
    ST_Y(location::geometry)::float8 AS lat,
    ST_X(location::geometry)::float8 AS lng,
    delivery_note, is_default, created_at, updated_at;

-- name: GetCustomerAddressByID :one
SELECT
    id, user_id, label, address,
    ST_Y(location::geometry)::float8 AS lat,
    ST_X(location::geometry)::float8 AS lng,
    delivery_note, is_default, created_at, updated_at
FROM customer_addresses
WHERE id = $1 AND user_id = $2;

-- name: ListCustomerAddressesByUser :many
SELECT
    id, user_id, label, address,
    ST_Y(location::geometry)::float8 AS lat,
    ST_X(location::geometry)::float8 AS lng,
    delivery_note, is_default, created_at, updated_at
FROM customer_addresses
WHERE user_id = $1
ORDER BY is_default DESC, created_at DESC;

-- name: CountAddressesByUser :one
SELECT count(*) FROM customer_addresses WHERE user_id = $1;

-- name: UnsetDefaultAddressForUser :exec
UPDATE customer_addresses
SET is_default = false, updated_at = now()
WHERE user_id = $1 AND is_default = true;

-- name: DeleteCustomerAddress :exec
DELETE FROM customer_addresses
WHERE id = $1 AND user_id = $2;

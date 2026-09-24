-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

-- +goose Down
-- Not dropping postgis here — other tables may depend on it later,
-- and CASCADE dropping an extension is destructive. Handle manually if ever needed.
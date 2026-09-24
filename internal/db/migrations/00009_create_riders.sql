-- +goose Up
CREATE TABLE riders (
    user_id          uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    status           rider_status NOT NULL DEFAULT 'offline',
    is_available     boolean NOT NULL DEFAULT false,
    assigned_area    text,
    current_location geography(Point, 4326),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_riders_current_location ON riders USING GIST(current_location);
CREATE INDEX idx_riders_is_available ON riders(is_available) WHERE is_available = true;

CREATE TRIGGER trg_riders_set_updated_at
BEFORE UPDATE ON riders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS riders;
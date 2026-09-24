-- +goose Up
CREATE TABLE customer_addresses (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label         text NOT NULL,
    address       text NOT NULL,
    location      geography(Point, 4326) NOT NULL,
    delivery_note text,
    is_default    boolean NOT NULL DEFAULT false,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_customer_addresses_user_id ON customer_addresses(user_id);
CREATE INDEX idx_customer_addresses_location ON customer_addresses USING GIST(location);

-- only one default address per user
CREATE UNIQUE INDEX idx_customer_addresses_one_default_per_user
    ON customer_addresses(user_id) WHERE is_default = true;

CREATE TRIGGER trg_customer_addresses_set_updated_at
BEFORE UPDATE ON customer_addresses
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS customer_addresses;
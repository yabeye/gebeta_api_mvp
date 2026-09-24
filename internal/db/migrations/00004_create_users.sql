-- +goose Up
CREATE TABLE users (
    id         uuid PRIMARY KEY REFERENCES auth.users(id) ON DELETE CASCADE,
    email      text NOT NULL UNIQUE,
    phone      text UNIQUE,
    status     user_status NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE INDEX idx_users_status ON users(status) WHERE deleted_at IS NULL;

CREATE TRIGGER trg_users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS users;
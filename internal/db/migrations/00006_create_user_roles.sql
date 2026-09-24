-- +goose Up
CREATE TABLE user_roles (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    role    app_role NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS user_roles;
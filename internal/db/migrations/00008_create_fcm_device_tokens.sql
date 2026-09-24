-- +goose Up
CREATE TABLE fcm_device_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      text NOT NULL UNIQUE,
    platform   device_platform NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_fcm_device_tokens_user_id ON fcm_device_tokens(user_id);

-- +goose Down
DROP TABLE IF EXISTS fcm_device_tokens;
-- +goose Up
CREATE TABLE notifications (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        text NOT NULL,
    title       text NOT NULL,
    description text,
    image_url   text,
    is_read     boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_id_created_at ON notifications(user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS notifications;
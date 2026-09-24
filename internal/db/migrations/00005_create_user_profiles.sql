-- +goose Up
CREATE TABLE user_profiles (
    user_id          uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name       text,
    last_name        text,
    profile_image_url text,
    birth_date       date,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_user_profiles_set_updated_at
BEFORE UPDATE ON user_profiles
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS user_profiles;
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION handle_new_auth_user()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO public.users (id, email, phone)
    VALUES (NEW.id, NEW.email, NEW.phone)
    ON CONFLICT (id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
-- +goose StatementEnd

CREATE TRIGGER trg_on_auth_user_created
AFTER INSERT ON auth.users
FOR EACH ROW EXECUTE FUNCTION handle_new_auth_user();

-- +goose Down
DROP TRIGGER IF EXISTS trg_on_auth_user_created ON auth.users;
-- +goose StatementBegin
DROP FUNCTION IF EXISTS handle_new_auth_user();
-- +goose StatementEnd
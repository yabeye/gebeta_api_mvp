-- +goose Up
CREATE TYPE user_status AS ENUM ('active', 'suspended', 'deleted');
CREATE TYPE app_role AS ENUM ('customer', 'rider', 'store_owner' ,'admin');
CREATE TYPE device_platform AS ENUM ('ios', 'android', 'web');
CREATE TYPE rider_status AS ENUM ('offline', 'online', 'on_delivery');

-- +goose Down
DROP TYPE IF EXISTS rider_status;
DROP TYPE IF EXISTS device_platform;
DROP TYPE IF EXISTS app_role;
DROP TYPE IF EXISTS user_status;
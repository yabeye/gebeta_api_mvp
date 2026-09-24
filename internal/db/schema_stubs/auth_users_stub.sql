-- This file exists only so sqlc can resolve the auth.users FK reference.
-- Supabase manages the real auth schema — never run this via goose.
CREATE SCHEMA IF NOT EXISTS auth;
CREATE TABLE IF NOT EXISTS auth.users (
    id uuid PRIMARY KEY
);
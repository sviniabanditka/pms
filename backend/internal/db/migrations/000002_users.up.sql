CREATE TABLE users (
    id                uuid PRIMARY KEY,
    email             citext NOT NULL UNIQUE,
    email_verified_at timestamptz,
    name              text NOT NULL DEFAULT '',
    avatar_url        text,
    password_hash     text,
    preferences       jsonb NOT NULL DEFAULT '{}'::jsonb,
    last_active_at    timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    version           bigint NOT NULL DEFAULT 0,
    deleted_at        timestamptz
);

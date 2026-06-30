-- API-токены (Personal Access Tokens) для публичного API.
-- БЕЗ RLS: auth-артефакт, ищется по token_hash ДО установки ws-контекста (как users).
-- Хранится только sha256-хеш; raw-секрет (pms_<rand>) показывается один раз при создании.
CREATE TABLE api_tokens (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    user_id      uuid NOT NULL REFERENCES users(id),
    name         text NOT NULL,
    token_hash   text NOT NULL UNIQUE,
    prefix       text NOT NULL,          -- для отображения в UI (pms_ab12…)
    last_used_at timestamptz,
    expires_at   timestamptz,            -- NULL = бессрочный
    revoked_at   timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX api_tokens_ws ON api_tokens (workspace_id);

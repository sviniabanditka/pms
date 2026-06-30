-- Webhooks: подписки воркспейса на события + лог доставок.
CREATE TABLE webhooks (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    url          text NOT NULL,
    secret       text NOT NULL,                 -- shared secret для HMAC-подписи (whsec_…)
    events       text[] NOT NULL DEFAULT '{}',  -- 'issue.create' | 'issue.*' | '*'
    active       boolean NOT NULL DEFAULT true,
    created_by   uuid NOT NULL REFERENCES users(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);
CREATE INDEX webhooks_ws ON webhooks (workspace_id) WHERE deleted_at IS NULL;
ALTER TABLE webhooks ENABLE ROW LEVEL SECURITY;
CREATE POLICY webhooks_tenant ON webhooks
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE webhook_deliveries (
    id            uuid PRIMARY KEY,
    webhook_id    uuid NOT NULL REFERENCES webhooks(id),
    workspace_id  uuid NOT NULL REFERENCES workspaces(id),
    event         text NOT NULL,
    status        text NOT NULL DEFAULT 'pending', -- pending | success | failed
    response_code int,
    attempts      int NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    delivered_at  timestamptz
);
CREATE INDEX webhook_deliveries_hook ON webhook_deliveries (webhook_id, created_at DESC);
ALTER TABLE webhook_deliveries ENABLE ROW LEVEL SECURITY;
CREATE POLICY webhook_deliveries_tenant ON webhook_deliveries
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

-- Курсор диспетчера (читает change_log по seq). Одна строка, без RLS (только jobs).
-- Стартуем с текущего max(seq): вебхуки шлются только для БУДУЩИХ событий (бэклог не фолбэчим).
CREATE TABLE webhook_cursor (
    only_one boolean PRIMARY KEY DEFAULT true,
    last_seq bigint NOT NULL DEFAULT 0
);
INSERT INTO webhook_cursor (only_one, last_seq) VALUES (true, COALESCE((SELECT MAX(seq) FROM change_log), 0));

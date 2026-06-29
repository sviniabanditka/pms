CREATE TABLE calendar_events (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    user_id      uuid NOT NULL,                    -- владелец/чей календарь
    kind         text NOT NULL DEFAULT 'event',    -- event | dayoff
    title        text NOT NULL DEFAULT '',
    description  text NOT NULL DEFAULT '',
    starts_at    timestamptz NOT NULL,
    ends_at      timestamptz NOT NULL,
    all_day      boolean NOT NULL DEFAULT false,
    created_by   uuid NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    deleted_at   timestamptz
);
CREATE INDEX calendar_events_ws_user ON calendar_events (workspace_id, user_id, starts_at);
ALTER TABLE calendar_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY calendar_events_tenant ON calendar_events
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

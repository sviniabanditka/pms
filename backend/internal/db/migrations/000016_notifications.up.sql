CREATE TABLE notifications (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    user_id      uuid NOT NULL REFERENCES users(id),
    type         text NOT NULL,        -- assigned | state_changed | commented | mentioned | …
    entity_type  text NOT NULL,        -- issue | …
    entity_id    uuid NOT NULL,
    actor_id     uuid REFERENCES users(id),
    payload      jsonb,
    read_at      timestamptz,
    at           timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0
);
CREATE INDEX notifications_user ON notifications (user_id, at DESC);
ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
CREATE POLICY notifications_tenant ON notifications
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

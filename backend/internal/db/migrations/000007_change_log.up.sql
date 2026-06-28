CREATE TABLE change_log (
    seq            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    workspace_id   uuid NOT NULL REFERENCES workspaces(id),
    entity_type    text NOT NULL,
    entity_id      uuid NOT NULL,
    version        bigint NOT NULL,
    op             text NOT NULL CHECK (op IN ('create','update','delete','archive')),
    changed_fields jsonb,
    actor_id       uuid REFERENCES users(id),
    at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX change_log_ws_seq ON change_log (workspace_id, seq);
CREATE INDEX change_log_entity ON change_log (entity_type, entity_id);

ALTER TABLE change_log ENABLE ROW LEVEL SECURITY;
CREATE POLICY change_log_tenant ON change_log
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

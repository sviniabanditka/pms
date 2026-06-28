CREATE TABLE attachments (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    entity_type  text NOT NULL,           -- issue | message | comment | doc | …
    entity_id    uuid NOT NULL,
    uploader     uuid NOT NULL,
    filename     text NOT NULL,
    content_type text NOT NULL DEFAULT '',
    size         bigint NOT NULL DEFAULT 0,
    s3_key       text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    deleted_at   timestamptz
);
CREATE INDEX attachments_entity ON attachments (workspace_id, entity_type, entity_id);
ALTER TABLE attachments ENABLE ROW LEVEL SECURITY;
CREATE POLICY attachments_tenant ON attachments USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

-- аватарки/лого: ключ объекта в S3 (NULL = дефолтная буква-аватар)
ALTER TABLE users ADD COLUMN avatar_key text;
ALTER TABLE workspaces ADD COLUMN logo_key text;

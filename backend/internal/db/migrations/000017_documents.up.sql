CREATE TABLE documents (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    parent_id    uuid REFERENCES documents(id),
    scope_kind   text NOT NULL DEFAULT 'team',
    team_id      uuid REFERENCES teams(id),
    project_id   uuid,
    title        text NOT NULL,
    body         jsonb NOT NULL DEFAULT '{}'::jsonb,
    icon         text,
    position     text,
    created_by   uuid NOT NULL REFERENCES users(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    archived_at  timestamptz,
    deleted_at   timestamptz
);
CREATE INDEX documents_team ON documents (workspace_id, team_id);
CREATE INDEX documents_parent ON documents (parent_id);
ALTER TABLE documents ENABLE ROW LEVEL SECURITY;
CREATE POLICY documents_tenant ON documents
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

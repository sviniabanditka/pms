CREATE TABLE comments (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    issue_id     uuid NOT NULL REFERENCES issues(id),
    author_id    uuid NOT NULL REFERENCES users(id),
    body         text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    edited_at    timestamptz,
    deleted_at   timestamptz
);
CREATE INDEX comments_issue ON comments (issue_id, created_at);
ALTER TABLE comments ENABLE ROW LEVEL SECURITY;
CREATE POLICY comments_tenant ON comments USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE time_entries (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    user_id      uuid NOT NULL,
    issue_id     uuid REFERENCES issues(id),   -- опц. привязка к задаче
    description  text NOT NULL DEFAULT '',
    started_at   timestamptz NOT NULL,
    ended_at     timestamptz,                  -- NULL = таймер идёт
    created_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    deleted_at   timestamptz
);
CREATE INDEX time_entries_ws_user ON time_entries (workspace_id, user_id, started_at);
CREATE INDEX time_entries_issue ON time_entries (issue_id);
ALTER TABLE time_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY time_entries_tenant ON time_entries
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

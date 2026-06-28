CREATE TABLE views (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    name         text NOT NULL,
    data_source  text NOT NULL DEFAULT 'issue',
    scope_kind   text NOT NULL DEFAULT 'team',
    team_id      uuid REFERENCES teams(id),
    owner_id     uuid REFERENCES users(id),
    shared       boolean NOT NULL DEFAULT false,
    filter       jsonb NOT NULL DEFAULT '{}'::jsonb,
    grouping     jsonb,
    ordering     jsonb,
    layout       text NOT NULL DEFAULT 'list',
    is_default   boolean NOT NULL DEFAULT false,
    position     text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    archived_at  timestamptz,
    deleted_at   timestamptz
);
CREATE INDEX views_ws_team ON views (workspace_id, team_id);

ALTER TABLE views ENABLE ROW LEVEL SECURITY;
CREATE POLICY views_tenant ON views
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

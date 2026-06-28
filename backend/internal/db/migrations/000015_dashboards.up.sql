CREATE TABLE dashboards (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    name         text NOT NULL,
    scope_kind   text NOT NULL DEFAULT 'workspace',
    team_id      uuid REFERENCES teams(id),
    owner_id     uuid REFERENCES users(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    archived_at  timestamptz,
    deleted_at   timestamptz
);
CREATE INDEX dashboards_ws ON dashboards (workspace_id);
ALTER TABLE dashboards ENABLE ROW LEVEL SECURITY;
CREATE POLICY dashboards_tenant ON dashboards
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE dashboard_widgets (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    dashboard_id uuid NOT NULL REFERENCES dashboards(id),
    type         text NOT NULL CHECK (type IN ('view','stat','chart','note')),
    title        text,
    x            int NOT NULL DEFAULT 0,
    y            int NOT NULL DEFAULT 0,
    w            int NOT NULL DEFAULT 1,
    h            int NOT NULL DEFAULT 1,
    config       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0
);
CREATE INDEX dashboard_widgets_dash ON dashboard_widgets (dashboard_id);
ALTER TABLE dashboard_widgets ENABLE ROW LEVEL SECURITY;
CREATE POLICY dashboard_widgets_tenant ON dashboard_widgets
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

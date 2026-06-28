CREATE TABLE projects (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    lead_id      uuid REFERENCES users(id),
    name         text NOT NULL,
    description  jsonb NOT NULL DEFAULT '{}'::jsonb,
    status       text NOT NULL DEFAULT 'planned'
                   CHECK (status IN ('backlog','planned','in_progress','paused','completed','canceled')),
    health       text CHECK (health IN ('on_track','at_risk','off_track')),
    start_date   date,
    target_date  date,
    color_token  text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    archived_at  timestamptz,
    deleted_at   timestamptz
);
CREATE INDEX projects_ws ON projects (workspace_id);
ALTER TABLE projects ENABLE ROW LEVEL SECURITY;
CREATE POLICY projects_tenant ON projects
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE cycles (
    id            uuid PRIMARY KEY,
    workspace_id  uuid NOT NULL REFERENCES workspaces(id),
    team_id       uuid NOT NULL REFERENCES teams(id),
    number        int NOT NULL,
    name          text,
    starts_on     date NOT NULL,
    ends_on       date NOT NULL,
    cooldown_days int NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    version       bigint NOT NULL DEFAULT 0,
    UNIQUE (team_id, number)
);
CREATE INDEX cycles_team ON cycles (team_id);
ALTER TABLE cycles ENABLE ROW LEVEL SECURITY;
CREATE POLICY cycles_tenant ON cycles
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

-- FK на issues (колонки добавлены в M1 без FK).
ALTER TABLE issues ADD CONSTRAINT issues_project_fk FOREIGN KEY (project_id) REFERENCES projects(id);
ALTER TABLE issues ADD CONSTRAINT issues_cycle_fk   FOREIGN KEY (cycle_id)   REFERENCES cycles(id);

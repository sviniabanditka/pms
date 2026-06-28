CREATE TABLE teams (
    id                uuid PRIMARY KEY,
    workspace_id      uuid NOT NULL REFERENCES workspaces(id),
    parent_id         uuid REFERENCES teams(id),
    path              ltree NOT NULL,
    key               text NOT NULL,
    name              text NOT NULL,
    private           boolean NOT NULL DEFAULT false,
    next_issue_number bigint NOT NULL DEFAULT 0,
    enabled_features  jsonb NOT NULL DEFAULT '{}'::jsonb,
    cycle_config      jsonb NOT NULL DEFAULT '{}'::jsonb,
    settings          jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    version           bigint NOT NULL DEFAULT 0,
    archived_at       timestamptz,
    deleted_at        timestamptz,
    UNIQUE (workspace_id, key),
    CONSTRAINT teams_key_fmt CHECK (key ~ '^[A-Z][A-Z0-9]{0,9}$'),
    CONSTRAINT teams_no_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE INDEX teams_parent ON teams (parent_id);
CREATE INDEX teams_path_gist ON teams USING gist (path);
CREATE INDEX teams_ws ON teams (workspace_id);

ALTER TABLE teams ENABLE ROW LEVEL SECURITY;
CREATE POLICY teams_tenant ON teams
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE team_members (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid NOT NULL REFERENCES teams(id),
    user_id      uuid NOT NULL REFERENCES users(id),
    role         text NOT NULL CHECK (role IN ('lead','member')),
    source       text NOT NULL DEFAULT 'direct' CHECK (source IN ('direct','inherited')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    UNIQUE (team_id, user_id)
);
CREATE INDEX team_members_user ON team_members (workspace_id, user_id);

ALTER TABLE team_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY team_members_tenant ON team_members
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

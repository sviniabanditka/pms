CREATE TABLE workflow_states (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid NOT NULL REFERENCES teams(id),
    category     text NOT NULL CHECK (category IN
                   ('backlog','unstarted','started','completed','canceled')),
    name         text NOT NULL,
    color_token  text NOT NULL,
    position     text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0
);
CREATE INDEX workflow_states_team ON workflow_states (team_id);

ALTER TABLE workflow_states ENABLE ROW LEVEL SECURITY;
CREATE POLICY workflow_states_tenant ON workflow_states
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE labels (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid REFERENCES teams(id),
    group_id     uuid REFERENCES labels(id),
    is_group     boolean NOT NULL DEFAULT false,
    name         text NOT NULL,
    color_token  text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0
);
CREATE INDEX labels_scope ON labels (workspace_id, team_id);

ALTER TABLE labels ENABLE ROW LEVEL SECURITY;
CREATE POLICY labels_tenant ON labels
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

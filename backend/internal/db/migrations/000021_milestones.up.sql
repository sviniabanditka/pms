CREATE TABLE milestones (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid NOT NULL REFERENCES teams(id),
    name         text NOT NULL,
    target_date  date,
    status       text NOT NULL DEFAULT 'planned', -- planned | active | done
    position     text NOT NULL DEFAULT 'n',
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    deleted_at   timestamptz
);
CREATE INDEX milestones_team ON milestones (workspace_id, team_id);
ALTER TABLE milestones ENABLE ROW LEVEL SECURITY;
CREATE POLICY milestones_tenant ON milestones USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

ALTER TABLE issues ADD COLUMN milestone_id uuid REFERENCES milestones(id);
CREATE INDEX issues_milestone ON issues (milestone_id);

CREATE TABLE issues (
    id             uuid PRIMARY KEY,
    workspace_id   uuid NOT NULL REFERENCES workspaces(id),
    team_id        uuid NOT NULL REFERENCES teams(id),
    number         bigint NOT NULL,
    identifier     text NOT NULL,                 -- team.key || '-' || number (team.key иммутабелен)
    parent_id      uuid REFERENCES issues(id),
    title          text NOT NULL,
    description    jsonb NOT NULL DEFAULT '{}'::jsonb,
    state_id       uuid NOT NULL REFERENCES workflow_states(id),
    priority       smallint NOT NULL DEFAULT 0,   -- 0 none .. 4 urgent
    assignee_id    uuid REFERENCES users(id),
    estimate       numeric,
    project_id     uuid,                          -- FK добавится в M4 (projects)
    cycle_id       uuid,                          -- FK добавится в M4 (cycles)
    due_date       date,
    start_date     date,
    scheduled_date date,
    started_at     timestamptz,                   -- авто по категории статуса
    completed_at   timestamptz,                   -- авто по категории статуса
    board_position text,                          -- fractional (колонка board)
    sub_position   text,                          -- fractional (среди siblings)
    created_by     uuid NOT NULL REFERENCES users(id),
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    version        bigint NOT NULL DEFAULT 0,
    archived_at    timestamptz,
    deleted_at     timestamptz,
    UNIQUE (team_id, number)
);
CREATE INDEX issues_ws_team ON issues (workspace_id, team_id);
CREATE INDEX issues_team_state ON issues (team_id, state_id);
CREATE INDEX issues_assignee ON issues (assignee_id);
CREATE INDEX issues_parent ON issues (parent_id);

ALTER TABLE issues ENABLE ROW LEVEL SECURITY;
CREATE POLICY issues_tenant ON issues
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE issue_labels (
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    issue_id     uuid NOT NULL REFERENCES issues(id),
    label_id     uuid NOT NULL REFERENCES labels(id),
    PRIMARY KEY (issue_id, label_id)
);
CREATE INDEX issue_labels_label ON issue_labels (label_id);

ALTER TABLE issue_labels ENABLE ROW LEVEL SECURITY;
CREATE POLICY issue_labels_tenant ON issue_labels
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

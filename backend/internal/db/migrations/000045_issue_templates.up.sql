-- Issue templates (переиспользуемый blueprint issue) + recurring (recurrence<>none → воркер плодит).
CREATE TABLE issue_templates (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    team_id         uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    name            text NOT NULL,
    title           text NOT NULL DEFAULT '',
    description     jsonb NOT NULL DEFAULT '{}'::jsonb,
    priority        smallint NOT NULL DEFAULT 0,
    assignee_id     uuid REFERENCES users(id) ON DELETE SET NULL,
    estimate        double precision,
    label_ids       jsonb NOT NULL DEFAULT '[]'::jsonb,
    recurrence      text NOT NULL DEFAULT 'none',   -- none|daily|weekly|monthly
    next_run_at     timestamptz,                    -- когда recurrence<>none
    last_spawned_at timestamptz,
    created_by      uuid NOT NULL REFERENCES users(id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    deleted_at      timestamptz
);

CREATE INDEX idx_issue_templates_ws ON issue_templates (workspace_id) WHERE deleted_at IS NULL;
-- частичный индекс под скан воркера (кросс-тенант): только активные recurring
CREATE INDEX idx_issue_templates_due ON issue_templates (next_run_at)
    WHERE recurrence <> 'none' AND deleted_at IS NULL AND next_run_at IS NOT NULL;

ALTER TABLE issue_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY issue_templates_ws ON issue_templates
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

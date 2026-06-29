CREATE TABLE automations (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid REFERENCES teams(id),   -- NULL = все команды воркспейса
    name         text NOT NULL DEFAULT '',
    enabled      boolean NOT NULL DEFAULT true,
    trigger      jsonb NOT NULL DEFAULT '{}'::jsonb,  -- {event, toStateId, ...}
    conditions   jsonb,                                -- Filter AST (опц.)
    actions      jsonb NOT NULL DEFAULT '[]'::jsonb,   -- [{type, ...}]
    created_by   uuid NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    deleted_at   timestamptz
);
CREATE INDEX automations_ws ON automations (workspace_id, enabled);
ALTER TABLE automations ENABLE ROW LEVEL SECURITY;
CREATE POLICY automations_tenant ON automations
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

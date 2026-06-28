CREATE TABLE custom_fields (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    team_id      uuid REFERENCES teams(id),        -- NULL = workspace-scope
    entity_type  text NOT NULL DEFAULT 'issue',
    key          text NOT NULL,                    -- стабильный ключ для Filter AST
    name         text NOT NULL,
    type         text NOT NULL CHECK (type IN
                   ('text','number','date','checkbox','select','multiselect','url')),
    config       jsonb NOT NULL DEFAULT '{}'::jsonb, -- напр. {"options":["A","B"]}
    position     text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    archived_at  timestamptz
);
CREATE INDEX custom_fields_scope ON custom_fields (workspace_id, team_id);
ALTER TABLE custom_fields ENABLE ROW LEVEL SECURITY;
CREATE POLICY custom_fields_tenant ON custom_fields
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE custom_field_values (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    field_id     uuid NOT NULL REFERENCES custom_fields(id),
    issue_id     uuid NOT NULL REFERENCES issues(id),
    value        jsonb,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    UNIQUE (field_id, issue_id)
);
CREATE INDEX custom_field_values_issue ON custom_field_values (issue_id);
CREATE INDEX custom_field_values_gin ON custom_field_values USING gin (value);
ALTER TABLE custom_field_values ENABLE ROW LEVEL SECURITY;
CREATE POLICY custom_field_values_tenant ON custom_field_values
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

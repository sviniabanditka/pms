CREATE TABLE issue_relations (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    source_id    uuid NOT NULL REFERENCES issues(id),
    target_id    uuid NOT NULL REFERENCES issues(id),
    type         text NOT NULL CHECK (type IN ('blocks','blocked_by','related','duplicate','duplicate_of')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    UNIQUE (source_id, target_id, type),
    CONSTRAINT issue_relations_no_self CHECK (source_id <> target_id)
);
CREATE INDEX issue_relations_source ON issue_relations (source_id);
CREATE INDEX issue_relations_target ON issue_relations (target_id);

ALTER TABLE issue_relations ENABLE ROW LEVEL SECURITY;
CREATE POLICY issue_relations_tenant ON issue_relations
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

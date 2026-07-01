-- AI Фаза 4: топик-кластеры issue (HDBSCAN на эмбеддингах). Пересчёт воркером.
CREATE TABLE issue_clusters (
    workspace_id uuid        NOT NULL,
    issue_id     uuid        PRIMARY KEY,
    cluster      int         NOT NULL,   -- id кластера (-1 = шум)
    updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX issue_clusters_ws ON issue_clusters (workspace_id, cluster);
ALTER TABLE issue_clusters ENABLE ROW LEVEL SECURITY;
CREATE POLICY issue_clusters_tenant ON issue_clusters
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

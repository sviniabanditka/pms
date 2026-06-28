CREATE TABLE channel_reads (
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    channel_id   uuid NOT NULL,
    user_id      uuid NOT NULL,
    last_read_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (channel_id, user_id)
);
ALTER TABLE channel_reads ENABLE ROW LEVEL SECURITY;
CREATE POLICY channel_reads_tenant ON channel_reads USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

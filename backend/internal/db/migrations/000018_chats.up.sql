CREATE TABLE channels (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    kind         text NOT NULL DEFAULT 'channel', -- channel | dm
    name         text NOT NULL DEFAULT '',
    created_by   uuid NOT NULL REFERENCES users(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0,
    archived_at  timestamptz,
    deleted_at   timestamptz
);
CREATE INDEX channels_ws ON channels (workspace_id);
ALTER TABLE channels ENABLE ROW LEVEL SECURITY;
CREATE POLICY channels_tenant ON channels USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE channel_members (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    channel_id   uuid NOT NULL REFERENCES channels(id),
    user_id      uuid NOT NULL REFERENCES users(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (channel_id, user_id)
);
CREATE INDEX channel_members_ch ON channel_members (channel_id);
CREATE INDEX channel_members_user ON channel_members (user_id);
ALTER TABLE channel_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY channel_members_tenant ON channel_members USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE messages (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    channel_id   uuid NOT NULL REFERENCES channels(id),
    author_id    uuid NOT NULL REFERENCES users(id),
    body         text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    edited_at    timestamptz,
    deleted_at   timestamptz
);
CREATE INDEX messages_channel ON messages (channel_id, created_at);
ALTER TABLE messages ENABLE ROW LEVEL SECURITY;
CREATE POLICY messages_tenant ON messages USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

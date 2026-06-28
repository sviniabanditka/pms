ALTER TABLE messages ADD COLUMN parent_id uuid REFERENCES messages(id);
CREATE INDEX messages_parent ON messages (parent_id);

CREATE TABLE message_reactions (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    message_id   uuid NOT NULL REFERENCES messages(id),
    user_id      uuid NOT NULL REFERENCES users(id),
    emoji        text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (message_id, user_id, emoji)
);
CREATE INDEX message_reactions_msg ON message_reactions (message_id);
ALTER TABLE message_reactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY message_reactions_tenant ON message_reactions USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

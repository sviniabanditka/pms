-- D42: реакции на комментарии задач (паритет с реакциями чата).
CREATE TABLE comment_reactions (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    comment_id   uuid NOT NULL REFERENCES comments(id),
    user_id      uuid NOT NULL REFERENCES users(id),
    emoji        text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (comment_id, user_id, emoji)
);
CREATE INDEX comment_reactions_comment ON comment_reactions (comment_id);
ALTER TABLE comment_reactions ENABLE ROW LEVEL SECURITY;
CREATE POLICY comment_reactions_tenant ON comment_reactions USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

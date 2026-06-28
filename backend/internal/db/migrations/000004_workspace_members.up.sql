CREATE TABLE workspace_members (
    id              uuid PRIMARY KEY,
    workspace_id    uuid NOT NULL REFERENCES workspaces(id),
    user_id         uuid NOT NULL REFERENCES users(id),
    role            text NOT NULL CHECK (role IN ('owner','admin','member','guest')),
    status          text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended')),
    sections_config jsonb NOT NULL DEFAULT '{}'::jsonb,
    joined_at       timestamptz NOT NULL DEFAULT now(),
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    version         bigint NOT NULL DEFAULT 0,
    UNIQUE (workspace_id, user_id)
);
CREATE INDEX workspace_members_user ON workspace_members (user_id);

ALTER TABLE workspace_members ENABLE ROW LEVEL SECURITY;
CREATE POLICY workspace_members_tenant ON workspace_members
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE invitations (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    email        citext NOT NULL,
    role         text NOT NULL CHECK (role IN ('owner','admin','member','guest')),
    token_hash   text NOT NULL UNIQUE,
    invited_by   uuid NOT NULL REFERENCES users(id),
    expires_at   timestamptz NOT NULL,
    accepted_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    version      bigint NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX invitations_active ON invitations (workspace_id, email)
    WHERE accepted_at IS NULL;

ALTER TABLE invitations ENABLE ROW LEVEL SECURITY;
CREATE POLICY invitations_tenant ON invitations
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

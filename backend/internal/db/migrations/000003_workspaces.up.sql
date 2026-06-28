CREATE TABLE workspaces (
    id          uuid PRIMARY KEY,
    slug        citext NOT NULL UNIQUE,
    name        text NOT NULL,
    logo_url    text,
    settings    jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_by  uuid NOT NULL REFERENCES users(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    version     bigint NOT NULL DEFAULT 0,
    archived_at timestamptz,
    deleted_at  timestamptz,
    CONSTRAINT workspaces_slug_fmt CHECK (slug ~ '^[a-z0-9-]{2,40}$')
);

ALTER TABLE workspaces ENABLE ROW LEVEL SECURITY;
CREATE POLICY workspaces_tenant ON workspaces
    USING (id = current_setting('app.workspace_id', true)::uuid);

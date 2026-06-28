CREATE TABLE meetings (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    channel_id   uuid REFERENCES channels(id),  -- huddle/scheduled, привязанный к каналу
    kind         text NOT NULL DEFAULT 'scheduled', -- scheduled | huddle
    title        text NOT NULL DEFAULT '',
    status       text NOT NULL DEFAULT 'scheduled', -- scheduled | live | ended
    scheduled_at timestamptz,
    created_by   uuid NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    started_at   timestamptz,
    ended_at     timestamptz,
    version      bigint NOT NULL DEFAULT 0,
    deleted_at   timestamptz
);
CREATE INDEX meetings_ws ON meetings (workspace_id, status);
CREATE INDEX meetings_channel ON meetings (channel_id);
ALTER TABLE meetings ENABLE ROW LEVEL SECURITY;
CREATE POLICY meetings_tenant ON meetings USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

CREATE TABLE meeting_participants (
    id           uuid PRIMARY KEY,
    workspace_id uuid NOT NULL REFERENCES workspaces(id),
    meeting_id   uuid NOT NULL REFERENCES meetings(id),
    user_id      uuid NOT NULL,
    role         text NOT NULL DEFAULT 'participant', -- host | participant
    state        text NOT NULL DEFAULT 'invited',     -- invited | joined | left
    joined_at    timestamptz,
    left_at      timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (meeting_id, user_id)
);
CREATE INDEX meeting_participants_meeting ON meeting_participants (workspace_id, meeting_id);
ALTER TABLE meeting_participants ENABLE ROW LEVEL SECURITY;
CREATE POLICY meeting_participants_tenant ON meeting_participants USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

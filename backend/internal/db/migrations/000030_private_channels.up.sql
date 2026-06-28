-- Приватные каналы (Slack-like): видны/доступны только участникам.
ALTER TABLE channels ADD COLUMN private boolean NOT NULL DEFAULT false;

ALTER TABLE users ADD COLUMN ical_token text;
CREATE UNIQUE INDEX users_ical_token ON users (ical_token) WHERE ical_token IS NOT NULL;

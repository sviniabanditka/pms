DROP INDEX IF EXISTS users_ical_token;
ALTER TABLE users DROP COLUMN IF EXISTS ical_token;

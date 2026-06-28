DROP INDEX IF EXISTS notifications_unemailed;
ALTER TABLE notifications DROP COLUMN IF EXISTS emailed_at;

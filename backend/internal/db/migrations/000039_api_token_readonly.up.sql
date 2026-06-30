-- Узкий scope: read-only токен может звать только read-RPC (как guest).
ALTER TABLE api_tokens ADD COLUMN read_only boolean NOT NULL DEFAULT false;

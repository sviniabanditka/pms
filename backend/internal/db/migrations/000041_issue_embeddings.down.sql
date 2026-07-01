DROP INDEX IF EXISTS issues_embedding_hnsw;
ALTER TABLE issues DROP COLUMN IF EXISTS embedded_at;
ALTER TABLE issues DROP COLUMN IF EXISTS embedding;

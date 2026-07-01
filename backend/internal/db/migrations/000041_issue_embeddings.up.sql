-- AI Фаза 1: эмбеддинги issue (semantic search + dup-detection).
-- pgvector; вектор + отметка когда посчитан (воркер сравнивает с updated_at).
CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE issues ADD COLUMN embedding vector(384);
ALTER TABLE issues ADD COLUMN embedded_at timestamptz;

-- HNSW cosine; изоляция по воркспейсу — через RLS/WHERE (векторы на RLS-таблице issues).
CREATE INDEX issues_embedding_hnsw ON issues USING hnsw (embedding vector_cosine_ops);

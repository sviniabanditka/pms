ALTER TABLE issues ADD COLUMN embedding vector(384);
ALTER TABLE issues ADD COLUMN embedded_at timestamptz;
CREATE INDEX issues_embedding_hnsw ON issues USING hnsw (embedding vector_cosine_ops);
DROP TABLE IF EXISTS embeddings;

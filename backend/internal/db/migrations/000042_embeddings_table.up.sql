-- AI Фаза 2: единая таблица эмбеддингов (issue/doc/comment/…) для cross-entity
-- семантического поиска. Одна HNSW, RLS по воркспейсу. Переносим issue-эмбеддинги
-- из колонки в таблицу.
CREATE TABLE embeddings (
    workspace_id uuid        NOT NULL,
    entity_type  text        NOT NULL,   -- issue | doc | comment
    entity_id    uuid        NOT NULL,
    embedding    vector(384) NOT NULL,
    embedded_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (entity_type, entity_id)
);
CREATE INDEX embeddings_ws ON embeddings (workspace_id);
CREATE INDEX embeddings_hnsw ON embeddings USING hnsw (embedding vector_cosine_ops);

ALTER TABLE embeddings ENABLE ROW LEVEL SECURITY;
CREATE POLICY embeddings_tenant ON embeddings
    USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

-- issue-эмбеддинги теперь в таблице (воркер перезальёт)
DROP INDEX IF EXISTS issues_embedding_hnsw;
ALTER TABLE issues DROP COLUMN IF EXISTS embedding;
ALTER TABLE issues DROP COLUMN IF EXISTS embedded_at;

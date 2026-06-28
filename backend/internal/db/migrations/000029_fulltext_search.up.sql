-- D21+: Postgres FTS — поиск по ТЕЛУ задач/доков + ранжирование + опечатки (pg_trgm).
-- Конфиг 'simple' (без стемминга) — язык-агностично (RU+EN контент); fuzzy даёт trgm.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- issues: identifier + title + description.md
ALTER TABLE issues ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    to_tsvector('simple',
      coalesce(identifier,'') || ' ' || coalesce(title,'') || ' ' || coalesce(description->>'md',''))
  ) STORED;
CREATE INDEX issues_search_gin ON issues USING gin (search_vector);
CREATE INDEX issues_title_trgm ON issues USING gin (title gin_trgm_ops);

-- documents: title + body.md (раньше искались только заголовки)
ALTER TABLE documents ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (
    to_tsvector('simple', coalesce(title,'') || ' ' || coalesce(body->>'md',''))
  ) STORED;
CREATE INDEX documents_search_gin ON documents USING gin (search_vector);
CREATE INDEX documents_title_trgm ON documents USING gin (title gin_trgm_ops);

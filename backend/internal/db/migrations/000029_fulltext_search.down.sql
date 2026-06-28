DROP INDEX IF EXISTS documents_title_trgm;
DROP INDEX IF EXISTS documents_search_gin;
ALTER TABLE documents DROP COLUMN IF EXISTS search_vector;
DROP INDEX IF EXISTS issues_title_trgm;
DROP INDEX IF EXISTS issues_search_gin;
ALTER TABLE issues DROP COLUMN IF EXISTS search_vector;
-- pg_trgm оставляем (может использоваться ещё).

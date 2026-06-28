-- Неявный каст text -> ltree: позволяет вставлять path как обычную строку (bun-параметр),
-- без ручного ?::ltree в каждом запросе.
CREATE CAST (text AS ltree) WITH INOUT AS IMPLICIT;

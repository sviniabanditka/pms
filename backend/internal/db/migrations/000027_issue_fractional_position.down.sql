-- Откат: строки -> double precision (порядок best-effort, фракционные ключи в число не парсятся).
ALTER TABLE issues ALTER COLUMN board_position TYPE double precision USING 0;
ALTER TABLE issues ALTER COLUMN sub_position   TYPE double precision USING 0;
ALTER TABLE issues ALTER COLUMN board_position SET DEFAULT 0;
ALTER TABLE issues ALTER COLUMN sub_position   SET DEFAULT 0;

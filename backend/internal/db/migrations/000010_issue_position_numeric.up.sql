-- board_position/sub_position: text -> double precision (порядок через midpoint между соседями).
-- Существующие строки: board_position = number (сохраняет порядок создания), sub_position = 0.
ALTER TABLE issues ALTER COLUMN board_position TYPE double precision USING (number)::double precision;
ALTER TABLE issues ALTER COLUMN sub_position   TYPE double precision USING 0;
ALTER TABLE issues ALTER COLUMN board_position SET DEFAULT 0;
ALTER TABLE issues ALTER COLUMN sub_position   SET DEFAULT 0;

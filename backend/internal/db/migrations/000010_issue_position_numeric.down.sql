ALTER TABLE issues ALTER COLUMN board_position TYPE text USING board_position::text;
ALTER TABLE issues ALTER COLUMN sub_position   TYPE text USING sub_position::text;
ALTER TABLE issues ALTER COLUMN board_position DROP DEFAULT;
ALTER TABLE issues ALTER COLUMN sub_position   DROP DEFAULT;

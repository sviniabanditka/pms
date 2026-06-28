-- D20: board_position/sub_position float -> фракционные строки (base36, нет исчерпания точности).
ALTER TABLE issues ALTER COLUMN board_position DROP DEFAULT;
ALTER TABLE issues ALTER COLUMN sub_position   DROP DEFAULT;
ALTER TABLE issues ALTER COLUMN board_position TYPE text USING board_position::text;
ALTER TABLE issues ALTER COLUMN sub_position   TYPE text USING 'i';

-- Переписать board_position на zero-padded порядковый внутри (team,state):
-- сортируется как текст в том же порядке, что был, и совместим с frac.Between().
WITH ord AS (
  SELECT id,
         lpad((row_number() OVER (PARTITION BY team_id, state_id
                                  ORDER BY board_position::double precision, created_at))::text,
              8, '0') AS pos
  FROM issues
)
UPDATE issues i SET board_position = ord.pos FROM ord WHERE ord.id = i.id;

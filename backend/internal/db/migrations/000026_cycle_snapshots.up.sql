-- Ежедневный слепок «сколько осталось» в цикле → строится burndown.
CREATE TABLE cycle_snapshots (
    workspace_id     uuid NOT NULL REFERENCES workspaces(id),
    cycle_id         uuid NOT NULL REFERENCES cycles(id),
    day              text NOT NULL, -- YYYY-MM-DD (строкой: ровный JSON-роундтрип, корректная сортировка)
    remaining_points double precision NOT NULL DEFAULT 0,
    remaining_count  int NOT NULL DEFAULT 0,
    total_points     double precision NOT NULL DEFAULT 0,
    total_count      int NOT NULL DEFAULT 0,
    PRIMARY KEY (cycle_id, day)
);
ALTER TABLE cycle_snapshots ENABLE ROW LEVEL SECURITY;
CREATE POLICY cycle_snapshots_tenant ON cycle_snapshots USING (workspace_id = current_setting('app.workspace_id', true)::uuid);

package jobs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"
)

type SnapshotArgs struct{}

func (SnapshotArgs) Kind() string { return "cycle_snapshot" }

// CycleSnapshotWorker — ежедневный слепок остатка по активным циклам (для burndown).
type CycleSnapshotWorker struct {
	river.WorkerDefaults[SnapshotArgs]
	db *bun.DB
}

func (w *CycleSnapshotWorker) Work(ctx context.Context, _ *river.Job[SnapshotArgs]) error {
	day := time.Now().UTC().Format("2006-01-02")
	// активные циклы = сегодня в [starts_on, ends_on]
	var cycles []struct {
		ID          uuid.UUID `bun:"id"`
		WorkspaceID uuid.UUID `bun:"workspace_id"`
	}
	if err := w.db.NewSelect().TableExpr("cycles AS cy").
		ColumnExpr("cy.id, cy.workspace_id").
		Where("CURRENT_DATE BETWEEN cy.starts_on AND cy.ends_on").Scan(ctx, &cycles); err != nil {
		return err
	}
	for _, c := range cycles {
		var agg struct {
			RemainingPoints float64 `bun:"rem_pts"`
			RemainingCount  int     `bun:"rem_cnt"`
			TotalPoints     float64 `bun:"tot_pts"`
			TotalCount      int     `bun:"tot_cnt"`
		}
		// остаток = задачи цикла, чей статус НЕ из категории completed/canceled; total = все живые
		err := w.db.NewSelect().TableExpr("issues AS i").
			Join("LEFT JOIN workflow_states s ON s.id = i.state_id").
			ColumnExpr("COALESCE(SUM(i.estimate) FILTER (WHERE s.category NOT IN ('completed','canceled')), 0) AS rem_pts").
			ColumnExpr("COUNT(*) FILTER (WHERE s.category NOT IN ('completed','canceled')) AS rem_cnt").
			ColumnExpr("COALESCE(SUM(i.estimate), 0) AS tot_pts").
			ColumnExpr("COUNT(*) AS tot_cnt").
			Where("i.cycle_id = ?", c.ID).Where("i.archived_at IS NULL").Where("i.deleted_at IS NULL").
			Scan(ctx, &agg)
		if err != nil {
			return err
		}
		_, err = w.db.NewInsert().Model(&struct {
			bun.BaseModel   `bun:"table:cycle_snapshots"`
			WorkspaceID     uuid.UUID `bun:"workspace_id"`
			CycleID         uuid.UUID `bun:"cycle_id"`
			Day             string    `bun:"day"`
			RemainingPoints float64   `bun:"remaining_points"`
			RemainingCount  int       `bun:"remaining_count"`
			TotalPoints     float64   `bun:"total_points"`
			TotalCount      int       `bun:"total_count"`
		}{
			WorkspaceID: c.WorkspaceID, CycleID: c.ID, Day: day,
			RemainingPoints: agg.RemainingPoints, RemainingCount: agg.RemainingCount,
			TotalPoints: agg.TotalPoints, TotalCount: agg.TotalCount,
		}).On("CONFLICT (cycle_id, day) DO UPDATE").
			Set("remaining_points = EXCLUDED.remaining_points").Set("remaining_count = EXCLUDED.remaining_count").
			Set("total_points = EXCLUDED.total_points").Set("total_count = EXCLUDED.total_count").Exec(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

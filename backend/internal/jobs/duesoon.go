package jobs

import (
	"context"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/domain/notifications"
	"github.com/pms/backend/internal/realtime"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"
)

type DueSoonArgs struct{}

func (DueSoonArgs) Kind() string { return "due_soon_scan" }

// DueSoonWorker — периодически ищет задачи со сроком в пределах суток и шлёт in-app due_soon.
type DueSoonWorker struct {
	river.WorkerDefaults[DueSoonArgs]
	db  *bun.DB
	pub *realtime.Publisher
}

func (w *DueSoonWorker) Work(ctx context.Context, _ *river.Job[DueSoonArgs]) error {
	var rows []struct {
		ID          uuid.UUID `bun:"id"`
		WorkspaceID uuid.UUID `bun:"workspace_id"`
		AssigneeID  uuid.UUID `bun:"assignee_id"`
	}
	err := w.db.NewSelect().TableExpr("issues AS i").
		ColumnExpr("i.id, i.workspace_id, i.assignee_id").
		Where("i.assignee_id IS NOT NULL").
		Where("i.completed_at IS NULL").Where("i.archived_at IS NULL").Where("i.deleted_at IS NULL").
		Where("COALESCE(i.scheduled_date, i.due_date) IS NOT NULL").
		Where("COALESCE(i.scheduled_date, i.due_date) BETWEEN CURRENT_DATE AND CURRENT_DATE + 1").
		Where("NOT EXISTS (SELECT 1 FROM notifications n WHERE n.entity_id = i.id AND n.user_id = i.assignee_id AND n.type = 'due_soon' AND n.at > now() - interval '20 hours')").
		Scan(ctx, &rows)
	if err != nil {
		return err
	}
	for _, r := range rows {
		// actor=Nil (системное), recipient=assignee. jobsDB BYPASSRLS → вставка кросс-тенантно.
		if err := notifications.Record(ctx, w.db, r.WorkspaceID, r.AssigneeID, uuid.Nil, "due_soon", "issue", r.ID, nil); err != nil {
			return err
		}
		_ = w.pub.PublishDelta(ctx, r.WorkspaceID, realtime.Delta{EntityType: "notification", EntityID: r.ID.String(), Op: "create"})
	}
	return nil
}

package jobs

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"
)

type NotifySweepArgs struct{}

func (NotifySweepArgs) Kind() string { return "notify_email_sweep" }

// NotifySweepWorker — берёт неотправленные in-app уведомления и ставит email-задачи (любой тип).
type NotifySweepWorker struct {
	river.WorkerDefaults[NotifySweepArgs]
	db *bun.DB
}

var subjects = map[string]string{
	"mention":       "You were mentioned in PMS",
	"assigned":      "A task was assigned to you",
	"state_changed": "A task status changed",
	"commented":     "New comment on your task",
	"due_soon":      "Task due soon",
}

func (w *NotifySweepWorker) Work(ctx context.Context, _ *river.Job[NotifySweepArgs]) error {
	var rows []struct {
		ID    uuid.UUID `bun:"id"`
		Type  string    `bun:"type"`
		Email string    `bun:"email"`
	}
	err := w.db.NewSelect().TableExpr("notifications AS n").
		ColumnExpr("n.id, n.type, u.email").
		Join("JOIN users u ON u.id = n.user_id").
		Where("n.emailed_at IS NULL").
		Where("n.at > now() - interval '1 day'").
		OrderExpr("n.at").Limit(100).Scan(ctx, &rows)
	if err != nil {
		return err
	}
	client := river.ClientFromContext[pgx.Tx](ctx)
	var done []uuid.UUID
	for _, r := range rows {
		if r.Email == "" {
			done = append(done, r.ID) // некому слать — пометить, чтобы не зацикливаться
			continue
		}
		subj := subjects[r.Type]
		if subj == "" {
			subj = "PMS notification"
		}
		if _, err := client.Insert(ctx, EmailArgs{To: r.Email, Subject: subj, Body: "Open PMS to see details."}, nil); err != nil {
			continue // оставим неотмеченным → повтор на следующем sweep
		}
		done = append(done, r.ID)
	}
	if len(done) > 0 {
		if _, err := w.db.NewUpdate().Table("notifications").Set("emailed_at = now()").Where("id IN (?)", bun.In(done)).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

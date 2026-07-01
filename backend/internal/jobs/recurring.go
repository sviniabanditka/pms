package jobs

import (
	"context"

	"github.com/riverqueue/river"

	"github.com/pms/backend/internal/domain/templates"
)

// RecurringIssueWorker (issue-шаблоны): периодически плодит issue из due recurring-шаблонов.
// Кросс-тенантно — templates.Service поверх pms_jobs (BYPASSRLS).

type RecurringArgs struct{}

func (RecurringArgs) Kind() string { return "recurring_issues" }

type RecurringIssueWorker struct {
	river.WorkerDefaults[RecurringArgs]
	svc *templates.Service
}

func (w *RecurringIssueWorker) Work(ctx context.Context, _ *river.Job[RecurringArgs]) error {
	if w.svc == nil {
		return nil
	}
	_, err := w.svc.SpawnDue(ctx)
	return err
}

func NewRecurringIssueWorker(svc *templates.Service) *RecurringIssueWorker {
	return &RecurringIssueWorker{svc: svc}
}

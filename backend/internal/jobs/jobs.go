// Package jobs — фоновые задачи на River (Postgres-очередь): due-soon напоминания + email-доставка.
package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pms/backend/internal/mailer"
	"github.com/pms/backend/internal/realtime"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/uptrace/bun"
)

type Deps struct {
	Pool             *pgxpool.Pool
	JobsDB           *bun.DB // роль pms_jobs (BYPASSRLS) — кросс-тенантный скан
	Pub              *realtime.Publisher
	Mailer           mailer.Sender
	DueSoonInterval  river.PeriodicSchedule
	EmailSweepEvery  river.PeriodicSchedule
	SnapshotEvery    river.PeriodicSchedule
	MeetingReapEvery river.PeriodicSchedule
	LiveKitAPIURL    string
	LiveKitAPIKey    string
	LiveKitSecret    string
}

// Start прогоняет River-миграции, поднимает воркеры и периодические задачи.
func Start(ctx context.Context, d Deps) (*river.Client[pgx.Tx], error) {
	driver := riverpgxv5.New(d.Pool)
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		return nil, err
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, err
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &EmailWorker{mailer: d.Mailer})
	river.AddWorker(workers, &DueSoonWorker{db: d.JobsDB, pub: d.Pub})
	river.AddWorker(workers, &NotifySweepWorker{db: d.JobsDB})
	river.AddWorker(workers, &CycleSnapshotWorker{db: d.JobsDB})
	river.AddWorker(workers, &MeetingReaperWorker{db: d.JobsDB, pub: d.Pub, apiURL: d.LiveKitAPIURL, key: d.LiveKitAPIKey, secret: d.LiveKitSecret})

	client, err := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 10}},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(d.DueSoonInterval, func() (river.JobArgs, *river.InsertOpts) { return DueSoonArgs{}, nil }, &river.PeriodicJobOpts{RunOnStart: true}),
			river.NewPeriodicJob(d.EmailSweepEvery, func() (river.JobArgs, *river.InsertOpts) { return NotifySweepArgs{}, nil }, &river.PeriodicJobOpts{RunOnStart: true}),
			river.NewPeriodicJob(d.SnapshotEvery, func() (river.JobArgs, *river.InsertOpts) { return SnapshotArgs{}, nil }, &river.PeriodicJobOpts{RunOnStart: true}),
			river.NewPeriodicJob(d.MeetingReapEvery, func() (river.JobArgs, *river.InsertOpts) { return MeetingReapArgs{}, nil }, &river.PeriodicJobOpts{RunOnStart: true}),
		},
	})
	if err != nil {
		return nil, err
	}
	if err := client.Start(ctx); err != nil {
		return nil, err
	}
	return client, nil
}

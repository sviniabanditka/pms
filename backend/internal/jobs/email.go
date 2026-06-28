package jobs

import (
	"context"

	"github.com/pms/backend/internal/mailer"
	"github.com/riverqueue/river"
)

type EmailArgs struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func (EmailArgs) Kind() string { return "email" }

type EmailWorker struct {
	river.WorkerDefaults[EmailArgs]
	mailer mailer.Sender
}

func (w *EmailWorker) Work(ctx context.Context, job *river.Job[EmailArgs]) error {
	return w.mailer.Send(ctx, mailer.Mail{To: job.Args.To, Subject: job.Args.Subject, Body: job.Args.Body})
}

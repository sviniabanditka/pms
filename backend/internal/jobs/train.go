package jobs

import (
	"context"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/embed"
)

// Model-train (AI Фаза 3): per-workspace классификатор priority на эмбеддингах issue.
// Периодический; кросс-тенантно (pms_jobs BYPASSRLS). Сайдкар решает, хватает ли
// данных (иначе no-op). Модель живёт на томе сайдкара, ключ = workspace_id.

type TrainArgs struct{}

func (TrainArgs) Kind() string { return "model_train" }

type TrainWorker struct {
	river.WorkerDefaults[TrainArgs]
	db    *bun.DB
	embed *embed.Client
}

func (w *TrainWorker) Work(ctx context.Context, _ *river.Job[TrainArgs]) error {
	if w.embed == nil {
		return nil
	}
	var wss []uuid.UUID
	if err := w.db.NewSelect().Table("workspaces").Column("id").Where("deleted_at IS NULL").Scan(ctx, &wss); err != nil {
		return err
	}
	for _, ws := range wss {
		var rows []struct {
			Vec      string `bun:"vec"`
			Priority int    `bun:"priority"`
		}
		err := w.db.NewRaw(
			`SELECT e.embedding::text AS vec, i.priority
			 FROM issues i JOIN embeddings e ON e.entity_type='issue' AND e.entity_id = i.id
			 WHERE i.workspace_id = ? AND i.deleted_at IS NULL`, ws).Scan(ctx, &rows)
		if err != nil || len(rows) < 40 {
			continue
		}
		vecs := make([][]float32, 0, len(rows))
		labels := make([]int, 0, len(rows))
		for _, r := range rows {
			v := embed.ParseVecLiteral(r.Vec)
			if v == nil {
				continue
			}
			vecs = append(vecs, v)
			labels = append(labels, r.Priority)
		}
		if len(vecs) >= 40 {
			_ = w.embed.Train(ctx, ws.String(), "priority", vecs, labels) // сайдкар решает достаточность
		}
	}
	return nil
}

func NewTrainWorker(db *bun.DB, e *embed.Client) *TrainWorker {
	return &TrainWorker{db: db, embed: e}
}

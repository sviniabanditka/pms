package jobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/embed"
)

// Model-train (AI Фаза 3): per-workspace модели на эмбеддингах issue —
// priority (classification), estimate (regression), slip-risk (classification).
// Периодический; кросс-тенантно (pms_jobs BYPASSRLS). Сайдкар решает достаточность.

type TrainArgs struct{}

func (TrainArgs) Kind() string { return "model_train" }

type TrainWorker struct {
	river.WorkerDefaults[TrainArgs]
	db    *bun.DB
	embed *embed.Client
}

// таргет: kind + task + SQL-выражение метки + доп. фильтр (какие issue берём в обучение).
type trainTarget struct {
	kind      string
	task      string
	labelExpr string
	where     string
}

var trainTargets = []trainTarget{
	{"priority", "classification", "i.priority::float8", ""},
	{"estimate", "regression", "i.estimate", "AND i.estimate IS NOT NULL"},
	{"slip", "classification", "(CASE WHEN i.completed_at::date > i.due_date THEN 1 ELSE 0 END)::float8", "AND i.completed_at IS NOT NULL AND i.due_date IS NOT NULL"},
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
		for _, t := range trainTargets {
			w.trainTarget(ctx, ws, t)
		}
	}
	return nil
}

func (w *TrainWorker) trainTarget(ctx context.Context, ws uuid.UUID, t trainTarget) {
	// петля фидбэка: issue с applied-подсказкой этого типа весят больше
	var appliedIDs []uuid.UUID
	_ = w.db.NewRaw(`SELECT DISTINCT entity_id FROM ai_feedback WHERE workspace_id = ? AND kind = ? AND action = 'applied' AND entity_id IS NOT NULL`, ws, t.kind).Scan(ctx, &appliedIDs)
	applied := make(map[uuid.UUID]bool, len(appliedIDs))
	for _, id := range appliedIDs {
		applied[id] = true
	}

	var rows []struct {
		ID    uuid.UUID `bun:"id"`
		Vec   string    `bun:"vec"`
		Label float64   `bun:"label"`
	}
	q := fmt.Sprintf(
		`SELECT i.id, e.embedding::text AS vec, (%s) AS label
		 FROM issues i JOIN embeddings e ON e.entity_type='issue' AND e.entity_id = i.id
		 WHERE i.workspace_id = ? AND i.deleted_at IS NULL %s`, t.labelExpr, t.where)
	if err := w.db.NewRaw(q, ws).Scan(ctx, &rows); err != nil || len(rows) < 40 {
		return
	}
	vecs := make([][]float32, 0, len(rows))
	labels := make([]float64, 0, len(rows))
	weights := make([]float64, 0, len(rows))
	for _, r := range rows {
		v := embed.ParseVecLiteral(r.Vec)
		if v == nil {
			continue
		}
		vecs = append(vecs, v)
		labels = append(labels, r.Label)
		if applied[r.ID] {
			weights = append(weights, 3.0) // подтверждённый пользователем сигнал
		} else {
			weights = append(weights, 1.0)
		}
	}
	if len(vecs) >= 40 {
		_ = w.embed.Train(ctx, ws.String(), t.kind, t.task, vecs, labels, weights)
	}
}

func NewTrainWorker(db *bun.DB, e *embed.Client) *TrainWorker {
	return &TrainWorker{db: db, embed: e}
}

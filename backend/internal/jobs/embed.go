package jobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/embed"
)

// Embed-sweep (AI Фаза 1–2): считает эмбеддинги для issue/doc/comment без вектора
// или со stale (embedded_at < changed_at) → единая таблица embeddings. Кросс-тенантно
// (pms_jobs BYPASSRLS). Каждый тик — маленький батч каждого типа (отдельный /embed
// запрос ≤32 → без OOM-пиков).

type EmbedSweepArgs struct{}

func (EmbedSweepArgs) Kind() string { return "embed_sweep" }

type EmbedSweepWorker struct {
	river.WorkerDefaults[EmbedSweepArgs]
	db    *bun.DB
	embed *embed.Client
}

// цель свипа: тип + SQL-текст (jsonb md извлекаем в SQL) + выражение «когда изменено».
type embedTarget struct {
	entityType string
	table      string
	textSQL    string
	changedSQL string
}

var embedTargets = []embedTarget{
	{"issue", "issues", "t.title || ' ' || COALESCE(t.description->>'md','')", "t.updated_at"},
	{"doc", "documents", "COALESCE(t.title,'') || ' ' || COALESCE(t.body->>'md','')", "t.updated_at"},
	{"comment", "comments", "t.body", "COALESCE(t.edited_at, t.created_at)"},
}

func (w *EmbedSweepWorker) Work(ctx context.Context, _ *river.Job[EmbedSweepArgs]) error {
	if w.embed == nil {
		return nil
	}
	for _, tgt := range embedTargets {
		_ = w.sweep(ctx, tgt) // не роняем весь свип из-за одного типа
	}
	return nil
}

func (w *EmbedSweepWorker) sweep(ctx context.Context, t embedTarget) error {
	var rows []struct {
		ID          uuid.UUID `bun:"id"`
		WorkspaceID uuid.UUID `bun:"workspace_id"`
		Text        string    `bun:"text"`
	}
	q := fmt.Sprintf(`
		SELECT t.id, t.workspace_id, (%s) AS text
		FROM %s t
		LEFT JOIN embeddings em ON em.entity_type = ? AND em.entity_id = t.id
		WHERE t.deleted_at IS NULL AND (em.entity_id IS NULL OR em.embedded_at < %s)
		ORDER BY %s DESC
		LIMIT 32`, t.textSQL, t.table, t.changedSQL, t.changedSQL)
	if err := w.db.NewRaw(q, t.entityType).Scan(ctx, &rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	texts := make([]string, len(rows))
	for i, r := range rows {
		texts[i] = r.Text
	}
	vecs, err := w.embed.Embed(ctx, texts, "passage")
	if err != nil || len(vecs) != len(rows) {
		return err
	}
	for i, r := range rows {
		if _, err := w.db.NewRaw(
			`INSERT INTO embeddings (workspace_id, entity_type, entity_id, embedding, embedded_at)
			 VALUES (?, ?, ?, ?::vector, now())
			 ON CONFLICT (entity_type, entity_id)
			 DO UPDATE SET embedding = EXCLUDED.embedding, embedded_at = now(), workspace_id = EXCLUDED.workspace_id`,
			r.WorkspaceID, t.entityType, r.ID, embed.VecLiteral(vecs[i]),
		).Exec(ctx); err != nil {
			continue
		}
	}
	return nil
}

func NewEmbedSweepWorker(db *bun.DB, e *embed.Client) *EmbedSweepWorker {
	return &EmbedSweepWorker{db: db, embed: e}
}

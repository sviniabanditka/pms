package jobs

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/embed"
)

// Embed-sweep: считает эмбеддинги для issue без вектора / со stale (embedded_at < updated_at).
// Кросс-тенантно (pms_jobs BYPASSRLS). Покрывает и бэкфилл, и текущие изменения.

type EmbedSweepArgs struct{}

func (EmbedSweepArgs) Kind() string { return "embed_sweep" }

type EmbedSweepWorker struct {
	river.WorkerDefaults[EmbedSweepArgs]
	db    *bun.DB
	embed *embed.Client
}

func descText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var d struct {
		MD string `json:"md"`
	}
	_ = json.Unmarshal(raw, &d)
	return d.MD
}

func (w *EmbedSweepWorker) Work(ctx context.Context, _ *river.Job[EmbedSweepArgs]) error {
	if w.embed == nil {
		return nil
	}
	var rows []struct {
		ID          uuid.UUID       `bun:"id"`
		Title       string          `bun:"title"`
		Description json.RawMessage `bun:"description"`
	}
	if err := w.db.NewSelect().Table("issues").Column("id", "title", "description").
		Where("(embedding IS NULL OR embedded_at IS NULL OR embedded_at < updated_at)").
		Where("deleted_at IS NULL").
		OrderExpr("updated_at DESC").Limit(32).Scan(ctx, &rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	texts := make([]string, len(rows))
	for i, r := range rows {
		t := r.Title
		if d := descText(r.Description); d != "" {
			t += " " + d
		}
		texts[i] = t
	}
	vecs, err := w.embed.Embed(ctx, texts, "passage")
	if err != nil {
		return err
	}
	if len(vecs) != len(rows) {
		return nil
	}
	for i, r := range rows {
		if _, err := w.db.NewUpdate().Table("issues").
			Set("embedding = ?::vector", embed.VecLiteral(vecs[i])).
			Set("embedded_at = now()").
			Where("id = ?", r.ID).Exec(ctx); err != nil {
			continue
		}
	}
	return nil
}

func NewEmbedSweepWorker(db *bun.DB, e *embed.Client) *EmbedSweepWorker {
	return &EmbedSweepWorker{db: db, embed: e}
}

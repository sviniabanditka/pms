package jobs

import (
	"context"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/embed"
)

// Cluster-worker (AI Фаза 4): топик-кластеризация issue по эмбеддингам (HDBSCAN в
// сайдкаре) → таблица issue_clusters. Периодический, кросс-тенантно (BYPASSRLS).

type ClusterArgs struct{}

func (ClusterArgs) Kind() string { return "cluster_issues" }

type ClusterWorker struct {
	river.WorkerDefaults[ClusterArgs]
	db    *bun.DB
	embed *embed.Client
}

type clusterRow struct {
	bun.BaseModel `bun:"table:issue_clusters"`
	WorkspaceID   uuid.UUID `bun:"workspace_id"`
	IssueID       uuid.UUID `bun:"issue_id"`
	Cluster       int       `bun:"cluster"`
}

func (w *ClusterWorker) Work(ctx context.Context, _ *river.Job[ClusterArgs]) error {
	if w.embed == nil {
		return nil
	}
	var wss []uuid.UUID
	if err := w.db.NewSelect().Table("workspaces").Column("id").Where("deleted_at IS NULL").Scan(ctx, &wss); err != nil {
		return err
	}
	for _, ws := range wss {
		w.clusterWorkspace(ctx, ws)
	}
	return nil
}

func (w *ClusterWorker) clusterWorkspace(ctx context.Context, ws uuid.UUID) {
	var rows []struct {
		ID  uuid.UUID `bun:"id"`
		Vec string    `bun:"vec"`
	}
	if err := w.db.NewRaw(
		`SELECT i.id, e.embedding::text AS vec
		 FROM issues i JOIN embeddings e ON e.entity_type='issue' AND e.entity_id = i.id
		 WHERE i.workspace_id = ? AND i.deleted_at IS NULL AND i.archived_at IS NULL`, ws).Scan(ctx, &rows); err != nil || len(rows) < 30 {
		return
	}
	vecs := make([][]float32, 0, len(rows))
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		v := embed.ParseVecLiteral(r.Vec)
		if v == nil {
			continue
		}
		vecs = append(vecs, v)
		ids = append(ids, r.ID)
	}
	labels, err := w.embed.Cluster(ctx, vecs, 8)
	if err != nil || len(labels) != len(ids) {
		return
	}
	out := make([]clusterRow, 0, len(ids))
	for i, id := range ids {
		out = append(out, clusterRow{WorkspaceID: ws, IssueID: id, Cluster: labels[i]})
	}
	// заменяем целиком для воркспейса
	_ = w.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*clusterRow)(nil)).Where("workspace_id = ?", ws).Exec(ctx); err != nil {
			return err
		}
		if len(out) > 0 {
			if _, err := tx.NewInsert().Model(&out).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

func NewClusterWorker(db *bun.DB, e *embed.Client) *ClusterWorker {
	return &ClusterWorker{db: db, embed: e}
}

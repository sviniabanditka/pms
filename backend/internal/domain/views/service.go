package views

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/changelog"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var ErrNameRequired = errors.New("view name required")

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

type CreateInput struct {
	ID         uuid.UUID
	Name       string
	DataSource string
	ScopeKind  string
	TeamID     *uuid.UUID
	Filter     []byte
	Grouping   []byte
	Layout     string
}

func (s *Service) CreateView(ctx context.Context, actor, wsID uuid.UUID, in CreateInput) (int64, error) {
	if strings.TrimSpace(in.Name) == "" {
		return 0, ErrNameRequired
	}
	def := func(v, d string) string {
		if v == "" {
			return d
		}
		return v
	}
	filter := in.Filter
	if len(filter) == 0 {
		filter = []byte("{}")
	}
	now := time.Now().UTC()

	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		v := &models.View{
			ID: in.ID, WorkspaceID: wsID, Name: strings.TrimSpace(in.Name),
			DataSource: def(in.DataSource, "issue"), ScopeKind: def(in.ScopeKind, "team"),
			TeamID: in.TeamID, OwnerID: &actor, Filter: filter, Grouping: in.Grouping,
			Layout: def(in.Layout, "list"), CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(v).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "view", in.ID, 0, "create", map[string]any{"name": v.Name}, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "view", EntityID: in.ID.String(), Op: "create"})
	return 0, nil
}

type UpdatePatch struct {
	Name     *string
	Filter   []byte // nil = не менять
	Grouping []byte
	Layout   *string
}

func (s *Service) UpdateView(ctx context.Context, actor, wsID, id uuid.UUID, p UpdatePatch) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewUpdate().Model((*models.View)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if p.Name != nil {
			q.Set("name = ?", strings.TrimSpace(*p.Name))
		}
		if p.Filter != nil {
			q.Set("filter = ?", p.Filter)
		}
		if p.Grouping != nil {
			q.Set("grouping = ?", p.Grouping)
		}
		if p.Layout != nil {
			q.Set("layout = ?", *p.Layout)
		}
		if _, err := q.Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "view", id, 0, "update", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "view", EntityID: id.String(), Op: "update"})
	return 0, nil
}

func (s *Service) DeleteView(ctx context.Context, actor, wsID, id uuid.UUID) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model((*models.View)(nil)).
			Set("deleted_at = now()").Set("updated_at = now()").Set("version = version + 1").
			Where("id = ?", id).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "view", id, 0, "delete", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "view", EntityID: id.String(), Op: "delete"})
	return 0, nil
}

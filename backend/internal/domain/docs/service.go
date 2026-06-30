package docs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/authz"
	"github.com/pms/backend/internal/changelog"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var ErrTitle = errors.New("doc title required")

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) CreateDoc(ctx context.Context, actor, wsID, id uuid.UUID, teamID *uuid.UUID, title string, parent *uuid.UUID) (int64, error) {
	if strings.TrimSpace(title) == "" {
		return 0, ErrTitle
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		d := &models.Document{
			ID: id, WorkspaceID: wsID, TeamID: teamID, ParentID: parent, ScopeKind: "team",
			Title: strings.TrimSpace(title), Body: json.RawMessage(`{}`), CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(d).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "document", id, 0, "create", map[string]any{"title": d.Title}, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "document", EntityID: id.String(), Op: "create"})
	return 0, nil
}

func (s *Service) UpdateDoc(ctx context.Context, actor, wsID, id uuid.UUID, title *string, body []byte) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewUpdate().Model((*models.Document)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if title != nil {
			q.Set("title = ?", strings.TrimSpace(*title))
		}
		if body != nil {
			q.Set("body = ?", json.RawMessage(body))
		}
		if _, err := q.Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "document", id, 0, "update", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "document", EntityID: id.String(), Op: "update"})
	return 0, nil
}

func (s *Service) DeleteDoc(ctx context.Context, actor, wsID, id uuid.UUID) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var owner uuid.UUID
		if err := tx.NewSelect().Model((*models.Document)(nil)).Column("created_by").Where("id = ?", id).Scan(ctx, &owner); err != nil {
			return err
		}
		if !authz.OwnerOrAdmin(ctx, owner, actor) {
			return authz.ErrForbidden
		}
		if _, err := tx.NewUpdate().Model((*models.Document)(nil)).
			Set("deleted_at = now()").Set("updated_at = now()").Set("version = version + 1").
			Where("id = ?", id).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "document", id, 0, "delete", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "document", EntityID: id.String(), Op: "delete"})
	return 0, nil
}

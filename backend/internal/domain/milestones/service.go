package milestones

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/authz"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var ErrName = errors.New("milestone name required")

// mustManage — майлстоунами команды управляет lead команды-или-ancestor либо ws-admin.
func mustManage(ctx context.Context, tx bun.Tx, actor, teamID uuid.UUID) error {
	ok, err := authz.CanManageTeam(ctx, tx, actor, teamID)
	if err != nil {
		return err
	}
	if !ok {
		return authz.ErrForbidden
	}
	return nil
}

func milestoneTeam(ctx context.Context, tx bun.Tx, id uuid.UUID) (uuid.UUID, error) {
	var t uuid.UUID
	err := tx.NewSelect().Model((*models.Milestone)(nil)).Column("team_id").Where("id = ?", id).Scan(ctx, &t)
	return t, err
}

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func (s *Service) Create(ctx context.Context, actor, wsID, id, teamID uuid.UUID, name, target string) error {
	if strings.TrimSpace(name) == "" {
		return ErrName
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := mustManage(ctx, tx, actor, teamID); err != nil {
			return err
		}
		m := &models.Milestone{ID: id, WorkspaceID: wsID, TeamID: teamID, Name: strings.TrimSpace(name), TargetDate: parseDate(target), Status: "planned", Position: "n", CreatedAt: now, UpdatedAt: now}
		_, e := tx.NewInsert().Model(m).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "milestone", EntityID: id.String(), Op: "create"})
	return nil
}

func (s *Service) Update(ctx context.Context, actor, wsID, id uuid.UUID, name, target, status *string) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		teamID, err := milestoneTeam(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := mustManage(ctx, tx, actor, teamID); err != nil {
			return err
		}
		q := tx.NewUpdate().Model((*models.Milestone)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if name != nil {
			q.Set("name = ?", strings.TrimSpace(*name))
		}
		if target != nil {
			q.Set("target_date = ?", parseDate(*target))
		}
		if status != nil {
			q.Set("status = ?", *status)
		}
		_, e := q.Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "milestone", EntityID: id.String(), Op: "update"})
	return nil
}

func (s *Service) Delete(ctx context.Context, actor, wsID, id uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		teamID, e := milestoneTeam(ctx, tx, id)
		if e != nil {
			return e
		}
		if e := mustManage(ctx, tx, actor, teamID); e != nil {
			return e
		}
		if _, e := tx.NewUpdate().Model((*models.Issue)(nil)).Set("milestone_id = NULL").Where("milestone_id = ?", id).Exec(ctx); e != nil {
			return e
		}
		_, e = tx.NewUpdate().Model((*models.Milestone)(nil)).Set("deleted_at = now()").Where("id = ?", id).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "milestone", EntityID: id.String(), Op: "delete"})
	return nil
}

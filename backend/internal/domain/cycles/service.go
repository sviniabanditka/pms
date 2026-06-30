package cycles

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/authz"
	"github.com/pms/backend/internal/changelog"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var ErrDates = errors.New("starts_on and ends_on required (ends after starts)")

// mustManage — циклами команды управляет lead команды-или-ancestor либо ws-admin.
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

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) CreateCycle(ctx context.Context, actor, wsID, teamID, id uuid.UUID, name string, startsOn, endsOn time.Time) (int64, error) {
	if endsOn.Before(startsOn) {
		return 0, ErrDates
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := mustManage(ctx, tx, actor, teamID); err != nil {
			return err
		}
		// сериализуем нумерацию по команде на время транзакции (MAX+1 иначе гонка → дубли number)
		if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(hashtext(?))", "cycle:"+teamID.String()); err != nil {
			return err
		}
		var number int
		if err := tx.NewSelect().Model((*models.Cycle)(nil)).
			ColumnExpr("COALESCE(MAX(number), 0) + 1").Where("team_id = ?", teamID).Scan(ctx, &number); err != nil {
			return err
		}
		c := &models.Cycle{
			ID: id, WorkspaceID: wsID, TeamID: teamID, Number: number,
			StartsOn: startsOn, EndsOn: endsOn, CreatedAt: now, UpdatedAt: now,
		}
		if name != "" {
			c.Name = &name
		}
		if _, err := tx.NewInsert().Model(c).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "cycle", id, 0, "create", map[string]any{"number": number}, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "cycle", EntityID: id.String(), Op: "create"})
	return 0, nil
}

func (s *Service) UpdateCycle(ctx context.Context, actor, wsID, id uuid.UUID, name *string, startsOn, endsOn *time.Time) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var teamID uuid.UUID
		if err := tx.NewSelect().Model((*models.Cycle)(nil)).Column("team_id").Where("id = ?", id).Scan(ctx, &teamID); err != nil {
			return err
		}
		if err := mustManage(ctx, tx, actor, teamID); err != nil {
			return err
		}
		// при частичном апдейте дат проверяем эффективный интервал (старт не позже конца)
		if startsOn != nil || endsOn != nil {
			var cur models.Cycle
			if err := tx.NewSelect().Model(&cur).Column("starts_on", "ends_on").Where("id = ?", id).Scan(ctx); err != nil {
				return err
			}
			s2, e2 := cur.StartsOn, cur.EndsOn
			if startsOn != nil {
				s2 = *startsOn
			}
			if endsOn != nil {
				e2 = *endsOn
			}
			if e2.Before(s2) {
				return ErrDates
			}
		}
		q := tx.NewUpdate().Model((*models.Cycle)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if name != nil {
			q.Set("name = ?", *name)
		}
		if startsOn != nil {
			q.Set("starts_on = ?", *startsOn)
		}
		if endsOn != nil {
			q.Set("ends_on = ?", *endsOn)
		}
		if _, err := q.Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "cycle", id, 0, "update", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "cycle", EntityID: id.String(), Op: "update"})
	return 0, nil
}

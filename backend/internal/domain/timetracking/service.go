// Package timetracking — учёт времени: таймер (start/stop) + ручные записи, привязка к задаче.
package timetracking

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

var ErrDates = errors.New("ended_at must be after started_at")

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) delta(ctx context.Context, wsID, id uuid.UUID, op string) {
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "time_entry", EntityID: id.String(), Op: op})
}

// StartTimer — останавливает текущий бегущий таймер юзера и стартует новый (running).
func (s *Service) StartTimer(ctx context.Context, actor, wsID, id uuid.UUID, issueID *uuid.UUID, desc string) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		// стопаем все бегущие записи юзера
		if _, err := tx.NewUpdate().Model((*models.TimeEntry)(nil)).
			Set("ended_at = ?", now).Set("version = version + 1").
			Where("user_id = ?", actor).Where("ended_at IS NULL").Where("deleted_at IS NULL").Exec(ctx); err != nil {
			return err
		}
		e := &models.TimeEntry{
			ID: id, WorkspaceID: wsID, UserID: actor, IssueID: issueID,
			Description: strings.TrimSpace(desc), StartedAt: now, CreatedAt: now,
		}
		if _, err := tx.NewInsert().Model(e).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "time_entry", id, 0, "create", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "create")
	return nil
}

// StopTimer — закрывает бегущий таймер юзера (если есть).
func (s *Service) StopTimer(ctx context.Context, actor, wsID uuid.UUID) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.TimeEntry)(nil)).
			Set("ended_at = ?", now).Set("version = version + 1").
			Where("user_id = ?", actor).Where("ended_at IS NULL").Where("deleted_at IS NULL").Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, actor, "update") // id не критичен — клиент перечитает
	return nil
}

// AddEntry — ручная запись с явными границами.
func (s *Service) AddEntry(ctx context.Context, actor, wsID, id uuid.UUID, issueID *uuid.UUID, desc string, startedAt, endedAt time.Time) error {
	if !endedAt.After(startedAt) {
		return ErrDates
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		e := &models.TimeEntry{
			ID: id, WorkspaceID: wsID, UserID: actor, IssueID: issueID,
			Description: strings.TrimSpace(desc), StartedAt: startedAt, EndedAt: &endedAt, CreatedAt: now,
		}
		if _, err := tx.NewInsert().Model(e).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "time_entry", id, 0, "create", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "create")
	return nil
}

// ReportRow — агрегат по группе.
type ReportRow struct {
	Key     string `bun:"k"`
	Minutes int64  `bun:"minutes"`
	Count   int64  `bun:"cnt"`
}

// Report — суммарное время по воркспейсу за период, сгруппированное по user|issue|day.
// RLS пускает читать все записи воркспейса (политика workspace-scoped), так что виден весь учёт команды.
func (s *Service) Report(ctx context.Context, actor, wsID uuid.UUID, from, to time.Time, groupBy string, issueID, userID *uuid.UUID) ([]ReportRow, int64, error) {
	var keyExpr string
	switch groupBy {
	case "issue":
		keyExpr = "COALESCE(issue_id::text, '')"
	case "day":
		keyExpr = "to_char(started_at, 'YYYY-MM-DD')"
	default:
		keyExpr = "user_id::text"
	}
	var rows []ReportRow
	var total int64
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewSelect().Model((*models.TimeEntry)(nil)).
			ColumnExpr(keyExpr+" AS k").
			ColumnExpr("ROUND(SUM(EXTRACT(EPOCH FROM (COALESCE(ended_at, now()) - started_at)) / 60))::bigint AS minutes").
			ColumnExpr("COUNT(*) AS cnt").
			Where("deleted_at IS NULL").
			Where("started_at >= ?", from).Where("started_at < ?", to)
		if issueID != nil {
			q = q.Where("issue_id = ?", *issueID)
		}
		if userID != nil {
			q = q.Where("user_id = ?", *userID)
		}
		if err := q.GroupExpr(keyExpr).OrderExpr("minutes DESC").Scan(ctx, &rows); err != nil {
			return err
		}
		for _, r := range rows {
			total += r.Minutes
		}
		return nil
	})
	return rows, total, err
}

// DeleteEntry — soft-delete своей записи.
func (s *Service) DeleteEntry(ctx context.Context, actor, wsID, id uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.TimeEntry)(nil)).
			Set("deleted_at = now()").Where("id = ?", id).Where("user_id = ?", actor).Exec(ctx)
		if err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "time_entry", id, 0, "delete", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "delete")
	return nil
}

// Package templates — issue-шаблоны (переиспользуемый blueprint) + recurring
// (recurrence<>none → RecurringIssueWorker плодит issue по next_run_at).
package templates

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/authz"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/domain/issues"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
)

var ErrNameRequired = errors.New("template name required")

// requireTeamAccess — шаблон привязан к команде: создавать/править может только
// тот, кто видит команду (private/ltree — как в issues). Заодно отсекает
// team_id чужого воркспейса: под RLS-tx чужая команда невидима → false.
func requireTeamAccess(ctx context.Context, tx bun.Tx, actor, teamID uuid.UUID) error {
	ok, err := authz.CanAccessTeam(ctx, tx, actor, teamID)
	if err != nil {
		return err
	}
	if !ok {
		return authz.ErrForbidden
	}
	return nil
}

type Service struct {
	db     *bun.DB
	issues *issues.Service
	pub    *realtime.Publisher
}

func NewService(bdb *bun.DB, issuesSvc *issues.Service, pub *realtime.Publisher) *Service {
	return &Service{db: bdb, issues: issuesSvc, pub: pub}
}

// Input — общие поля для create/update.
type Input struct {
	TeamID        uuid.UUID
	Name          string
	Title         string
	DescriptionMD string
	Priority      int
	AssigneeID    *uuid.UUID
	Estimate      *float64
	LabelIDs      []uuid.UUID
	Recurrence    string
}

func normRec(r string) string {
	switch r {
	case "daily", "weekly", "monthly":
		return r
	default:
		return "none"
	}
}

// advance — следующее срабатывание после from по каденции (проматывает мимо now, если воркер простаивал).
func advance(from time.Time, rec string) time.Time {
	step := func(t time.Time) time.Time {
		switch rec {
		case "daily":
			return t.AddDate(0, 0, 1)
		case "weekly":
			return t.AddDate(0, 0, 7)
		case "monthly":
			return t.AddDate(0, 1, 0)
		}
		return t.AddDate(100, 0, 0) // none: не должно вызываться
	}
	now := time.Now().UTC()
	next := step(from)
	for i := 0; i < 1000 && !next.After(now); i++ {
		next = step(next)
	}
	return next
}

func descJSON(md string) json.RawMessage {
	md = strings.TrimSpace(md)
	if md == "" {
		return json.RawMessage("{}")
	}
	b, _ := json.Marshal(map[string]string{"md": md})
	return b
}

func (s *Service) Create(ctx context.Context, actor, wsID uuid.UUID, in Input) (uuid.UUID, error) {
	if strings.TrimSpace(in.Name) == "" {
		return uuid.Nil, ErrNameRequired
	}
	rec := normRec(in.Recurrence)
	id := uuid.New()
	now := time.Now().UTC()
	labels, _ := json.Marshal(in.LabelIDs)
	t := &models.IssueTemplate{
		ID: id, WorkspaceID: wsID, TeamID: in.TeamID, Name: strings.TrimSpace(in.Name),
		Title: strings.TrimSpace(in.Title), Description: descJSON(in.DescriptionMD),
		Priority: int16(in.Priority), AssigneeID: in.AssigneeID, Estimate: in.Estimate,
		LabelIDs: labels, Recurrence: rec, CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
	}
	if rec != "none" {
		nr := advance(now, rec)
		t.NextRunAt = &nr
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if e := requireTeamAccess(ctx, tx, actor, in.TeamID); e != nil {
			return e
		}
		_, e := tx.NewInsert().Model(t).Exec(ctx)
		return e
	})
	if err != nil {
		return uuid.Nil, err
	}
	s.publish(ctx, wsID, id, "create")
	return id, nil
}

func (s *Service) Update(ctx context.Context, actor, wsID, id uuid.UUID, in Input) error {
	if strings.TrimSpace(in.Name) == "" {
		return ErrNameRequired
	}
	rec := normRec(in.Recurrence)
	now := time.Now().UTC()
	labels, _ := json.Marshal(in.LabelIDs)
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		// прежняя каденция — чтобы понять, надо ли (пере)ставить next_run_at
		var prev models.IssueTemplate
		if err := tx.NewSelect().Model(&prev).Where("id = ?", id).Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		// доступ и к старой команде шаблона, и к новой (если переносят)
		if err := requireTeamAccess(ctx, tx, actor, prev.TeamID); err != nil {
			return err
		}
		if in.TeamID != prev.TeamID {
			if err := requireTeamAccess(ctx, tx, actor, in.TeamID); err != nil {
				return err
			}
		}
		q := tx.NewUpdate().Model((*models.IssueTemplate)(nil)).
			Set("team_id = ?", in.TeamID).Set("name = ?", strings.TrimSpace(in.Name)).
			Set("title = ?", strings.TrimSpace(in.Title)).Set("description = ?", descJSON(in.DescriptionMD)).
			Set("priority = ?", int16(in.Priority)).Set("assignee_id = ?", in.AssigneeID).
			Set("estimate = ?", in.Estimate).Set("label_ids = ?", labels).
			Set("recurrence = ?", rec).Set("updated_at = ?", now)
		switch {
		case rec == "none":
			q = q.Set("next_run_at = NULL")
		case prev.Recurrence != rec || prev.NextRunAt == nil:
			q = q.Set("next_run_at = ?", advance(now, rec)) // новая/изменённая каденция → пересчёт
		}
		_, e := q.Where("id = ?", id).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	s.publish(ctx, wsID, id, "update")
	return nil
}

func (s *Service) Delete(ctx context.Context, actor, wsID, id uuid.UUID) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var t models.IssueTemplate
		if e := tx.NewSelect().Model(&t).Column("created_by").Where("id = ?", id).Where("deleted_at IS NULL").Scan(ctx); e != nil {
			return e
		}
		// delete = creator|admin (как у прочих сущностей)
		if !authz.OwnerOrAdmin(ctx, t.CreatedBy, actor) {
			return authz.ErrForbidden
		}
		_, e := tx.NewUpdate().Model((*models.IssueTemplate)(nil)).
			Set("deleted_at = ?", now).Where("id = ?", id).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	s.publish(ctx, wsID, id, "delete")
	return nil
}

// Spawn — создать issue из шаблона прямо сейчас (RPC "create now").
func (s *Service) Spawn(ctx context.Context, actor, wsID, id uuid.UUID) (uuid.UUID, error) {
	var t models.IssueTemplate
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewSelect().Model(&t).Where("id = ?", id).Where("deleted_at IS NULL").Scan(ctx)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return s.spawn(ctx, actor, &t)
}

// spawn — сделать issue из шаблона через issues.Service (валидит доступ, шлёт дельту, триггерит автоматизации).
func (s *Service) spawn(ctx context.Context, actor uuid.UUID, t *models.IssueTemplate) (uuid.UUID, error) {
	// team_id обязан принадлежать воркспейсу шаблона: на BYPASSRLS-коннекте воркера
	// RLS не отсечёт чужую команду, поэтому проверяем явно.
	var teamOK bool
	if err := db.WithWorkspace(ctx, s.db, t.WorkspaceID, func(ctx context.Context, tx bun.Tx) error {
		ok, e := tx.NewSelect().Model((*models.Team)(nil)).
			Where("id = ?", t.TeamID).Where("workspace_id = ?", t.WorkspaceID).Exists(ctx)
		teamOK = ok
		return e
	}); err != nil {
		return uuid.Nil, err
	}
	if !teamOK {
		return uuid.Nil, authz.ErrForbidden
	}
	id := uuid.New()
	title := strings.TrimSpace(t.Title)
	if title == "" {
		title = t.Name
	}
	if _, err := s.issues.CreateIssue(ctx, actor, t.WorkspaceID, t.TeamID, id, title, "", int(t.Priority), t.AssigneeID, nil); err != nil {
		return uuid.Nil, err
	}
	// дополняем поля, которых нет в CreateIssue (best-effort)
	var d struct {
		MD string `json:"md"`
	}
	_ = json.Unmarshal(t.Description, &d)
	if d.MD != "" {
		_, _ = s.issues.SetDescription(ctx, actor, t.WorkspaceID, id, d.MD)
	}
	if t.Estimate != nil {
		_, _ = s.issues.SetEstimate(ctx, actor, t.WorkspaceID, id, *t.Estimate)
	}
	var labelIDs []uuid.UUID
	_ = json.Unmarshal(t.LabelIDs, &labelIDs)
	for _, lid := range labelIDs {
		_ = s.issues.AddLabel(ctx, actor, t.WorkspaceID, id, lid)
	}
	return id, nil
}

// SpawnDue — воркер: кросс-тенантно (BYPASSRLS) проматывает все due recurring-шаблоны.
func (s *Service) SpawnDue(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	var due []models.IssueTemplate
	if err := s.db.NewSelect().Model(&due).
		Where("recurrence <> 'none'").Where("deleted_at IS NULL").
		Where("next_run_at IS NOT NULL").Where("next_run_at <= ?", now).Scan(ctx); err != nil {
		return 0, err
	}
	spawned := 0
	for i := range due {
		t := &due[i]
		// claim-first: сдвигаем next_run_at ДО спавна (оптимистично по прежнему значению).
		// Фейл спавна → occurrence пропущен, ретрай на следующей каденции (не hot-loop);
		// фейл после спавна не задвоит issue; конкурентный воркер не возьмёт тот же claim.
		next := advance(now, t.Recurrence)
		res, err := s.db.NewUpdate().Model((*models.IssueTemplate)(nil)).
			Set("next_run_at = ?", next).Set("updated_at = ?", now).
			Where("id = ?", t.ID).Where("next_run_at = ?", t.NextRunAt).Exec(ctx)
		if err != nil {
			slog.Warn("recurring: claim failed", "template", t.ID, "err", err)
			continue
		}
		if n, _ := res.RowsAffected(); n == 0 {
			continue // кто-то уже забрал
		}
		if _, err := s.spawn(ctx, t.CreatedBy, t); err != nil {
			slog.Warn("recurring: spawn failed", "template", t.ID, "ws", t.WorkspaceID, "err", err)
			continue
		}
		if _, err := s.db.NewUpdate().Model((*models.IssueTemplate)(nil)).
			Set("last_spawned_at = ?", now).Where("id = ?", t.ID).Exec(ctx); err != nil {
			slog.Warn("recurring: last_spawned_at update failed", "template", t.ID, "err", err)
		}
		s.publish(ctx, t.WorkspaceID, t.ID, "update")
		spawned++
	}
	return spawned, nil
}

func (s *Service) publish(ctx context.Context, wsID, id uuid.UUID, op string) {
	if s.pub != nil {
		_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "issue_template", EntityID: id.String(), Op: op})
	}
}

package issues

import (
	"encoding/json"
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/changelog"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/frac"
	"github.com/pms/backend/internal/domain/notifications"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var (
	ErrTitleRequired   = errors.New("title required")
	ErrTeamNotFound    = errors.New("team not found")
	ErrStateNotFound   = errors.New("state not found for team")
	ErrIssueNotFound   = errors.New("issue not found")
	ErrInvalidRelation = errors.New("invalid relation")
)

var relTypes = map[string]bool{"blocks": true, "blocked_by": true, "related": true, "duplicate": true, "duplicate_of": true}

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

// stateTimestamps — чистая функция перехода статуса: какие started_at/completed_at
// должны быть при попадании в category. TDD-узел (§10.1).
func stateTimestamps(category string, now time.Time, curStarted, curCompleted *time.Time) (started, completed *time.Time) {
	started, completed = curStarted, curCompleted
	switch category {
	case "started":
		if started == nil {
			started = &now
		}
		completed = nil
	case "completed", "canceled":
		if started == nil {
			started = &now
		}
		if completed == nil {
			completed = &now
		}
	default: // backlog, unstarted
		started = nil
		completed = nil
	}
	return started, completed
}

// CreateIssue: атомарный KEY-N, дефолтный статус, авто started/completed, change_log.
func (s *Service) CreateIssue(ctx context.Context, actor, wsID, teamID, id uuid.UUID, title, stateID string, priority int, assignee, parent *uuid.UUID) (int64, error) {
	if strings.TrimSpace(title) == "" {
		return 0, ErrTitleRequired
	}
	now := time.Now().UTC()

	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		// sub-issue наследует команду родителя (ADR-0004 #2).
		if parent != nil {
			p := new(models.Issue)
			if err := tx.NewSelect().Model(p).Column("team_id").Where("id = ?", *parent).Scan(ctx); err != nil {
				return ErrIssueNotFound
			}
			teamID = p.TeamID
		}
		// Атомарная нумерация: row-lock на teams сериализует конкурентные создания.
		var key string
		var number int64
		if err := tx.NewRaw(
			`UPDATE teams SET next_issue_number = next_issue_number + 1, updated_at = now(), version = version + 1 WHERE id = ? RETURNING key, next_issue_number`,
			teamID).Scan(ctx, &key, &number); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrTeamNotFound
			}
			return err
		}

		// Резолв статуса (явный или дефолтный backlog команды).
		var stateUUID uuid.UUID
		var category string
		st := tx.NewSelect().Model((*models.WorkflowState)(nil)).Column("id", "category").Where("team_id = ?", teamID)
		if stateID != "" {
			parsed, perr := uuid.Parse(stateID)
			if perr != nil {
				return ErrStateNotFound
			}
			st = st.Where("id = ?", parsed)
		} else {
			st = st.Where("category = 'backlog'").Limit(1)
		}
		if err := st.Scan(ctx, &stateUUID, &category); err != nil {
			return ErrStateNotFound
		}

		// позиция: дописываем в конец колонки (team, state). Фракционный ключ-строка (D20).
		var maxPos string
		if err := tx.NewSelect().Model((*models.Issue)(nil)).
			ColumnExpr("COALESCE(MAX(board_position), '')").
			Where("team_id = ?", teamID).Where("state_id = ?", stateUUID).Scan(ctx, &maxPos); err != nil {
			return err
		}

		started, completed := stateTimestamps(category, now, nil, nil)
		issue := &models.Issue{
			ID: id, WorkspaceID: wsID, TeamID: teamID, Number: number,
			Identifier: key + "-" + strconv.FormatInt(number, 10),
			Title:      strings.TrimSpace(title), Description: []byte("{}"),
			StateID: stateUUID, Priority: int16(priority), AssigneeID: assignee, ParentID: parent,
			StartedAt: started, CompletedAt: completed,
			BoardPosition: frac.Between(maxPos, ""), SubPosition: frac.First(), CreatedBy: actor,
			CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(issue).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "issue", id, 0, "create",
			map[string]any{"identifier": issue.Identifier, "title": issue.Title}, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "issue", EntityID: id.String(), Op: "create"})
	return 0, nil
}

// UpdateIssueState: меняет статус + пересчитывает started_at/completed_at.
func (s *Service) UpdateIssueState(ctx context.Context, actor, wsID, id uuid.UUID, stateID string) (int64, error) {
	newState, err := uuid.Parse(stateID)
	if err != nil {
		return 0, ErrStateNotFound
	}
	now := time.Now().UTC()

	err = db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		cur := new(models.Issue)
		if err := tx.NewSelect().Model(cur).Column("team_id", "started_at", "completed_at", "created_by", "assignee_id").Where("id = ?", id).Scan(ctx); err != nil {
			return ErrIssueNotFound
		}
		var category string
		if err := tx.NewSelect().Model((*models.WorkflowState)(nil)).Column("category").
			Where("id = ?", newState).Where("team_id = ?", cur.TeamID).Scan(ctx, &category); err != nil {
			return ErrStateNotFound
		}
		started, completed := stateTimestamps(category, now, cur.StartedAt, cur.CompletedAt)
		_, err := tx.NewUpdate().Model((*models.Issue)(nil)).
			Set("state_id = ?", newState).Set("started_at = ?", started).Set("completed_at = ?", completed).
			Set("updated_at = now()").Set("version = version + 1").Where("id = ?", id).Exec(ctx)
		if err != nil {
			return err
		}
		if err := changelog.Record(ctx, tx, wsID, "issue", id, 0, "update", map[string]any{"stateId": newState}, actor); err != nil {
			return err
		}
		// уведомить создателя + assignee (кроме actor)
		recips := map[uuid.UUID]bool{}
		if cur.CreatedBy != actor {
			recips[cur.CreatedBy] = true
		}
		if cur.AssigneeID != nil && *cur.AssigneeID != actor {
			recips[*cur.AssigneeID] = true
		}
		for r := range recips {
			if err := notifications.Record(ctx, tx, wsID, r, actor, "state_changed", "issue", id, nil); err != nil {
				return err
			}
		}
		return nil
	})
	return 0, s.afterMutate(ctx, wsID, id, err)
}

// ReorderIssue: перенос в колонку stateID (если меняется) + позиция между before/after.
func (s *Service) ReorderIssue(ctx context.Context, actor, wsID, id uuid.UUID, stateID string, beforeID, afterID *uuid.UUID) (int64, error) {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		cur := new(models.Issue)
		if err := tx.NewSelect().Model(cur).Column("team_id", "state_id", "started_at", "completed_at").Where("id = ?", id).Scan(ctx); err != nil {
			return ErrIssueNotFound
		}
		targetState := cur.StateID
		if stateID != "" {
			parsed, perr := uuid.Parse(stateID)
			if perr != nil {
				return ErrStateNotFound
			}
			targetState = parsed
		}
		var category string
		if err := tx.NewSelect().Model((*models.WorkflowState)(nil)).Column("category").
			Where("id = ?", targetState).Where("team_id = ?", cur.TeamID).Scan(ctx, &category); err != nil {
			return ErrStateNotFound
		}

		posOf := func(nid *uuid.UUID) string {
			if nid == nil {
				return ""
			}
			var p string
			if err := tx.NewSelect().Model((*models.Issue)(nil)).Column("board_position").Where("id = ?", *nid).Scan(ctx, &p); err != nil {
				return "" // сосед не найден → открытый край
			}
			return p
		}
		beforePos := posOf(beforeID)
		afterPos := posOf(afterID)
		// Защита от инверсии (рассинхрон клиента): если before>=after — дописать после before.
		if beforePos != "" && afterPos != "" && beforePos >= afterPos {
			afterPos = ""
		}
		newPos := frac.Between(beforePos, afterPos)

		started, completed := stateTimestamps(category, now, cur.StartedAt, cur.CompletedAt)
		if _, err := tx.NewUpdate().Model((*models.Issue)(nil)).
			Set("state_id = ?", targetState).Set("board_position = ?", newPos).
			Set("started_at = ?", started).Set("completed_at = ?", completed).
			Set("updated_at = now()").Set("version = version + 1").Where("id = ?", id).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "issue", id, 0, "update", map[string]any{"reorder": true}, actor)
	})
	return 0, s.afterMutate(ctx, wsID, id, err)
}

// AddRelation: типизированная связь source→target. Зеркало выводится в чтении (одна строка).
func (s *Service) AddRelation(ctx context.Context, actor, wsID, id, source, target uuid.UUID, typ string) (int64, error) {
	if !relTypes[typ] || source == target {
		return 0, ErrInvalidRelation
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		rel := &models.IssueRelation{ID: id, WorkspaceID: wsID, SourceID: source, TargetID: target, Type: typ, CreatedAt: now}
		if _, err := tx.NewInsert().Model(rel).Exec(ctx); err != nil {
			if strings.Contains(err.Error(), "issue_relations_source_id_target_id_type_key") {
				return ErrInvalidRelation
			}
			return err
		}
		if err := changelog.Record(ctx, tx, wsID, "issue", source, 0, "update", map[string]any{"relation": true}, actor); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "issue", target, 0, "update", map[string]any{"relation": true}, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "issue", EntityID: source.String(), Op: "update"})
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "issue", EntityID: target.String(), Op: "update"})
	return 0, nil
}

// RemoveRelation по id связи.
func (s *Service) RemoveRelation(ctx context.Context, actor, wsID, relID uuid.UUID) (int64, error) {
	var src, tgt uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		rel := new(models.IssueRelation)
		if err := tx.NewSelect().Model(rel).Column("source_id", "target_id").Where("id = ?", relID).Scan(ctx); err != nil {
			return ErrInvalidRelation
		}
		src, tgt = rel.SourceID, rel.TargetID
		if _, err := tx.NewDelete().Model((*models.IssueRelation)(nil)).Where("id = ?", relID).Exec(ctx); err != nil {
			return err
		}
		if err := changelog.Record(ctx, tx, wsID, "issue", src, 0, "update", nil, actor); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "issue", tgt, 0, "update", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "issue", EntityID: src.String(), Op: "update"})
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "issue", EntityID: tgt.String(), Op: "update"})
	return 0, nil
}

// AssignIssue: assignee=nil → снять. Уведомляет нового assignee (≠ actor).
func (s *Service) AssignIssue(ctx context.Context, actor, wsID, id uuid.UUID, assignee *uuid.UUID) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewUpdate().Model((*models.Issue)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if assignee == nil {
			q.Set("assignee_id = NULL")
		} else {
			q.Set("assignee_id = ?", *assignee)
		}
		if _, err := q.Exec(ctx); err != nil {
			return err
		}
		if err := changelog.Record(ctx, tx, wsID, "issue", id, 0, "update", nil, actor); err != nil {
			return err
		}
		if assignee != nil && *assignee != actor {
			return notifications.Record(ctx, tx, wsID, *assignee, actor, "assigned", "issue", id, nil)
		}
		return nil
	})
	return 0, s.afterMutate(ctx, wsID, id, err)
}

// SetPriority.
func (s *Service) SetPriority(ctx context.Context, actor, wsID, id uuid.UUID, priority int) (int64, error) {
	err := s.simpleUpdate(ctx, actor, wsID, id, func(q *bun.UpdateQuery) {
		q.Set("priority = ?", int16(priority))
	})
	return 0, err
}

// SetCustomFieldValue: upsert значения; пустой value → снять.
func (s *Service) SetCustomFieldValue(ctx context.Context, actor, wsID, issueID, fieldID uuid.UUID, value []byte) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if len(value) == 0 || string(value) == "null" {
			if _, err := tx.NewDelete().Model((*models.CustomFieldValue)(nil)).
				Where("field_id = ?", fieldID).Where("issue_id = ?", issueID).Exec(ctx); err != nil {
				return err
			}
		} else {
			vid, _ := uuid.NewV7()
			cfv := &models.CustomFieldValue{
				ID: vid, WorkspaceID: wsID, FieldID: fieldID, IssueID: issueID, Value: value,
				CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
			}
			if _, err := tx.NewInsert().Model(cfv).
				On("CONFLICT (field_id, issue_id) DO UPDATE").
				Set("value = EXCLUDED.value").Set("updated_at = now()").Exec(ctx); err != nil {
				return err
			}
		}
		return changelog.Record(ctx, tx, wsID, "issue", issueID, 0, "update", map[string]any{"customField": fieldID}, actor)
	})
	return 0, s.afterMutate(ctx, wsID, issueID, err)
}

// SetScheduledDate: "" → снять; иначе YYYY-MM-DD.
func (s *Service) SetScheduledDate(ctx context.Context, actor, wsID, id uuid.UUID, date string) (int64, error) {
	return 0, s.simpleUpdate(ctx, actor, wsID, id, func(q *bun.UpdateQuery) {
		if date == "" {
			q.Set("scheduled_date = NULL")
		} else {
			q.Set("scheduled_date = ?::date", date)
		}
	})
}

// SetDueDate: "" → снять; иначе YYYY-MM-DD.
func (s *Service) SetDueDate(ctx context.Context, actor, wsID, id uuid.UUID, date string) (int64, error) {
	return 0, s.simpleUpdate(ctx, actor, wsID, id, func(q *bun.UpdateQuery) {
		if date == "" {
			q.Set("due_date = NULL")
		} else {
			q.Set("due_date = ?::date", date)
		}
	})
}

// SetCycle: cycle=nil → снять.
func (s *Service) SetCycle(ctx context.Context, actor, wsID, id uuid.UUID, cycle *uuid.UUID) (int64, error) {
	return 0, s.simpleUpdate(ctx, actor, wsID, id, func(q *bun.UpdateQuery) {
		if cycle == nil {
			q.Set("cycle_id = NULL")
		} else {
			q.Set("cycle_id = ?", *cycle)
		}
	})
}

// SetTitle.
func (s *Service) SetTitle(ctx context.Context, actor, wsID, id uuid.UUID, title string) (int64, error) {
	if strings.TrimSpace(title) == "" {
		return 0, ErrTitleRequired
	}
	err := s.simpleUpdate(ctx, actor, wsID, id, func(q *bun.UpdateQuery) {
		q.Set("title = ?", strings.TrimSpace(title))
	})
	return 0, err
}

// SetDescription: markdown → хранится как jsonb {"md": ...}.
func (s *Service) SetDescription(ctx context.Context, actor, wsID, id uuid.UUID, md string) (int64, error) {
	body, _ := json.Marshal(map[string]string{"md": md})
	err := s.simpleUpdate(ctx, actor, wsID, id, func(q *bun.UpdateQuery) {
		q.Set("description = ?", json.RawMessage(body))
	})
	return 0, err
}

// ArchiveIssue.
func (s *Service) ArchiveIssue(ctx context.Context, actor, wsID, id uuid.UUID) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.Issue)(nil)).
			Set("archived_at = now()").Set("updated_at = now()").Set("version = version + 1").
			Where("id = ?", id).Exec(ctx)
		if err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "issue", id, 0, "archive", nil, actor)
	})
	return 0, s.afterMutate(ctx, wsID, id, err)
}

// simpleUpdate — общий паттерн: set-поля + version++ + change_log + publish.
// SetMilestone: milestone=nil → снять.
func (s *Service) SetMilestone(ctx context.Context, actor, wsID, id uuid.UUID, ms *uuid.UUID) (int64, error) {
	return 0, s.simpleUpdate(ctx, actor, wsID, id, func(q *bun.UpdateQuery) {
		if ms == nil {
			q.Set("milestone_id = NULL")
		} else {
			q.Set("milestone_id = ?", *ms)
		}
	})
}

// SetEstimate: estimate<=0 → снять.
func (s *Service) SetEstimate(ctx context.Context, actor, wsID, id uuid.UUID, estimate float64) (int64, error) {
	return 0, s.simpleUpdate(ctx, actor, wsID, id, func(q *bun.UpdateQuery) {
		if estimate <= 0 {
			q.Set("estimate = NULL")
		} else {
			q.Set("estimate = ?", estimate)
		}
	})
}

type activityDTO struct {
	Op            string          `json:"op"`
	Version       int64           `json:"version"`
	ActorID       *uuid.UUID      `json:"actorId,omitempty"`
	At            time.Time       `json:"at"`
	ChangedFields json.RawMessage `json:"changedFields,omitempty"`
}

// GetActivity — история изменений сущности из change_log.
func (s *Service) GetActivity(ctx context.Context, actor, wsID, entityID uuid.UUID) ([]byte, error) {
	var rows []models.ChangeLog
	var out []byte
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewSelect().Model(&rows).Where("entity_id = ?", entityID).Order("seq DESC").Limit(100).Scan(ctx); err != nil {
			return err
		}
		dtos := make([]activityDTO, len(rows))
		for i := range rows {
			dtos[i] = activityDTO{Op: rows[i].Op, Version: rows[i].Version, ActorID: rows[i].ActorID, At: rows[i].At, ChangedFields: json.RawMessage(rows[i].ChangedFields)}
		}
		raw, _ := json.Marshal(dtos)
		out = raw
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []byte("[]")
	}
	return out, nil
}

func (s *Service) AddLabel(ctx context.Context, actor, wsID, issueID, labelID uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(&models.IssueLabel{WorkspaceID: wsID, IssueID: issueID, LabelID: labelID}).
			On("CONFLICT DO NOTHING").Exec(ctx)
		return err
	})
	return s.afterMutate(ctx, wsID, issueID, err)
}

func (s *Service) RemoveLabel(ctx context.Context, actor, wsID, issueID, labelID uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewDelete().Model((*models.IssueLabel)(nil)).
			Where("issue_id = ?", issueID).Where("label_id = ?", labelID).Exec(ctx)
		return err
	})
	return s.afterMutate(ctx, wsID, issueID, err)
}

func (s *Service) simpleUpdate(ctx context.Context, actor, wsID, id uuid.UUID, setFn func(*bun.UpdateQuery)) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewUpdate().Model((*models.Issue)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		setFn(q)
		if _, err := q.Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "issue", id, 0, "update", nil, actor)
	})
	return s.afterMutate(ctx, wsID, id, err)
}

func (s *Service) afterMutate(ctx context.Context, wsID, id uuid.UUID, err error) error {
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "issue", EntityID: id.String(), Op: "update"})
	return nil
}

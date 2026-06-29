// Package automations — правила «триггер → действия» над задачами.
// Engine вызывается после issue-мутаций; действия пишутся напрямую в БД (без рекурсии через issue-сервис).
package automations

import (
	"context"
	"encoding/json"
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

var ErrName = errors.New("automation name required")

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) delta(ctx context.Context, wsID, id uuid.UUID, op string) {
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "automation", EntityID: id.String(), Op: op})
}

func (s *Service) Create(ctx context.Context, actor, wsID, id uuid.UUID, teamID *uuid.UUID, name string, enabled bool, trigger, conditions, actions json.RawMessage) error {
	if strings.TrimSpace(name) == "" {
		return ErrName
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		a := &models.Automation{
			ID: id, WorkspaceID: wsID, TeamID: teamID, Name: strings.TrimSpace(name), Enabled: enabled,
			Trigger: defJSON(trigger, "{}"), Conditions: conditions, Actions: defJSON(actions, "[]"),
			CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(a).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "automation", id, 0, "create", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "create")
	return nil
}

func (s *Service) Update(ctx context.Context, actor, wsID, id uuid.UUID, teamID *uuid.UUID, name string, enabled bool, trigger, conditions, actions json.RawMessage) error {
	if strings.TrimSpace(name) == "" {
		return ErrName
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.Automation)(nil)).Where("id = ?", id).
			Set("team_id = ?", teamID).Set("name = ?", strings.TrimSpace(name)).Set("enabled = ?", enabled).
			Set("trigger = ?", defJSON(trigger, "{}")).Set("conditions = ?", conditions).Set("actions = ?", defJSON(actions, "[]")).
			Set("updated_at = now()").Set("version = version + 1").Exec(ctx)
		if err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "automation", id, 0, "update", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "update")
	return nil
}

func (s *Service) SetEnabled(ctx context.Context, actor, wsID, id uuid.UUID, enabled bool) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.Automation)(nil)).Where("id = ?", id).
			Set("enabled = ?", enabled).Set("updated_at = now()").Set("version = version + 1").Exec(ctx)
		if err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "automation", id, 0, "update", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "update")
	return nil
}

func (s *Service) Delete(ctx context.Context, actor, wsID, id uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.Automation)(nil)).Where("id = ?", id).Set("deleted_at = now()").Exec(ctx)
		if err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "automation", id, 0, "delete", nil, actor)
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "delete")
	return nil
}

func defJSON(v json.RawMessage, fallback string) json.RawMessage {
	if len(v) == 0 {
		return json.RawMessage(fallback)
	}
	return v
}

// --- Engine ---

type trig struct {
	Event      string `json:"event"`      // created | status_changed | assignee_changed | priority_changed
	ToStateID  string `json:"toStateId"`  // опц.
	ToCategory string `json:"toCategory"` // опц. (started/completed/...)
}

type act struct {
	Type      string `json:"type"` // set_status | set_assignee | set_priority | add_label | post_message
	StateID   string `json:"stateId"`
	UserID    string `json:"userId"` // '' = unassign, 'me' = actor
	Priority  int    `json:"priority"`
	LabelID   string `json:"labelId"`
	ChannelID string `json:"channelId"`
	Text      string `json:"text"`
}

// RunForIssue прогоняет включённые автоматизации для задачи после события event.
func (s *Service) RunForIssue(ctx context.Context, wsID, issueID uuid.UUID, event string, actor uuid.UUID) {
	touched := map[uuid.UUID]bool{}
	var msgIDs []struct {
		ch  uuid.UUID
		mid uuid.UUID
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		iss := new(models.Issue)
		if err := tx.NewSelect().Model(iss).Column("id", "team_id", "state_id", "priority", "assignee_id", "identifier").Where("id = ?", issueID).Scan(ctx); err != nil {
			return err
		}
		var rules []models.Automation
		if err := tx.NewSelect().Model(&rules).
			Where("enabled = true").Where("deleted_at IS NULL").
			Where("team_id IS NULL OR team_id = ?", iss.TeamID).Scan(ctx); err != nil {
			return err
		}
		for _, r := range rules {
			var t trig
			_ = json.Unmarshal(r.Trigger, &t)
			if t.Event != event {
				continue
			}
			if !triggerMatches(ctx, tx, t, iss) {
				continue
			}
			var actions []act
			_ = json.Unmarshal(r.Actions, &actions)
			for _, a := range actions {
				mid, err := applyAction(ctx, tx, wsID, iss, a, actor)
				if err != nil {
					return err
				}
				if a.Type == "post_message" && mid != uuid.Nil {
					cid, _ := uuid.Parse(a.ChannelID)
					msgIDs = append(msgIDs, struct {
						ch  uuid.UUID
						mid uuid.UUID
					}{cid, mid})
				} else {
					touched[issueID] = true
				}
			}
		}
		return nil
	})
	if err != nil {
		return // автоматизации best-effort: не валим основную мутацию
	}
	for id := range touched {
		s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "issue", EntityID: id.String(), Op: "update"})
	}
	for _, m := range msgIDs {
		s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "message", EntityID: m.mid.String(), Op: "create", Actor: actor.String()})
	}
}

func triggerMatches(ctx context.Context, tx bun.Tx, t trig, iss *models.Issue) bool {
	if t.Event == "status_changed" {
		if t.ToStateID != "" {
			if iss.StateID.String() != t.ToStateID {
				return false
			}
		}
		if t.ToCategory != "" {
			var cat string
			if err := tx.NewSelect().Model((*models.WorkflowState)(nil)).Column("category").Where("id = ?", iss.StateID).Scan(ctx, &cat); err != nil || cat != t.ToCategory {
				return false
			}
		}
	}
	return true
}

// applyAction применяет действие напрямую (без issue-сервиса → нет рекурсии). Возвращает id сообщения для post_message.
func applyAction(ctx context.Context, tx bun.Tx, wsID uuid.UUID, iss *models.Issue, a act, actor uuid.UUID) (uuid.UUID, error) {
	switch a.Type {
	case "set_status":
		sid, err := uuid.Parse(a.StateID)
		if err != nil {
			return uuid.Nil, nil
		}
		var cat string
		// статус должен принадлежать команде задачи, иначе пропускаем (правило на «все команды» с чужим статусом)
		if err := tx.NewSelect().Model((*models.WorkflowState)(nil)).Column("category").Where("id = ?", sid).Where("team_id = ?", iss.TeamID).Scan(ctx, &cat); err != nil {
			return uuid.Nil, nil
		}
		q := tx.NewUpdate().Model((*models.Issue)(nil)).Where("id = ?", iss.ID).
			Set("state_id = ?", sid).Set("updated_at = now()").Set("version = version + 1")
		// зеркалит issues.stateTimestamps: started→clear completed; completed/canceled→set both; backlog/unstarted→clear оба
		switch cat {
		case "started":
			q.Set("started_at = COALESCE(started_at, now())").Set("completed_at = NULL")
		case "completed", "canceled":
			q.Set("started_at = COALESCE(started_at, now())").Set("completed_at = COALESCE(completed_at, now())")
		default: // backlog, unstarted
			q.Set("started_at = NULL").Set("completed_at = NULL")
		}
		if _, err := q.Exec(ctx); err != nil {
			return uuid.Nil, err
		}
		iss.StateID = sid
		return uuid.Nil, changelog.Record(ctx, tx, wsID, "issue", iss.ID, 0, "update", map[string]any{"stateId": sid}, actor)
	case "set_assignee":
		q := tx.NewUpdate().Model((*models.Issue)(nil)).Where("id = ?", iss.ID).Set("updated_at = now()").Set("version = version + 1")
		uid := a.UserID
		if uid == "me" {
			uid = actor.String()
		}
		if uid == "" {
			q.Set("assignee_id = NULL")
		} else {
			u, err := uuid.Parse(uid)
			if err != nil {
				return uuid.Nil, nil
			}
			q.Set("assignee_id = ?", u)
		}
		if _, err := q.Exec(ctx); err != nil {
			return uuid.Nil, err
		}
		return uuid.Nil, changelog.Record(ctx, tx, wsID, "issue", iss.ID, 0, "update", nil, actor)
	case "set_priority":
		if _, err := tx.NewUpdate().Model((*models.Issue)(nil)).Where("id = ?", iss.ID).
			Set("priority = ?", int16(a.Priority)).Set("updated_at = now()").Set("version = version + 1").Exec(ctx); err != nil {
			return uuid.Nil, err
		}
		return uuid.Nil, changelog.Record(ctx, tx, wsID, "issue", iss.ID, 0, "update", nil, actor)
	case "add_label":
		lid, err := uuid.Parse(a.LabelID)
		if err != nil {
			return uuid.Nil, nil
		}
		_, err = tx.NewInsert().Model(&models.IssueLabel{WorkspaceID: wsID, IssueID: iss.ID, LabelID: lid}).On("CONFLICT DO NOTHING").Exec(ctx)
		return uuid.Nil, err
	case "post_message":
		cid, err := uuid.Parse(a.ChannelID)
		if err != nil {
			return uuid.Nil, nil
		}
		mid, _ := uuid.NewV7()
		body := strings.ReplaceAll(a.Text, "{issue}", iss.Identifier)
		if _, err := tx.NewInsert().Model(&models.Message{
			ID: mid, WorkspaceID: wsID, ChannelID: cid, AuthorID: actor, Body: body, CreatedAt: time.Now().UTC(),
		}).Exec(ctx); err != nil {
			return uuid.Nil, err
		}
		return mid, nil
	}
	return uuid.Nil, nil
}

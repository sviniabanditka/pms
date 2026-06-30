package teams

import (
	"context"
	"errors"
	"regexp"
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

var (
	ErrInvalidKey     = errors.New("key must match ^[A-Z][A-Z0-9]{0,9}$")
	ErrInvalidName    = errors.New("name required")
	ErrKeyTaken       = errors.New("team key already taken in workspace")
	ErrParentNotFound = errors.New("parent team not found")
	ErrCycle          = errors.New("cannot move team under its own descendant")
	ErrTeamNotFound   = errors.New("team not found")
)

var keyRe = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,9}$`)

// Дефолтный набор статусов команды — по одному на каждую категорию (§teams-config).
var defaultStateDefs = []struct{ Category, Name, Color, Pos string }{
	{"backlog", "Backlog", "status-backlog", "a"},
	{"unstarted", "Todo", "status-unstarted", "b"},
	{"started", "In Progress", "status-started", "c"},
	{"completed", "Done", "status-completed", "d"},
	{"canceled", "Canceled", "status-canceled", "e"},
}

// ltreeLabel превращает UUID в валидную ltree-метку (ltree запрещает дефисы).
func ltreeLabel(id uuid.UUID) string {
	return strings.ReplaceAll(id.String(), "-", "_")
}

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

// requireManage — управлять командой (конфиг/членство) может lead команды-или-ancestor либо ws-admin.
func requireManage(ctx context.Context, tx bun.Tx, actor, teamID uuid.UUID) error {
	ok, err := authz.CanManageTeam(ctx, tx, actor, teamID)
	if err != nil {
		return err
	}
	if !ok {
		return authz.ErrForbidden
	}
	return nil
}

// requireManageN — для nullable team_id: NULL = workspace-scoped → только ws-admin.
func requireManageN(ctx context.Context, tx bun.Tx, actor uuid.UUID, teamID *uuid.UUID) error {
	if teamID == nil {
		if r, _ := authz.RoleFromContext(ctx); r.IsAdmin() {
			return nil
		}
		return authz.ErrForbidden
	}
	return requireManage(ctx, tx, actor, *teamID)
}

// stateTeam / labelTeam — резолв команды по id дочерней сущности (для гейтинга update/delete).
func stateTeam(ctx context.Context, tx bun.Tx, id uuid.UUID) (uuid.UUID, error) {
	var t uuid.UUID
	err := tx.NewSelect().Model((*models.WorkflowState)(nil)).Column("team_id").Where("id = ?", id).Scan(ctx, &t)
	return t, err
}

func labelTeam(ctx context.Context, tx bun.Tx, id uuid.UUID) (*uuid.UUID, error) {
	var t *uuid.UUID
	err := tx.NewSelect().Model((*models.Label)(nil)).Column("team_id").Where("id = ?", id).Scan(ctx, &t)
	return t, err
}

// AddMember добавляет/обновляет участника команды (lead команды-или-ancestor | ws-admin).
func (s *Service) AddMember(ctx context.Context, actor, wsID, id, teamID, userID uuid.UUID, role string) error {
	if role != "lead" && role != "member" {
		role = "member"
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireManage(ctx, tx, actor, teamID); err != nil {
			return err
		}
		ex, err := tx.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ?", wsID).Where("user_id = ?", userID).Where("status = 'active'").Exists(ctx)
		if err != nil {
			return err
		}
		if !ex {
			return ErrTeamNotFound // целевой не активный член воркспейса
		}
		tm := &models.TeamMember{ID: id, WorkspaceID: wsID, TeamID: teamID, UserID: userID, Role: role, Source: "direct", CreatedAt: now, UpdatedAt: now}
		_, e := tx.NewInsert().Model(tm).
			On("CONFLICT (team_id, user_id) DO UPDATE SET role = EXCLUDED.role, updated_at = EXCLUDED.updated_at").Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "team_member", EntityID: id.String(), Op: "create"})
	return nil
}

// RemoveMember убирает участника команды (lead команды-или-ancestor | ws-admin).
func (s *Service) RemoveMember(ctx context.Context, actor, wsID, teamID, userID uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireManage(ctx, tx, actor, teamID); err != nil {
			return err
		}
		_, e := tx.NewDelete().Model((*models.TeamMember)(nil)).Where("team_id = ?", teamID).Where("user_id = ?", userID).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "team_member", EntityID: userID.String(), Op: "delete"})
	return nil
}

// LeaveTeam — выйти из команды самому (без manage-прав).
func (s *Service) LeaveTeam(ctx context.Context, actor, wsID, teamID uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, e := tx.NewDelete().Model((*models.TeamMember)(nil)).Where("team_id = ?", teamID).Where("user_id = ?", actor).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "team_member", EntityID: actor.String(), Op: "delete"})
	return nil
}

// CreateTeam создаёт команду (path/ltree, валидный key), дефолтные статусы, lead-членство
// и записи change_log — одна транзакция под RLS-контекстом воркспейса.
var ErrStateInUse = errors.New("state has issues; reassign first")

func (s *Service) CreateWorkflowState(ctx context.Context, actor, wsID, id, teamID uuid.UUID, category, name, color string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}
	if category == "" {
		category = "unstarted"
	}
	if color == "" {
		color = "status-" + category
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireManage(ctx, tx, actor, teamID); err != nil {
			return err
		}
		var maxPos string
		_ = tx.NewSelect().Model((*models.WorkflowState)(nil)).ColumnExpr("COALESCE(MAX(position),'a')").
			Where("team_id = ?", teamID).Scan(ctx, &maxPos)
		st := &models.WorkflowState{ID: id, WorkspaceID: wsID, TeamID: teamID, Category: category, Name: strings.TrimSpace(name), ColorToken: color, Position: maxPos + "n", CreatedAt: now, UpdatedAt: now}
		_, e := tx.NewInsert().Model(st).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workflow_state", EntityID: id.String(), Op: "create"})
	return nil
}

func (s *Service) UpdateWorkflowState(ctx context.Context, actor, wsID, id uuid.UUID, name, color, category *string) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		teamID, err := stateTeam(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := requireManage(ctx, tx, actor, teamID); err != nil {
			return err
		}
		q := tx.NewUpdate().Model((*models.WorkflowState)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if name != nil {
			q.Set("name = ?", strings.TrimSpace(*name))
		}
		if color != nil {
			q.Set("color_token = ?", *color)
		}
		if category != nil {
			q.Set("category = ?", *category)
		}
		_, e := q.Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workflow_state", EntityID: id.String(), Op: "update"})
	return nil
}

func (s *Service) DeleteWorkflowState(ctx context.Context, actor, wsID, id uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		teamID, e := stateTeam(ctx, tx, id)
		if e != nil {
			return e
		}
		if e := requireManage(ctx, tx, actor, teamID); e != nil {
			return e
		}
		n, e := tx.NewSelect().Model((*models.Issue)(nil)).Where("state_id = ?", id).Where("archived_at IS NULL").Count(ctx)
		if e != nil {
			return e
		}
		if n > 0 {
			return ErrStateInUse
		}
		_, e = tx.NewDelete().Model((*models.WorkflowState)(nil)).Where("id = ?", id).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workflow_state", EntityID: id.String(), Op: "delete"})
	return nil
}

var ErrTeamHasIssues = errors.New("team has issues; archive or move them first")

func (s *Service) UpdateLabel(ctx context.Context, actor, wsID, id uuid.UUID, name, color *string) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		tID, err := labelTeam(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := requireManageN(ctx, tx, actor, tID); err != nil {
			return err
		}
		q := tx.NewUpdate().Model((*models.Label)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if name != nil {
			q.Set("name = ?", strings.TrimSpace(*name))
		}
		if color != nil {
			q.Set("color_token = ?", *color)
		}
		_, e := q.Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "label", EntityID: id.String(), Op: "update"})
	return nil
}

func (s *Service) DeleteLabel(ctx context.Context, actor, wsID, id uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		tID, err := labelTeam(ctx, tx, id)
		if err != nil {
			return err
		}
		if err := requireManageN(ctx, tx, actor, tID); err != nil {
			return err
		}
		if _, e := tx.NewDelete().Model((*models.IssueLabel)(nil)).Where("label_id = ?", id).Exec(ctx); e != nil {
			return e
		}
		_, e := tx.NewDelete().Model((*models.Label)(nil)).Where("id = ?", id).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "label", EntityID: id.String(), Op: "delete"})
	return nil
}

// DeleteTeam — мягкое удаление команды (и её под-команд по ltree-пути), если нет активных задач.
func (s *Service) DeleteTeam(ctx context.Context, actor, wsID, id uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireManage(ctx, tx, actor, id); err != nil {
			return err
		}
		t := new(models.Team)
		if err := tx.NewSelect().Model(t).Column("path").Where("id = ?", id).Scan(ctx); err != nil {
			return ErrTeamNotFound
		}
		// id целевой + потомки по ltree
		var ids []uuid.UUID
		if err := tx.NewSelect().Model((*models.Team)(nil)).Column("id").
			Where("path <@ ?", t.Path).Where("deleted_at IS NULL").Scan(ctx, &ids); err != nil {
			return err
		}
		n, err := tx.NewSelect().Model((*models.Issue)(nil)).Where("team_id IN (?)", bun.In(ids)).Where("archived_at IS NULL").Count(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			return ErrTeamHasIssues
		}
		_, err = tx.NewUpdate().Model((*models.Team)(nil)).
			Set("deleted_at = now()").Set("updated_at = now()").Set("version = version + 1").
			Where("id IN (?)", bun.In(ids)).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "team", EntityID: id.String(), Op: "delete"})
	return nil
}

// CreateLabel — лейбл (team-scoped или workspace, если teamID=nil).
func (s *Service) CreateLabel(ctx context.Context, actor, wsID, id uuid.UUID, teamID *uuid.UUID, name, color string) error {
	if strings.TrimSpace(name) == "" {
		return ErrInvalidName
	}
	if color == "" {
		color = "#6b747e"
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireManageN(ctx, tx, actor, teamID); err != nil {
			return err
		}
		l := &models.Label{ID: id, WorkspaceID: wsID, TeamID: teamID, Name: strings.TrimSpace(name), ColorToken: color, CreatedAt: now, UpdatedAt: now}
		_, err := tx.NewInsert().Model(l).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "label", EntityID: id.String(), Op: "create"})
	return nil
}

func (s *Service) CreateTeam(ctx context.Context, actor, wsID, id uuid.UUID, parentID *uuid.UUID, key, name string, private bool) (int64, error) {
	key = strings.ToUpper(strings.TrimSpace(key))
	name = strings.TrimSpace(name)
	if !keyRe.MatchString(key) {
		return 0, ErrInvalidKey
	}
	if name == "" {
		return 0, ErrInvalidName
	}
	now := time.Now().UTC()
	label := ltreeLabel(id)

	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		// root-команду создаёт только ws-admin; вложенную — lead родителя(-или-ancestor) либо ws-admin.
		if err := requireManageN(ctx, tx, actor, parentID); err != nil {
			return err
		}
		path := label
		if parentID != nil {
			parent := new(models.Team)
			if err := tx.NewSelect().Model(parent).Column("path").Where("id = ?", *parentID).Scan(ctx); err != nil {
				return ErrParentNotFound
			}
			path = parent.Path + "." + label
		}

		team := &models.Team{
			ID: id, WorkspaceID: wsID, ParentID: parentID, Path: path, Key: key, Name: name, Private: private,
			NextIssueNumber: 0, EnabledFeatures: []byte("{}"), CycleConfig: []byte("{}"),
			Settings: []byte("{}"), CreatedAt: now, UpdatedAt: now,
		}
		// path вставляется как строка — неявный каст text->ltree (миграция 0008).
		if _, err := tx.NewInsert().Model(team).Exec(ctx); err != nil {
			if strings.Contains(err.Error(), "teams_workspace_id_key_key") {
				return ErrKeyTaken
			}
			return err
		}

		states := make([]models.WorkflowState, 0, len(defaultStateDefs))
		for _, d := range defaultStateDefs {
			sid, _ := uuid.NewV7()
			states = append(states, models.WorkflowState{
				ID: sid, WorkspaceID: wsID, TeamID: id, Category: d.Category,
				Name: d.Name, ColorToken: d.Color, Position: d.Pos, CreatedAt: now, UpdatedAt: now,
			})
		}
		if _, err := tx.NewInsert().Model(&states).Exec(ctx); err != nil {
			return err
		}

		tmid, _ := uuid.NewV7()
		tm := &models.TeamMember{
			ID: tmid, WorkspaceID: wsID, TeamID: id, UserID: actor,
			Role: "lead", Source: "direct", CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(tm).Exec(ctx); err != nil {
			return err
		}

		// change_log на весь бандл создания команды — чтобы Changes(since) ничего не пропустил.
		if err := changelog.Record(ctx, tx, wsID, "team", id, 0, "create",
			map[string]any{"key": key, "name": name}, actor); err != nil {
			return err
		}
		for i := range states {
			if err := changelog.Record(ctx, tx, wsID, "workflow_state", states[i].ID, 0, "create", nil, actor); err != nil {
				return err
			}
		}
		return changelog.Record(ctx, tx, wsID, "team_member", tmid, 0, "create", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "team", EntityID: id.String(), Version: 0, Op: "create"})
	return 0, nil
}

// MoveTeam меняет родителя команды и пересчитывает ltree-path для неё и всего поддерева.
// Защита от цикла: новый родитель не может быть внутри поддерева перемещаемой команды.
func (s *Service) UpdateTeam(ctx context.Context, actor, wsID, id uuid.UUID, name *string, private *bool) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireManage(ctx, tx, actor, id); err != nil {
			return err
		}
		q := tx.NewUpdate().Model((*models.Team)(nil)).Where("id = ?", id).
			Set("updated_at = now()").Set("version = version + 1")
		if name != nil && strings.TrimSpace(*name) != "" {
			q.Set("name = ?", strings.TrimSpace(*name))
		}
		if private != nil {
			q.Set("private = ?", *private)
		}
		_, e := q.Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "team", EntityID: id.String(), Op: "update"})
	return nil
}

func (s *Service) MoveTeam(ctx context.Context, actor, wsID, id uuid.UUID, newParent *uuid.UUID) (int64, error) {
	if newParent != nil && *newParent == id {
		return 0, ErrCycle
	}
	label := ltreeLabel(id)

	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireManage(ctx, tx, actor, id); err != nil {
			return err
		}
		if newParent != nil {
			if err := requireManage(ctx, tx, actor, *newParent); err != nil {
				return err
			}
		}
		self := new(models.Team)
		if err := tx.NewSelect().Model(self).Column("path").Where("id = ?", id).Scan(ctx); err != nil {
			return ErrTeamNotFound
		}
		oldPath := self.Path

		newPath := label
		if newParent != nil {
			parent := new(models.Team)
			if err := tx.NewSelect().Model(parent).Column("path").Where("id = ?", *newParent).Scan(ctx); err != nil {
				return ErrParentNotFound
			}
			// cycle guard: parent внутри поддерева self?
			if parent.Path == oldPath || strings.HasPrefix(parent.Path, oldPath+".") {
				return ErrCycle
			}
			newPath = parent.Path + "." + label
		}

		// затронутые id (для change_log) — до апдейта.
		var affected []uuid.UUID
		if err := tx.NewSelect().Model((*models.Team)(nil)).Column("id").
			Where("path <@ ?::ltree", oldPath).Scan(ctx, &affected); err != nil {
			return err
		}

		// сама команда: parent_id + path.
		if _, err := tx.ExecContext(ctx,
			`UPDATE teams SET parent_id = ?, path = ?::ltree, updated_at = now(), version = version + 1 WHERE id = ?`,
			newParent, newPath, id); err != nil {
			return err
		}
		// потомки: заменить префикс oldPath -> newPath (parent_id не трогаем).
		if _, err := tx.ExecContext(ctx,
			`UPDATE teams SET path = (?::ltree || subpath(path, nlevel(?::ltree))), updated_at = now(), version = version + 1 WHERE path <@ ?::ltree AND id <> ?`,
			newPath, oldPath, oldPath, id); err != nil {
			return err
		}

		for _, aid := range affected {
			if err := changelog.Record(ctx, tx, wsID, "team", aid, 0, "update", map[string]any{"path": true}, actor); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "team", EntityID: id.String(), Op: "update"})
	return 0, nil
}

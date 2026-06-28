package workspaces

import (
	"encoding/json"
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/changelog"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidSlug = errors.New("slug must match ^[a-z0-9-]{2,40}$")
	ErrInvalidName = errors.New("name required")
	ErrSlugTaken   = errors.New("slug already taken")
	ErrEmail       = errors.New("email required")
)

var slugRe = regexp.MustCompile(`^[a-z0-9-]{2,40}$`)

const defaultSettings = `{"estimateScale":"points","teamNestingPolicy":"inherit-public"}`

type Service struct {
	db   *bun.DB
	pub  *realtime.Publisher
	jobs *bun.DB // pms_jobs (BYPASSRLS) — cross-tenant чтения
}

func NewService(bdb *bun.DB, pub *realtime.Publisher, jobs *bun.DB) *Service {
	return &Service{db: bdb, pub: pub, jobs: jobs}
}

// WorkspaceBrief — элемент списка «мои воркспейсы».
type WorkspaceBrief struct {
	ID   uuid.UUID `bun:"id" json:"id"`
	Slug string    `bun:"slug" json:"slug"`
	Name string    `bun:"name" json:"name"`
}

// InviteMember: если юзер с таким email существует — добавляем участником сразу; иначе создаём приглашение.
func (s *Service) InviteMember(ctx context.Context, actor, wsID, id uuid.UUID, email, role string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return ErrEmail
	}
	if role == "" {
		role = "member"
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var u models.User
		err := tx.NewSelect().Model(&u).Where("email = ?", email).Limit(1).Scan(ctx)
		if err == nil && u.ID != uuid.Nil {
			mid, _ := uuid.NewV7()
			_, e := tx.NewInsert().Model(&models.WorkspaceMember{
				ID: mid, WorkspaceID: wsID, UserID: u.ID, Role: role, Status: "active",
				SectionsConfig: json.RawMessage("{}"), JoinedAt: now,
			}).On("CONFLICT DO NOTHING").Exec(ctx)
			return e
		}
		tok, _ := uuid.NewV7()
		_, e := tx.NewInsert().Model(&models.Invitation{
			ID: id, WorkspaceID: wsID, Email: email, Role: role, TokenHash: tok.String(),
			InvitedBy: actor, ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now,
		}).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workspace_member", EntityID: id.String(), Op: "create"})
	return nil
}

// UpdateMemberRole меняет роль участника.
func (s *Service) UpdateMemberRole(ctx context.Context, actor, wsID, userID uuid.UUID, role string) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, e := tx.NewUpdate().Model((*models.WorkspaceMember)(nil)).
			Set("role = ?", role).Set("version = version + 1").
			Where("workspace_id = ?", wsID).Where("user_id = ?", userID).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workspace_member", EntityID: userID.String(), Op: "update"})
	return nil
}

// UpdateSectionsConfig — конфиг разделов рейла текущего пользователя (per-user-per-workspace).
// UpdateSettings — name и/или settings-jsonb воркспейса.
func (s *Service) UpdateSettings(ctx context.Context, actor, wsID uuid.UUID, name *string, settings []byte) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewUpdate().Model((*models.Workspace)(nil)).Where("id = ?", wsID).
			Set("updated_at = now()").Set("version = version + 1")
		if name != nil && strings.TrimSpace(*name) != "" {
			q.Set("name = ?", strings.TrimSpace(*name))
		}
		if len(settings) > 0 {
			q.Set("settings = ?", json.RawMessage(settings))
		}
		_, e := q.Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workspace", EntityID: wsID.String(), Op: "update"})
	return nil
}

func (s *Service) RemoveMember(ctx context.Context, actor, wsID, userID uuid.UUID) error {
	var memberID uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		m := new(models.WorkspaceMember)
		if err := tx.NewSelect().Model(m).Column("id").Where("workspace_id = ?", wsID).Where("user_id = ?", userID).Scan(ctx); err != nil {
			return err
		}
		memberID = m.ID
		_, e := tx.NewDelete().Model((*models.WorkspaceMember)(nil)).Where("id = ?", memberID).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workspace_member", EntityID: memberID.String(), Op: "delete"})
	return nil
}

// DeleteWorkspace — мягкое удаление (помечает deleted_at). RLS защищает тенант.
func (s *Service) DeleteWorkspace(ctx context.Context, actor, wsID uuid.UUID) error {
	return db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, e := tx.NewUpdate().Model((*models.Workspace)(nil)).
			Set("deleted_at = now()").Set("updated_at = now()").Set("version = version + 1").
			Where("id = ?", wsID).Exec(ctx)
		return e
	})
}

func (s *Service) UpdateSectionsConfig(ctx context.Context, actor, wsID uuid.UUID, config []byte) (int64, error) {
	if len(config) == 0 {
		config = []byte("{}")
	}
	var memberID uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		m := new(models.WorkspaceMember)
		if err := tx.NewSelect().Model(m).Column("id").Where("workspace_id = ?", wsID).Where("user_id = ?", actor).Scan(ctx); err != nil {
			return err
		}
		memberID = m.ID
		if _, err := tx.NewUpdate().Model((*models.WorkspaceMember)(nil)).
			Set("sections_config = ?", json.RawMessage(config)).Set("updated_at = now()").Set("version = version + 1").
			Where("id = ?", memberID).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "workspace_member", memberID, 0, "update", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workspace_member", EntityID: memberID.String(), Op: "update"})
	return 0, nil
}

// ListForUser — воркспейсы пользователя (cross-tenant через BYPASSRLS-роль,
// строго по user_id — юзер видит только свои членства).
func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceBrief, error) {
	var out []WorkspaceBrief
	err := s.jobs.NewSelect().
		TableExpr("workspaces AS w").
		ColumnExpr("w.id, w.slug, w.name").
		Join("JOIN workspace_members AS m ON m.workspace_id = w.id").
		Where("m.user_id = ?", userID).
		Where("w.deleted_at IS NULL").
		Order("w.name").
		Scan(ctx, &out)
	return out, err
}

// CreateWorkspace создаёт воркспейс + owner-членство + записи change_log (одна транзакция).
// RLS-контекст = id нового воркспейса (его же INSERT обязан удовлетворять WITH CHECK политики).
func (s *Service) CreateWorkspace(ctx context.Context, actor, id uuid.UUID, slug, name string) (int64, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	name = strings.TrimSpace(name)
	if !slugRe.MatchString(slug) {
		return 0, ErrInvalidSlug
	}
	if name == "" {
		return 0, ErrInvalidName
	}
	now := time.Now().UTC()

	err := db.WithWorkspace(ctx, s.db, id, func(ctx context.Context, tx bun.Tx) error {
		ws := &models.Workspace{
			ID: id, Slug: slug, Name: name, Settings: []byte(defaultSettings),
			CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(ws).Exec(ctx); err != nil {
			if strings.Contains(err.Error(), "workspaces_slug_key") {
				return ErrSlugTaken
			}
			return err
		}

		mid, _ := uuid.NewV7()
		mem := &models.WorkspaceMember{
			ID: mid, WorkspaceID: id, UserID: actor, Role: "owner", Status: "active",
			SectionsConfig: []byte("{}"), JoinedAt: now, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(mem).Exec(ctx); err != nil {
			return err
		}

		if err := changelog.Record(ctx, tx, id, "workspace", id, 0, "create",
			map[string]any{"slug": slug, "name": name}, actor); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, id, "workspace_member", mid, 0, "create", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	// publish после коммита (дельта видна другим клиентам).
	_ = s.pub.PublishDelta(ctx, id, realtime.Delta{EntityType: "workspace", EntityID: id.String(), Version: 0, Op: "create"})
	return 0, nil
}

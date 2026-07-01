package workspaces

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	ErrInvalidSlug    = errors.New("slug must match ^[a-z0-9-]{2,40}$")
	ErrInvalidName    = errors.New("name required")
	ErrSlugTaken      = errors.New("slug already taken")
	ErrEmail          = errors.New("email required")
	ErrForbidden      = errors.New("insufficient workspace permissions")
	ErrInviteNotFound = errors.New("invitation not found")
	ErrInviteExpired  = errors.New("invitation expired")
	ErrInviteEmail    = errors.New("invitation email mismatch")
	ErrInvalidRole    = errors.New("invalid role")
	ErrLastOwner      = errors.New("workspace must keep at least one owner")
)

// memberRole — текущая активная роль участника (ErrForbidden-обёртки делает вызывающий).
func memberRole(ctx context.Context, tx bun.Tx, wsID, userID uuid.UUID) (string, error) {
	var role string
	err := tx.NewSelect().Model((*models.WorkspaceMember)(nil)).Column("role").
		Where("workspace_id = ?", wsID).Where("user_id = ?", userID).Where("status = 'active'").Scan(ctx, &role)
	return role, err
}

// ownerCount — число активных owner'ов (last-owner guard).
func ownerCount(ctx context.Context, tx bun.Tx, wsID uuid.UUID) (int, error) {
	return tx.NewSelect().Model((*models.WorkspaceMember)(nil)).
		Where("workspace_id = ?", wsID).Where("role = 'owner'").Where("status = 'active'").Count(ctx)
}

var slugRe = regexp.MustCompile(`^[a-z0-9-]{2,40}$`)

// hashToken — токен инвайта хранится как sha256-hex (raw отдаётся только пригласившему).
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

const defaultSettings = `{"estimateScale":"points","teamNestingPolicy":"inherit-public"}`

type Service struct {
	db   *bun.DB
	pub  *realtime.Publisher
	jobs *bun.DB // pms_jobs (BYPASSRLS) — cross-tenant чтения
}

func NewService(bdb *bun.DB, pub *realtime.Publisher, jobs *bun.DB) *Service {
	return &Service{db: bdb, pub: pub, jobs: jobs}
}

// requireRole проверяет, что actor имеет одну из разрешённых ролей в воркспейсе.
func requireRole(ctx context.Context, tx bun.Tx, wsID, actor uuid.UUID, roles ...string) error {
	var role string
	if err := tx.NewSelect().Model((*models.WorkspaceMember)(nil)).Column("role").
		Where("workspace_id = ?", wsID).Where("user_id = ?", actor).Where("status = 'active'").Scan(ctx, &role); err != nil {
		return ErrForbidden
	}
	for _, r := range roles {
		if role == r {
			return nil
		}
	}
	return ErrForbidden
}

// RoleOf возвращает активную роль actor в воркспейсе; ErrForbidden, если не участник.
// Используется membership-интерсептором (один гейт на все unary-RPC с workspace_id).
func (s *Service) RoleOf(ctx context.Context, wsID, actor uuid.UUID) (string, error) {
	var role string
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewSelect().Model((*models.WorkspaceMember)(nil)).Column("role").
			Where("workspace_id = ?", wsID).Where("user_id = ?", actor).Where("status = 'active'").Scan(ctx, &role)
	})
	if err != nil {
		return "", ErrForbidden
	}
	return role, nil
}

// IsMember — actor является активным участником воркспейса (для проверок вне RPC, напр. upload).
func (s *Service) IsMember(ctx context.Context, wsID, actor uuid.UUID) (bool, error) {
	var ok bool
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		ex, err := tx.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ?", wsID).Where("user_id = ?", actor).Where("status = 'active'").Exists(ctx)
		ok = ex
		return err
	})
	return ok, err
}

// WorkspaceBrief — элемент списка «мои воркспейсы».
type WorkspaceBrief struct {
	ID   uuid.UUID `bun:"id" json:"id"`
	Slug string    `bun:"slug" json:"slug"`
	Name string    `bun:"name" json:"name"`
}

// InviteMember: существующий юзер → сразу участник (token=""); новый → invitation с токеном
// (token возвращается пригласившему для invite-ссылки /invite/<token>).
func (s *Service) InviteMember(ctx context.Context, actor, wsID, id uuid.UUID, email, role string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return "", ErrEmail
	}
	if role == "" {
		role = "member"
	}
	if !authz.Role(role).Valid() {
		return "", ErrInvalidRole
	}
	now := time.Now().UTC()
	token := ""
	existing := false
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireRole(ctx, tx, wsID, actor, "owner", "admin"); err != nil {
			return err
		}
		actorRole, err := memberRole(ctx, tx, wsID, actor)
		if err != nil {
			return ErrForbidden
		}
		// elevation-bound: нельзя пригласить с ролью выше своей.
		if !authz.Role(actorRole).AtLeast(authz.Role(role)) {
			return ErrForbidden
		}
		var u models.User
		err = tx.NewSelect().Model(&u).Where("email = ?", email).Limit(1).Scan(ctx)
		if err == nil && u.ID != uuid.Nil {
			existing = true
			mid, _ := uuid.NewV7()
			_, e := tx.NewInsert().Model(&models.WorkspaceMember{
				ID: mid, WorkspaceID: wsID, UserID: u.ID, Role: role, Status: "active",
				SectionsConfig: json.RawMessage("{}"), JoinedAt: now,
			}).On("CONFLICT DO NOTHING").Exec(ctx)
			return e
		}
		tok, _ := uuid.NewV7()
		token = tok.String()
		_, e := tx.NewInsert().Model(&models.Invitation{
			ID: id, WorkspaceID: wsID, Email: email, Role: role, TokenHash: hashToken(token),
			InvitedBy: actor, ExpiresAt: now.Add(7 * 24 * time.Hour), CreatedAt: now, UpdatedAt: now,
		}).Exec(ctx)
		return e
	})
	if err != nil {
		return "", err
	}
	if existing {
		_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workspace_member", EntityID: id.String(), Op: "create"})
	} else {
		_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "invitation", EntityID: id.String(), Op: "create"})
	}
	return token, nil
}

// AcceptInvitation: вошедший юзер принимает приглашение по токену. Токен = авторизация
// (ищем cross-tenant через jobs/BYPASSRLS, т.к. acceptor ещё не член воркспейса).
// Требуем совпадения email приглашения с email юзера. Возвращает wsID присоединённого воркспейса.
func (s *Service) AcceptInvitation(ctx context.Context, actor uuid.UUID, token string) (uuid.UUID, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return uuid.Nil, ErrInviteNotFound
	}
	now := time.Now().UTC()

	var u models.User
	if err := s.jobs.NewSelect().Model(&u).Where("id = ?", actor).Limit(1).Scan(ctx); err != nil {
		return uuid.Nil, err
	}
	var inv models.Invitation
	if err := s.jobs.NewSelect().Model(&inv).
		Where("token_hash = ?", hashToken(token)).Where("accepted_at IS NULL").Limit(1).Scan(ctx); err != nil {
		return uuid.Nil, ErrInviteNotFound
	}
	if now.After(inv.ExpiresAt) {
		return uuid.Nil, ErrInviteExpired
	}
	if !strings.EqualFold(strings.TrimSpace(u.Email), inv.Email) {
		return uuid.Nil, ErrInviteEmail
	}

	mid, _ := uuid.NewV7()
	if _, err := s.jobs.NewInsert().Model(&models.WorkspaceMember{
		ID: mid, WorkspaceID: inv.WorkspaceID, UserID: actor, Role: inv.Role, Status: "active",
		SectionsConfig: json.RawMessage("{}"), JoinedAt: now, CreatedAt: now, UpdatedAt: now,
	}).On("CONFLICT DO NOTHING").Exec(ctx); err != nil {
		return uuid.Nil, err
	}
	if _, err := s.jobs.NewUpdate().Model((*models.Invitation)(nil)).
		Set("accepted_at = ?", now).Set("updated_at = ?", now).Where("id = ?", inv.ID).Exec(ctx); err != nil {
		return uuid.Nil, err
	}
	_ = s.pub.PublishDelta(ctx, inv.WorkspaceID, realtime.Delta{EntityType: "workspace_member", EntityID: mid.String(), Op: "create"})
	_ = s.pub.PublishDelta(ctx, inv.WorkspaceID, realtime.Delta{EntityType: "invitation", EntityID: inv.ID.String(), Op: "update"})
	return inv.WorkspaceID, nil
}

// UpdateMemberRole меняет роль участника.
func (s *Service) UpdateMemberRole(ctx context.Context, actor, wsID, userID uuid.UUID, role string) error {
	if !authz.Role(role).Valid() {
		return ErrInvalidRole
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireRole(ctx, tx, wsID, actor, "owner", "admin"); err != nil {
			return err
		}
		actorRole, err := memberRole(ctx, tx, wsID, actor)
		if err != nil {
			return ErrForbidden
		}
		targetRole, err := memberRole(ctx, tx, wsID, userID)
		if err != nil {
			return err
		}
		// elevation-bound: нельзя выдать роль выше своей и нельзя трогать вышестоящего.
		if !authz.Role(actorRole).AtLeast(authz.Role(role)) || !authz.Role(actorRole).AtLeast(authz.Role(targetRole)) {
			return ErrForbidden
		}
		// last-owner guard: нельзя понизить последнего owner'а.
		if targetRole == string(authz.RoleOwner) && role != string(authz.RoleOwner) {
			n, err := ownerCount(ctx, tx, wsID)
			if err != nil {
				return err
			}
			if n <= 1 {
				return ErrLastOwner
			}
		}
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
		if err := requireRole(ctx, tx, wsID, actor, "owner", "admin"); err != nil {
			return err
		}
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
		targetRole, err := memberRole(ctx, tx, wsID, userID)
		if err != nil {
			return err
		}
		// admin/owner может убрать любого (но не вышестоящего); обычный участник — только себя (leave).
		if actor != userID {
			if err := requireRole(ctx, tx, wsID, actor, "owner", "admin"); err != nil {
				return err
			}
			actorRole, err := memberRole(ctx, tx, wsID, actor)
			if err != nil {
				return ErrForbidden
			}
			if !authz.Role(actorRole).AtLeast(authz.Role(targetRole)) {
				return ErrForbidden
			}
		}
		// last-owner guard: нельзя убрать/покинуть, если это последний owner.
		if targetRole == string(authz.RoleOwner) {
			n, err := ownerCount(ctx, tx, wsID)
			if err != nil {
				return err
			}
			if n <= 1 {
				return ErrLastOwner
			}
		}
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

// RevokeInvitation — отозвать pending-приглашение (owner/admin).
func (s *Service) RevokeInvitation(ctx context.Context, actor, wsID, id uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireRole(ctx, tx, wsID, actor, "owner", "admin"); err != nil {
			return err
		}
		_, e := tx.NewDelete().Model((*models.Invitation)(nil)).Where("id = ?", id).Where("accepted_at IS NULL").Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "invitation", EntityID: id.String(), Op: "delete"})
	return nil
}

// DeleteWorkspace — мягкое удаление (помечает deleted_at). RLS защищает тенант.
func (s *Service) DeleteWorkspace(ctx context.Context, actor, wsID uuid.UUID) error {
	return db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := requireRole(ctx, tx, wsID, actor, "owner"); err != nil {
			return err
		}
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
	out := []WorkspaceBrief{} // непустой срез → JSON `[]`, не `null` (клиент делает .length)
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

// --- Meta: компактный снапшот метаданных воркспейса для нативного клиента ---
// (states/teams/users/labels; имена для issue-строк, пикеры для create/quick-actions).
// bootstrap-стрим RN-враждебен → отдаём unary через REST.

type Meta struct {
	Teams  []MetaTeam  `json:"teams"`
	States []MetaState `json:"states"`
	Users  []MetaUser  `json:"users"`
	Labels []MetaLabel `json:"labels"`
}
type MetaTeam struct {
	ID   uuid.UUID `json:"id" bun:"id"`
	Key  string    `json:"key" bun:"key"`
	Name string    `json:"name" bun:"name"`
}
type MetaState struct {
	ID         uuid.UUID `json:"id" bun:"id"`
	TeamID     uuid.UUID `json:"teamId" bun:"team_id"`
	Category   string    `json:"category" bun:"category"`
	Name       string    `json:"name" bun:"name"`
	ColorToken string    `json:"colorToken" bun:"color_token"`
	Position   string    `json:"position" bun:"position"`
}
type MetaUser struct {
	ID    uuid.UUID `json:"id" bun:"id"`
	Name  string    `json:"name" bun:"name"`
	Email string    `json:"email" bun:"email"`
}
type MetaLabel struct {
	ID         uuid.UUID  `json:"id" bun:"id"`
	TeamID     *uuid.UUID `json:"teamId,omitempty" bun:"team_id"`
	Name       string     `json:"name" bun:"name"`
	ColorToken string     `json:"colorToken" bun:"color_token"`
}

func (s *Service) Meta(ctx context.Context, wsID uuid.UUID) (Meta, error) {
	var m Meta
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewSelect().Model((*models.Team)(nil)).Column("id", "key", "name").
			Where("deleted_at IS NULL AND archived_at IS NULL").Order("key").Scan(ctx, &m.Teams); err != nil {
			return err
		}
		if err := tx.NewSelect().Model((*models.WorkflowState)(nil)).
			Column("id", "team_id", "category", "name", "color_token", "position").
			OrderExpr("position ASC").Scan(ctx, &m.States); err != nil {
			return err
		}
		if err := tx.NewSelect().Model((*models.Label)(nil)).
			Column("id", "team_id", "name", "color_token").Scan(ctx, &m.Labels); err != nil {
			return err
		}
		return tx.NewSelect().TableExpr("users AS u").Column("u.id", "u.name", "u.email").
			Join("JOIN workspace_members wm ON wm.user_id = u.id").
			Where("wm.workspace_id = ?", wsID).Where("wm.status = 'active'").
			OrderExpr("u.name ASC").Scan(ctx, &m.Users)
	})
	return m, err
}

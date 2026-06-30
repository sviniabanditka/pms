//go:build integration

// Интеграционные тесты слоя авторизации (permissions). Поднимают реальный Postgres
// (testcontainers), прогоняют миграции, сидят воркспейс с ролями и публичными/приватными
// (вложенными) командами, и проверяют РЕАЛЬНЫЕ сервис-методы под RLS:
//
//	RoleOf (кросс-тенант), CanAccessTeam/CanManageTeam (public/private + ltree-наследование),
//	Bootstrap-фильтрацию private-команд, elevation-bound + last-owner guard, accept по хешу токена.
//
// Запуск: go test -tags=integration ./internal/authz/...
package authz_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/authz"
	"github.com/pms/backend/internal/db"
	syncsvc "github.com/pms/backend/internal/domain/sync"
	"github.com/pms/backend/internal/domain/workspaces"
	"github.com/pms/backend/internal/realtime"
)

// noopPub — Publisher с недостижимым redis: PublishDelta вернёт ошибку, а сервисы её
// игнорят (`_ = s.pub.PublishDelta(...)`) — так success-пути работают без реального redis.
func noopPub() *realtime.Publisher {
	return realtime.NewPublisher(redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: time.Second, MaxRetries: -1}))
}

const rolesSQL = `
CREATE ROLE pms_app  LOGIN PASSWORD 'pms_app';
CREATE ROLE pms_jobs LOGIN PASSWORD 'pms_jobs' BYPASSRLS;
GRANT CONNECT ON DATABASE pms TO pms_app, pms_jobs;
GRANT USAGE  ON SCHEMA public TO pms_app, pms_jobs;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pms_app, pms_jobs;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO pms_app, pms_jobs;
`

func setupPG(t *testing.T) (owner, app, jobs *bun.DB) {
	t.Helper()
	ctx := context.Background()
	c, err := tcpg.Run(ctx, "postgres:16",
		tcpg.WithDatabase("pms"), tcpg.WithUsername("postgres"), tcpg.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = c.Terminate(ctx) })

	base, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("conn string: %v", err)
	}
	appDSN := strings.Replace(base, "postgres:postgres@", "pms_app:pms_app@", 1)
	jobsDSN := strings.Replace(base, "postgres:postgres@", "pms_jobs:pms_jobs@", 1)

	raw, err := sql.Open("pg", base)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.Exec(rolesSQL); err != nil {
		t.Fatalf("roles: %v", err)
	}
	_ = raw.Close()

	if err := db.Migrate(base); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if owner, err = db.OpenBun(base); err != nil {
		t.Fatalf("open owner: %v", err)
	}
	if app, err = db.OpenBun(appDSN); err != nil {
		t.Fatalf("open app: %v", err)
	}
	if jobs, err = db.OpenBun(jobsDSN); err != nil {
		t.Fatalf("open jobs: %v", err)
	}
	t.Cleanup(func() { _ = owner.Close(); _ = app.Close(); _ = jobs.Close() })
	return
}

func label(id uuid.UUID) string { return strings.ReplaceAll(id.String(), "-", "_") }

// world — засеянный мир (как owner таблиц, RLS не действует).
type world struct {
	wsA                             uuid.UUID
	ownerU, adminU, memberU, guestU uuid.UUID
	outsiderU                       uuid.UUID
	tPub, tPriv, tParent, tChild    uuid.UUID
	issPriv                         uuid.UUID
}

func seed(t *testing.T, owner *bun.DB) world {
	t.Helper()
	ctx := context.Background()
	exec := func(q string, args ...any) {
		if _, err := owner.NewRaw(q, args...).Exec(ctx); err != nil {
			t.Fatalf("seed exec %q: %v", q, err)
		}
	}
	w := world{
		wsA: uuid.New(), ownerU: uuid.New(), adminU: uuid.New(), memberU: uuid.New(),
		guestU: uuid.New(), outsiderU: uuid.New(),
		tPub: uuid.New(), tPriv: uuid.New(), tParent: uuid.New(), tChild: uuid.New(), issPriv: uuid.New(),
	}
	usr := func(id uuid.UUID, email string) {
		exec(`INSERT INTO users (id, email, name, created_at, updated_at) VALUES (?, ?, 'U', now(), now())`, id, email)
	}
	usr(w.ownerU, "owner@e.com")
	usr(w.adminU, "admin@e.com")
	usr(w.memberU, "member@e.com")
	usr(w.guestU, "guest@e.com")
	usr(w.outsiderU, "outsider@e.com")

	exec(`INSERT INTO workspaces (id, slug, name, created_by) VALUES (?, 'ws-a', 'ws-a', ?)`, w.wsA, w.ownerU)
	mem := func(u uuid.UUID, role string) {
		exec(`INSERT INTO workspace_members (id, workspace_id, user_id, role) VALUES (?, ?, ?, ?)`, uuid.New(), w.wsA, u, role)
	}
	mem(w.ownerU, "owner")
	mem(w.adminU, "admin")
	mem(w.memberU, "member")
	mem(w.guestU, "guest")
	// outsiderU намеренно НЕ член wsA

	team := func(id uuid.UUID, key, name, path string, private bool) {
		exec(`INSERT INTO teams (id, workspace_id, path, key, name, private) VALUES (?, ?, ?, ?, ?, ?)`,
			id, w.wsA, path, key, name, private)
	}
	team(w.tPub, "PUB", "Public", label(w.tPub), false)
	team(w.tPriv, "PRIV", "Private", label(w.tPriv), true)
	team(w.tParent, "PAR", "Parent", label(w.tParent), false)
	team(w.tChild, "CHI", "Child", label(w.tParent)+"."+label(w.tChild), true) // private subteam of Parent

	// memberU — lead команды tParent (→ наследует доступ/управление к tChild по ltree).
	exec(`INSERT INTO team_members (id, workspace_id, team_id, user_id, role) VALUES (?, ?, ?, ?, 'lead')`,
		uuid.New(), w.wsA, w.tParent, w.memberU)
	// ownerU — lead tPriv (чтобы у приватной команды был хоть один член, но НЕ memberU).
	exec(`INSERT INTO team_members (id, workspace_id, team_id, user_id, role) VALUES (?, ?, ?, ?, 'lead')`,
		uuid.New(), w.wsA, w.tPriv, w.ownerU)

	// статус + issue в приватной tPriv (для проверки фильтрации bootstrap).
	sPriv := uuid.New()
	exec(`INSERT INTO workflow_states (id, workspace_id, team_id, category, name, color_token, position) VALUES (?, ?, ?, 'backlog', 'Backlog', 'status-backlog', 'a')`,
		sPriv, w.wsA, w.tPriv)
	exec(`INSERT INTO issues (id, workspace_id, team_id, number, identifier, title, state_id, created_by) VALUES (?, ?, ?, 1, 'PRIV-1', 'secret', ?, ?)`,
		w.issPriv, w.wsA, w.tPriv, sPriv, w.ownerU)
	return w
}

// accessAs вызывает CanAccessTeam в RLS-скоупе wsA с ролью role в ctx.
func accessAs(t *testing.T, app *bun.DB, ws uuid.UUID, role authz.Role, actor, team uuid.UUID) bool {
	t.Helper()
	var ok bool
	err := db.WithWorkspace(context.Background(), app, ws, func(ctx context.Context, tx bun.Tx) error {
		var e error
		ok, e = authz.CanAccessTeam(authz.WithRole(ctx, role), tx, actor, team)
		return e
	})
	if err != nil {
		t.Fatalf("CanAccessTeam: %v", err)
	}
	return ok
}

func manageAs(t *testing.T, app *bun.DB, ws uuid.UUID, role authz.Role, actor, team uuid.UUID) bool {
	t.Helper()
	var ok bool
	err := db.WithWorkspace(context.Background(), app, ws, func(ctx context.Context, tx bun.Tx) error {
		var e error
		ok, e = authz.CanManageTeam(authz.WithRole(ctx, role), tx, actor, team)
		return e
	})
	if err != nil {
		t.Fatalf("CanManageTeam: %v", err)
	}
	return ok
}

func TestRoleOf_CrossTenant(t *testing.T) {
	owner, app, jobs := setupPG(t)
	w := seed(t, owner)
	svc := workspaces.NewService(app, noopPub(), jobs)
	ctx := context.Background()

	if r, err := svc.RoleOf(ctx, w.wsA, w.memberU); err != nil || r != "member" {
		t.Fatalf("RoleOf(member) = %q,%v; want member,nil", r, err)
	}
	if r, err := svc.RoleOf(ctx, w.wsA, w.ownerU); err != nil || r != "owner" {
		t.Fatalf("RoleOf(owner) = %q,%v; want owner,nil", r, err)
	}
	// outsider не член → ErrForbidden (закрытая кросс-тенант брешь)
	if _, err := svc.RoleOf(ctx, w.wsA, w.outsiderU); err == nil {
		t.Fatal("RoleOf(outsider) must be ErrForbidden")
	}
}

func TestCanAccessTeam(t *testing.T) {
	owner, app, _ := setupPG(t)
	w := seed(t, owner)

	// member-роль в ctx → форсим DB-путь (не admin-bypass)
	if !accessAs(t, app, w.wsA, authz.RoleMember, w.memberU, w.tPub) {
		t.Error("public team must be accessible to any member")
	}
	if accessAs(t, app, w.wsA, authz.RoleMember, w.memberU, w.tPriv) {
		t.Error("private team must be hidden from non-member")
	}
	// tChild приватная, но memberU — lead tParent → наследует доступ вниз по ltree
	if !accessAs(t, app, w.wsA, authz.RoleMember, w.memberU, w.tChild) {
		t.Error("member of ancestor must access nested private subteam (ltree inheritance)")
	}
	// admin видит любую приватную (bypass)
	if !accessAs(t, app, w.wsA, authz.RoleAdmin, w.adminU, w.tPriv) {
		t.Error("admin must access any private team")
	}
}

func TestCanManageTeam(t *testing.T) {
	owner, app, _ := setupPG(t)
	w := seed(t, owner)

	if !manageAs(t, app, w.wsA, authz.RoleMember, w.memberU, w.tParent) {
		t.Error("lead must manage own team")
	}
	if !manageAs(t, app, w.wsA, authz.RoleMember, w.memberU, w.tChild) {
		t.Error("lead of ancestor must manage nested subteam (inheritance)")
	}
	if manageAs(t, app, w.wsA, authz.RoleMember, w.memberU, w.tPriv) {
		t.Error("non-lead member must NOT manage a team they don't lead")
	}
	if manageAs(t, app, w.wsA, authz.RoleMember, w.memberU, w.tPub) {
		t.Error("non-lead must NOT manage a public team")
	}
	if !manageAs(t, app, w.wsA, authz.RoleAdmin, w.adminU, w.tPriv) {
		t.Error("admin must manage any team")
	}
}

// chunkIDs возвращает множество id сущностей в чанке указанного типа.
func chunkIDs(t *testing.T, chunks []syncsvc.Chunk, entityType string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, c := range chunks {
		if c.EntityType != entityType {
			continue
		}
		var rows []map[string]any
		if err := json.Unmarshal(c.Entities, &rows); err != nil {
			t.Fatalf("unmarshal %s: %v", entityType, err)
		}
		for _, r := range rows {
			if id, ok := r["id"].(string); ok {
				out[id] = true
			}
		}
	}
	return out
}

func TestBootstrap_PrivateTeamFilter(t *testing.T) {
	owner, app, _ := setupPG(t)
	w := seed(t, owner)
	sync := syncsvc.NewService(app)
	ctx := context.Background()

	// memberU: не член tPriv → не видит её team/issue; tChild видит (наследование).
	chunks, _, err := sync.Bootstrap(ctx, w.memberU, w.wsA, nil)
	if err != nil {
		t.Fatalf("bootstrap member: %v", err)
	}
	teams := chunkIDs(t, chunks, "team")
	issues := chunkIDs(t, chunks, "issue")
	if teams[w.tPriv.String()] {
		t.Error("member must NOT receive the private team in bootstrap")
	}
	if issues[w.issPriv.String()] {
		t.Error("member must NOT receive private-team issues in bootstrap")
	}
	if !teams[w.tPub.String()] {
		t.Error("member must receive public teams")
	}
	if !teams[w.tChild.String()] {
		t.Error("member must receive nested private subteam via inheritance")
	}

	// adminU: видит всё.
	achunks, _, err := sync.Bootstrap(ctx, w.adminU, w.wsA, nil)
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	if !chunkIDs(t, achunks, "team")[w.tPriv.String()] {
		t.Error("admin must receive the private team")
	}
	if !chunkIDs(t, achunks, "issue")[w.issPriv.String()] {
		t.Error("admin must receive private-team issues")
	}
}

func TestElevationBound(t *testing.T) {
	owner, app, jobs := setupPG(t)
	w := seed(t, owner)
	svc := workspaces.NewService(app, noopPub(), jobs)
	ctx := context.Background()

	// admin не может пригласить с ролью owner (выше своей)
	if _, err := svc.InviteMember(ctx, w.adminU, w.wsA, uuid.New(), "new@e.com", "owner"); err == nil {
		t.Fatal("admin inviting an owner must be denied (elevation-bound)")
	}
	// admin не может повысить участника до owner
	if err := svc.UpdateMemberRole(ctx, w.adminU, w.wsA, w.memberU, "owner"); err == nil {
		t.Fatal("admin promoting to owner must be denied (elevation-bound)")
	}
	// но admin МОЖЕТ назначить admin (равную) — это разрешено
	if err := svc.UpdateMemberRole(ctx, w.adminU, w.wsA, w.memberU, "admin"); err != nil {
		t.Fatalf("admin assigning admin must be allowed: %v", err)
	}
}

func TestLastOwnerGuard(t *testing.T) {
	owner, app, jobs := setupPG(t)
	w := seed(t, owner)
	svc := workspaces.NewService(app, noopPub(), jobs)
	ctx := context.Background()

	// единственный owner не может понизить себя
	if err := svc.UpdateMemberRole(ctx, w.ownerU, w.wsA, w.ownerU, "member"); err == nil {
		t.Fatal("demoting the sole owner must be denied (last-owner guard)")
	}
	// и не может выйти/быть удалён
	if err := svc.RemoveMember(ctx, w.ownerU, w.wsA, w.ownerU); err == nil {
		t.Fatal("removing the sole owner must be denied (last-owner guard)")
	}
}

func TestInvitationAcceptByHashedToken(t *testing.T) {
	owner, app, jobs := setupPG(t)
	w := seed(t, owner)
	svc := workspaces.NewService(app, noopPub(), jobs)
	ctx := context.Background()

	token := uuid.New().String()
	sum := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(sum[:])
	// инвайт хранит ХЕШ токена (raw отдаётся только пригласившему)
	if _, err := owner.NewRaw(
		`INSERT INTO invitations (id, workspace_id, email, role, token_hash, invited_by, expires_at) VALUES (?, ?, 'outsider@e.com', 'member', ?, ?, now() + interval '7 days')`,
		uuid.New(), w.wsA, tokenHash, w.ownerU).Exec(ctx); err != nil {
		t.Fatalf("insert invitation: %v", err)
	}

	// неверный токен → not found (хеш не совпадает)
	if _, err := svc.AcceptInvitation(ctx, w.outsiderU, "wrong-token"); err == nil {
		t.Fatal("accept with wrong token must fail")
	}
	// верный raw-токен → accept хеширует и матчит → outsider становится членом
	gotWS, err := svc.AcceptInvitation(ctx, w.outsiderU, token)
	if err != nil {
		t.Fatalf("accept valid token: %v", err)
	}
	if gotWS != w.wsA {
		t.Fatalf("accept returned ws %v, want %v", gotWS, w.wsA)
	}
	if r, err := svc.RoleOf(ctx, w.wsA, w.outsiderU); err != nil || r != "member" {
		t.Fatalf("outsider after accept: role=%q err=%v; want member", r, err)
	}
}

//go:build integration

// Интеграционные тесты RLS-изоляции тенантов (D13). Поднимают реальный Postgres
// через testcontainers, прогоняют миграции, проверяют что pms_app видит только свой
// воркспейс и не может писать в чужой. Запуск: go test -tags=integration ./internal/db/...
package db

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/uptrace/bun"
)

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

// setupPG поднимает контейнер, создаёт роли, мигрирует. Возвращает bun-коннекты owner/app.
func setupPG(t *testing.T) (owner, app *bun.DB) {
	t.Helper()
	ctx := context.Background()
	c, err := tcpg.Run(ctx, "pgvector/pgvector:pg16",
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

	// роли — отдельным database/sql коннектом (без плейсхолдеров)
	raw, err := sql.Open("pg", base)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := raw.Exec(rolesSQL); err != nil {
		t.Fatalf("roles: %v", err)
	}
	_ = raw.Close()

	if err := Migrate(base); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if owner, err = OpenBun(base); err != nil {
		t.Fatalf("open owner: %v", err)
	}
	if app, err = OpenBun(appDSN); err != nil {
		t.Fatalf("open app: %v", err)
	}
	t.Cleanup(func() { _ = owner.Close(); _ = app.Close() })
	return
}

func TestRLSTenantIsolation(t *testing.T) {
	owner, app := setupPG(t)
	ctx := context.Background()

	// seed как owner таблиц (RLS на владельца не действует)
	uid, wsA, wsB := uuid.New(), uuid.New(), uuid.New()
	mustExec(t, ctx, owner, `INSERT INTO users (id, email, name, created_at, updated_at) VALUES (?, ?, 'U', now(), now())`, uid, "rls@example.com")
	mustExec(t, ctx, owner, `INSERT INTO workspaces (id, slug, name, created_by) VALUES (?, 'ws-a', 'ws-a', ?)`, wsA, uid)
	mustExec(t, ctx, owner, `INSERT INTO workspaces (id, slug, name, created_by) VALUES (?, 'ws-b', 'ws-b', ?)`, wsB, uid)
	ins := `INSERT INTO calendar_events (id, workspace_id, user_id, starts_at, ends_at, created_by) VALUES (?, ?, ?, now(), now(), ?)`
	mustExec(t, ctx, owner, ins, uuid.New(), wsA, uid, uid)
	mustExec(t, ctx, owner, ins, uuid.New(), wsB, uid, uid)

	// pms_app в скоупе A видит только данные A
	if n := scopedCount(t, ctx, app, wsA, "calendar_events"); n != 1 {
		t.Fatalf("scope A events: want 1, got %d", n)
	}
	if n := scopedCount(t, ctx, app, wsB, "calendar_events"); n != 1 {
		t.Fatalf("scope B events: want 1, got %d", n)
	}
	if n := scopedCount(t, ctx, app, wsA, "workspaces"); n != 1 {
		t.Fatalf("scope A workspaces: want 1, got %d", n)
	}

	// запись в чужой тенант блокируется RLS (USING = WITH CHECK при отсутствии явного)
	err := WithWorkspace(ctx, app, wsA, func(ctx context.Context, tx bun.Tx) error {
		_, e := tx.NewRaw(ins, uuid.New(), wsB, uid, uid).Exec(ctx)
		return e
	})
	if err == nil {
		t.Fatalf("cross-tenant INSERT must be denied by RLS")
	}
}

func mustExec(t *testing.T, ctx context.Context, db *bun.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.NewRaw(q, args...).Exec(ctx); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

func scopedCount(t *testing.T, ctx context.Context, db *bun.DB, ws uuid.UUID, table string) int {
	t.Helper()
	var n int
	err := WithWorkspace(ctx, db, ws, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewRaw("SELECT count(*) FROM " + table).Scan(ctx, &n)
	})
	if err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

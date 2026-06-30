//go:build integration

// Интеграционные тесты доменных сервисов под реальным Postgres+RLS. Один контейнер
// на весь пакет (TestMain), изоляция между тестами — через отдельный workspace на тест
// (RLS скоупит по ws). Management/access-операции зовём с owner-ролью в ctx (как интерсептор).
// Запуск: go test -tags=integration ./internal/integration/...
package integration

import (
	"context"
	"os"
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
	"github.com/pms/backend/internal/domain/calendar"
	"github.com/pms/backend/internal/domain/chats"
	"github.com/pms/backend/internal/domain/comments"
	"github.com/pms/backend/internal/domain/customfields"
	"github.com/pms/backend/internal/domain/cycles"
	"github.com/pms/backend/internal/domain/issues"
	"github.com/pms/backend/internal/domain/milestones"
	"github.com/pms/backend/internal/domain/teams"
	"github.com/pms/backend/internal/domain/timetracking"
	"github.com/pms/backend/internal/realtime"
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

var (
	ownerDB *bun.DB // владелец таблиц (bypass RLS) — для сидов/ассертов
	appDB   *bun.DB // pms_app (RLS) — реальный путь сервисов
	jobsDB  *bun.DB // pms_jobs (BYPASSRLS) — для сервисов с cross-tenant чтениями
	pub     *realtime.Publisher
)

type noopRedisLogger struct{}

func (noopRedisLogger) Printf(ctx context.Context, format string, v ...interface{}) {}

func TestMain(m *testing.M) {
	redis.SetLogger(noopRedisLogger{}) // глушим спам недостижимого no-op redis (PublishDelta игнорится сервисами)
	ctx := context.Background()
	c, err := tcpg.Run(ctx, "postgres:16",
		tcpg.WithDatabase("pms"), tcpg.WithUsername("postgres"), tcpg.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		panic(err)
	}
	base, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic(err)
	}
	raw, err := tcRawExec(base)
	if err != nil {
		panic(err)
	}
	_ = raw
	if err := db.Migrate(base); err != nil {
		panic(err)
	}
	if ownerDB, err = db.OpenBun(base); err != nil {
		panic(err)
	}
	appDSN := strings.Replace(base, "postgres:postgres@", "pms_app:pms_app@", 1)
	if appDB, err = db.OpenBun(appDSN); err != nil {
		panic(err)
	}
	jobsDSN := strings.Replace(base, "postgres:postgres@", "pms_jobs:pms_jobs@", 1)
	if jobsDB, err = db.OpenBun(jobsDSN); err != nil {
		panic(err)
	}
	pub = realtime.NewPublisher(redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: time.Second, MaxRetries: -1}))

	code := m.Run()
	_ = ownerDB.Close()
	_ = appDB.Close()
	_ = jobsDB.Close()
	_ = c.Terminate(ctx)
	os.Exit(code)
}

func tcRawExec(base string) (bool, error) {
	d, err := db.OpenBun(base)
	if err != nil {
		return false, err
	}
	defer d.Close()
	_, err = d.ExecContext(context.Background(), rolesSQL)
	return true, err
}

// adminCtx — ctx с owner-ролью (управление/доступ к командам проходит, как через интерсептор).
func adminCtx() context.Context { return authz.WithRole(context.Background(), authz.RoleOwner) }

// newWS создаёт изолированный воркспейс + owner-юзера. Возвращает (wsID, actor).
func newWS(t *testing.T) (uuid.UUID, uuid.UUID) {
	t.Helper()
	ws, actor := uuid.New(), uuid.New()
	mustExec(t, `INSERT INTO users (id, email, name, created_at, updated_at) VALUES (?, ?, 'U', now(), now())`, actor, actor.String()+"@e.com")
	mustExec(t, `INSERT INTO workspaces (id, slug, name, created_by) VALUES (?, ?, 'ws', ?)`, ws, "ws-"+actor.String()[:8], actor)
	mustExec(t, `INSERT INTO workspace_members (id, workspace_id, user_id, role) VALUES (?, ?, ?, 'owner')`, uuid.New(), ws, actor)
	return ws, actor
}

func newUser(t *testing.T) uuid.UUID {
	t.Helper()
	id := uuid.New()
	mustExec(t, `INSERT INTO users (id, email, name, created_at, updated_at) VALUES (?, ?, 'U', now(), now())`, id, id.String()+"@e.com")
	return id
}

func mustExec(t *testing.T, q string, args ...any) {
	t.Helper()
	if _, err := ownerDB.NewRaw(q, args...).Exec(context.Background()); err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
}

func scalar[T any](t *testing.T, q string, args ...any) T {
	t.Helper()
	var v T
	if err := ownerDB.NewRaw(q, args...).Scan(context.Background(), &v); err != nil {
		t.Fatalf("scan %q: %v", q, err)
	}
	return v
}

// makeTeam создаёт команду через сервис и возвращает её id.
func makeTeam(t *testing.T, ws, actor uuid.UUID, key string) uuid.UUID {
	t.Helper()
	svc := teams.NewService(appDB, pub)
	id := uuid.New()
	if _, err := svc.CreateTeam(adminCtx(), actor, ws, id, nil, key, key+" team", false); err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	return id
}

func TestTeams_CreateConstraints(t *testing.T) {
	ws, actor := newWS(t)
	svc := teams.NewService(appDB, pub)

	teamID := makeTeam(t, ws, actor, "ENG")

	// дефолтные статусы (5 категорий) + lead-членство создателя
	if n := scalar[int](t, `SELECT count(*) FROM workflow_states WHERE team_id = ?`, teamID); n != 5 {
		t.Errorf("default states = %d, want 5", n)
	}
	if n := scalar[int](t, `SELECT count(*) FROM team_members WHERE team_id = ? AND user_id = ? AND role = 'lead'`, teamID, actor); n != 1 {
		t.Errorf("creator must be lead, got %d", n)
	}

	// дубль ключа → ErrKeyTaken
	if _, err := svc.CreateTeam(adminCtx(), actor, ws, uuid.New(), nil, "ENG", "dup", false); err != teams.ErrKeyTaken {
		t.Errorf("duplicate key: got %v, want ErrKeyTaken", err)
	}
	// невалидный ключ (не проходит regex даже после upper-case: старт с цифры) → ErrInvalidKey
	if _, err := svc.CreateTeam(adminCtx(), actor, ws, uuid.New(), nil, "1AB", "bad", false); err != teams.ErrInvalidKey {
		t.Errorf("invalid key: got %v, want ErrInvalidKey", err)
	}
}

func TestTeams_MoveCycleGuard(t *testing.T) {
	ws, actor := newWS(t)
	svc := teams.NewService(appDB, pub)

	parent := makeTeam(t, ws, actor, "PAR")
	child := uuid.New()
	if _, err := svc.CreateTeam(adminCtx(), actor, ws, child, &parent, "CHI", "child", false); err != nil {
		t.Fatalf("create child: %v", err)
	}
	// переместить родителя под собственного потомка → ErrCycle
	if _, err := svc.MoveTeam(adminCtx(), actor, ws, parent, &child); err != teams.ErrCycle {
		t.Errorf("move parent under child: got %v, want ErrCycle", err)
	}
}

func TestTeams_DeleteWithIssuesBlocked(t *testing.T) {
	ws, actor := newWS(t)
	tsvc := teams.NewService(appDB, pub)
	isvc := issues.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "DEL")

	if _, err := isvc.CreateIssue(adminCtx(), actor, ws, teamID, uuid.New(), "live", "", 0, nil, nil); err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if err := tsvc.DeleteTeam(adminCtx(), actor, ws, teamID); err != teams.ErrTeamHasIssues {
		t.Errorf("delete team with active issue: got %v, want ErrTeamHasIssues", err)
	}
}

func TestIssues_NumberingAndSubIssue(t *testing.T) {
	ws, actor := newWS(t)
	svc := issues.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "KEY")

	a, b := uuid.New(), uuid.New()
	if _, err := svc.CreateIssue(adminCtx(), actor, ws, teamID, a, "first", "", 0, nil, nil); err != nil {
		t.Fatalf("create a: %v", err)
	}
	if _, err := svc.CreateIssue(adminCtx(), actor, ws, teamID, b, "second", "", 0, nil, nil); err != nil {
		t.Fatalf("create b: %v", err)
	}
	if id := scalar[string](t, `SELECT identifier FROM issues WHERE id = ?`, a); id != "KEY-1" {
		t.Errorf("first identifier = %q, want KEY-1", id)
	}
	if id := scalar[string](t, `SELECT identifier FROM issues WHERE id = ?`, b); id != "KEY-2" {
		t.Errorf("second identifier = %q, want KEY-2", id)
	}
	// sub-issue наследует команду родителя
	sub := uuid.New()
	if _, err := svc.CreateIssue(adminCtx(), actor, ws, uuid.Nil, sub, "child", "", 0, nil, &a); err != nil {
		t.Fatalf("create sub: %v", err)
	}
	if tid := scalar[string](t, `SELECT team_id::text FROM issues WHERE id = ?`, sub); tid != teamID.String() {
		t.Errorf("sub-issue team = %s, want %s (inherit parent)", tid, teamID)
	}
}

func TestIssues_StateTransitionTimestamps(t *testing.T) {
	ws, actor := newWS(t)
	svc := issues.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "ST")

	iss := uuid.New()
	if _, err := svc.CreateIssue(adminCtx(), actor, ws, teamID, iss, "x", "", 0, nil, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	// в started → started_at проставляется
	started := scalar[string](t, `SELECT id::text FROM workflow_states WHERE team_id = ? AND category = 'started'`, teamID)
	if _, err := svc.UpdateIssueState(adminCtx(), actor, ws, iss, started); err != nil {
		t.Fatalf("to started: %v", err)
	}
	if n := scalar[int](t, `SELECT count(*) FROM issues WHERE id = ? AND started_at IS NOT NULL AND completed_at IS NULL`, iss); n != 1 {
		t.Error("started state must set started_at and clear completed_at")
	}
	// в completed → completed_at проставляется
	done := scalar[string](t, `SELECT id::text FROM workflow_states WHERE team_id = ? AND category = 'completed'`, teamID)
	if _, err := svc.UpdateIssueState(adminCtx(), actor, ws, iss, done); err != nil {
		t.Fatalf("to done: %v", err)
	}
	if n := scalar[int](t, `SELECT count(*) FROM issues WHERE id = ? AND completed_at IS NOT NULL`, iss); n != 1 {
		t.Error("completed state must set completed_at")
	}
}

func TestIssues_Relations(t *testing.T) {
	ws, actor := newWS(t)
	svc := issues.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "REL")
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if _, err := svc.CreateIssue(adminCtx(), actor, ws, teamID, id, "x", "", 0, nil, nil); err != nil {
			t.Fatalf("create: %v", err)
		}
	}
	if _, err := svc.AddRelation(adminCtx(), actor, ws, uuid.New(), a, b, "blocks"); err != nil {
		t.Fatalf("add relation: %v", err)
	}
	if n := scalar[int](t, `SELECT count(*) FROM issue_relations WHERE source_id = ? AND target_id = ? AND type = 'blocks'`, a, b); n != 1 {
		t.Error("relation row must exist")
	}
	// связь на себя → ErrInvalidRelation
	if _, err := svc.AddRelation(adminCtx(), actor, ws, uuid.New(), a, a, "blocks"); err != issues.ErrInvalidRelation {
		t.Errorf("self relation: got %v, want ErrInvalidRelation", err)
	}
	// невалидный тип → ErrInvalidRelation
	if _, err := svc.AddRelation(adminCtx(), actor, ws, uuid.New(), a, b, "bogus"); err != issues.ErrInvalidRelation {
		t.Errorf("bad type: got %v, want ErrInvalidRelation", err)
	}
}

func TestCycles_NumberingAndDates(t *testing.T) {
	ws, actor := newWS(t)
	svc := cycles.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "CY")
	now := time.Now().UTC()

	if _, err := svc.CreateCycle(adminCtx(), actor, ws, teamID, uuid.New(), "C1", now, now.AddDate(0, 0, 14)); err != nil {
		t.Fatalf("c1: %v", err)
	}
	c2 := uuid.New()
	if _, err := svc.CreateCycle(adminCtx(), actor, ws, teamID, c2, "C2", now, now.AddDate(0, 0, 14)); err != nil {
		t.Fatalf("c2: %v", err)
	}
	if num := scalar[int](t, `SELECT number FROM cycles WHERE id = ?`, c2); num != 2 {
		t.Errorf("second cycle number = %d, want 2", num)
	}
	// конец раньше начала → ErrDates
	if _, err := svc.CreateCycle(adminCtx(), actor, ws, teamID, uuid.New(), "bad", now, now.AddDate(0, 0, -1)); err != cycles.ErrDates {
		t.Errorf("bad dates: got %v, want ErrDates", err)
	}
}

func TestComments_AuthorOnly(t *testing.T) {
	ws, actor := newWS(t)
	isvc := issues.NewService(appDB, pub)
	csvc := comments.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "CM")
	iss := uuid.New()
	if _, err := isvc.CreateIssue(adminCtx(), actor, ws, teamID, iss, "x", "", 0, nil, nil); err != nil {
		t.Fatalf("create issue: %v", err)
	}

	other := newUser(t)
	cid := uuid.New()
	ctx := adminCtx()
	if err := csvc.AddComment(ctx, actor, ws, cid, iss, "hello"); err != nil {
		t.Fatalf("add comment: %v", err)
	}
	// чужой не может править/удалять
	if err := csvc.EditComment(ctx, other, ws, cid, "hacked"); err != comments.ErrForbidden {
		t.Errorf("edit others' comment: got %v, want ErrForbidden", err)
	}
	if err := csvc.DeleteComment(ctx, other, ws, cid); err != comments.ErrForbidden {
		t.Errorf("delete others' comment: got %v, want ErrForbidden", err)
	}
	// автор может
	if err := csvc.EditComment(ctx, actor, ws, cid, "edited"); err != nil {
		t.Errorf("author edit: %v", err)
	}
}

func TestMilestones_DeleteNullsIssue(t *testing.T) {
	ws, actor := newWS(t)
	isvc := issues.NewService(appDB, pub)
	msvc := milestones.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "MS")

	iss, ms := uuid.New(), uuid.New()
	if _, err := isvc.CreateIssue(adminCtx(), actor, ws, teamID, iss, "x", "", 0, nil, nil); err != nil {
		t.Fatalf("create issue: %v", err)
	}
	if err := msvc.Create(adminCtx(), actor, ws, ms, teamID, "v1", ""); err != nil {
		t.Fatalf("create milestone: %v", err)
	}
	if _, err := isvc.SetMilestone(adminCtx(), actor, ws, iss, &ms); err != nil {
		t.Fatalf("set milestone: %v", err)
	}
	if err := msvc.Delete(adminCtx(), actor, ws, ms); err != nil {
		t.Fatalf("delete milestone: %v", err)
	}
	if n := scalar[int](t, `SELECT count(*) FROM issues WHERE id = ? AND milestone_id IS NULL`, iss); n != 1 {
		t.Error("deleting a milestone must null issues' milestone_id")
	}
}

func TestChats_PrivateAccessAndDM(t *testing.T) {
	ws, actorA := newWS(t)
	svc := chats.NewService(appDB, pub)
	ctx := context.Background()

	ch := uuid.New()
	if err := svc.CreateChannel(ctx, actorA, ws, ch, "secret", true); err != nil {
		t.Fatalf("create channel: %v", err)
	}
	actorB := newUser(t)
	// не-участник приватного канала не может писать (canAccess)
	if err := svc.SendMessage(ctx, actorB, ws, uuid.New(), ch, "sneak", nil); err != chats.ErrForbidden {
		t.Errorf("non-member SendMessage: got %v, want ErrForbidden", err)
	}
	// создатель — участник, может
	if err := svc.SendMessage(ctx, actorA, ws, uuid.New(), ch, "hi", nil); err != nil {
		t.Errorf("member SendMessage: %v", err)
	}

	// DM дедуплицируется: повторное создание между теми же юзерами → тот же канал
	dm1, err := svc.CreateDM(ctx, actorA, ws, uuid.New(), actorB)
	if err != nil {
		t.Fatalf("create dm1: %v", err)
	}
	dm2, err := svc.CreateDM(ctx, actorA, ws, uuid.New(), actorB)
	if err != nil {
		t.Fatalf("create dm2: %v", err)
	}
	if dm1 != dm2 {
		t.Errorf("DM must dedup: %v != %v", dm1, dm2)
	}
}

func TestTimeTracking_StartStop(t *testing.T) {
	ws, actor := newWS(t)
	isvc := issues.NewService(appDB, pub)
	tsvc := timetracking.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "TT")
	iss := uuid.New()
	if _, err := isvc.CreateIssue(adminCtx(), actor, ws, teamID, iss, "x", "", 0, nil, nil); err != nil {
		t.Fatalf("create issue: %v", err)
	}
	ctx := context.Background()

	if err := tsvc.StartTimer(ctx, actor, ws, uuid.New(), &iss, "work"); err != nil {
		t.Fatalf("start: %v", err)
	}
	if n := scalar[int](t, `SELECT count(*) FROM time_entries WHERE user_id = ? AND ended_at IS NULL`, actor); n != 1 {
		t.Errorf("running timer rows = %d, want 1", n)
	}
	if err := tsvc.StopTimer(ctx, actor, ws); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if n := scalar[int](t, `SELECT count(*) FROM time_entries WHERE user_id = ? AND ended_at IS NOT NULL`, actor); n != 1 {
		t.Errorf("stopped timer rows = %d, want 1", n)
	}
	// ручная запись с концом раньше начала → ErrDates
	now := time.Now().UTC()
	if err := tsvc.AddEntry(ctx, actor, ws, uuid.New(), &iss, "bad", now, now.Add(-time.Hour)); err != timetracking.ErrDates {
		t.Errorf("bad dates: got %v, want ErrDates", err)
	}
}

func TestCustomFields_CreateValidationAndArchive(t *testing.T) {
	ws, actor := newWS(t)
	svc := customfields.NewService(appDB, pub)
	teamID := makeTeam(t, ws, actor, "CF")
	ctx := adminCtx()

	fid := uuid.New()
	if _, err := svc.CreateCustomField(ctx, actor, ws, fid, &teamID, "Severity", "select", []byte(`{"options":["lo","hi"]}`)); err != nil {
		t.Fatalf("create field: %v", err)
	}
	// пустое имя → ErrName
	if _, err := svc.CreateCustomField(ctx, actor, ws, uuid.New(), &teamID, "  ", "text", nil); err != customfields.ErrName {
		t.Errorf("empty name: got %v, want ErrName", err)
	}
	// невалидный тип → ErrType
	if _, err := svc.CreateCustomField(ctx, actor, ws, uuid.New(), &teamID, "X", "bogus", nil); err != customfields.ErrType {
		t.Errorf("bad type: got %v, want ErrType", err)
	}
	// архивация
	if _, err := svc.ArchiveCustomField(ctx, actor, ws, fid); err != nil {
		t.Fatalf("archive: %v", err)
	}
	if n := scalar[int](t, `SELECT count(*) FROM custom_fields WHERE id = ? AND archived_at IS NOT NULL`, fid); n != 1 {
		t.Error("archived field must have archived_at")
	}
}

func TestCalendar_OwnerGuard(t *testing.T) {
	ws, actorA := newWS(t)
	svc := calendar.NewService(appDB, pub, jobsDB)
	ctx := context.Background()
	now := time.Now().UTC()

	ev := uuid.New()
	if err := svc.CreateEvent(ctx, actorA, ws, ev, "standup", "", "event", now, now.Add(time.Hour), false, "none", nil); err != nil {
		t.Fatalf("create event: %v", err)
	}
	actorB := newUser(t)
	// чужой не может править → ErrNotOwner
	if err := svc.UpdateEvent(ctx, actorB, ws, ev, "hijack", "", "event", now, now.Add(time.Hour), false, "none", nil); err != calendar.ErrNotOwner {
		t.Errorf("non-owner update: got %v, want ErrNotOwner", err)
	}
	// владелец может
	if err := svc.UpdateEvent(ctx, actorA, ws, ev, "standup2", "", "event", now, now.Add(time.Hour), false, "none", nil); err != nil {
		t.Errorf("owner update: %v", err)
	}
	// конец раньше начала → ErrDates
	if err := svc.CreateEvent(ctx, actorA, ws, uuid.New(), "bad", "", "event", now, now.Add(-time.Hour), false, "none", nil); err != calendar.ErrDates {
		t.Errorf("bad dates: got %v, want ErrDates", err)
	}
}

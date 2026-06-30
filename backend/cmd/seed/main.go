// Command seed наполняет БД крупным реалистичным набором данных:
// несколько воркспейсов, пересекающиеся участники, команды, сотни задач,
// майлстоуны, циклы, лейблы, подзадачи, оценки, трекинг времени, комментарии,
// каналы (открытые/приватные) с тредами и реакциями, документы, views,
// митинги и заполненные календари.
//
// Подключается под owner-ролью (MIGRATE_DATABASE_URL) → обходит RLS и пишет
// сразу во все тенанты. Запуск: `make seed` или `go run ./cmd/seed`.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/auth"
	"github.com/pms/backend/internal/config"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/frac"
	"github.com/pms/backend/internal/models"
)

// ---- конфигурация объёма (флаги) ----
var (
	flagReset = flag.Bool("reset", true, "TRUNCATE всех доменных таблиц перед сидингом")
	flagScale = flag.Float64("scale", 1.0, "множитель объёма задач/сообщений (0.1 = быстрый прогон)")
	flagSeed  = flag.Int64("seed", 42, "seed ГПСЧ (детерминизм)")
	flagPass  = flag.String("password", "password", "пароль для всех сид-юзеров")
)

func main() {
	flag.Parse()
	cfg := config.Load()
	bdb, err := db.OpenBun(cfg.MigrateDatabaseURL) // owner → bypass RLS
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer bdb.Close()
	ctx := context.Background()

	if *flagReset {
		if err := reset(ctx, bdb); err != nil {
			log.Fatalf("reset: %v", err)
		}
		log.Println("reset: домены очищены")
	}

	g := &gen{
		rng:   rand.New(rand.NewSource(*flagSeed)),
		scale: *flagScale,
		now:   time.Now().UTC(),
	}
	hash, err := auth.HashPassword(*flagPass)
	if err != nil {
		log.Fatalf("hash: %v", err)
	}
	g.pwHash = hash

	if err := g.run(ctx, bdb); err != nil {
		log.Fatalf("seed: %v", err)
	}
	g.report(ctx, bdb)
	log.Printf("seed готов · scale=%.2f · пароль всех юзеров: %q", *flagScale, *flagPass)
}

// reset вычищает доменные таблицы (порядок не важен из-за CASCADE).
func reset(ctx context.Context, bdb *bun.DB) error {
	tables := []string{
		"message_reactions", "messages", "channel_reads", "channel_members", "channels",
		"comment_reactions", "comments", "time_entries", "calendar_events",
		"meeting_participants", "meetings", "issue_relations", "issue_labels",
		"cycle_snapshots", "dashboard_widgets", "dashboards", "custom_field_values",
		"custom_fields", "automations", "notifications", "attachments", "documents",
		"views", "issues", "milestones", "cycles", "projects", "labels",
		"workflow_states", "team_members", "teams", "invitations", "workspace_members",
		"workspaces", "users", "change_log",
	}
	_, err := bdb.ExecContext(ctx, "TRUNCATE "+strings.Join(tables, ", ")+" RESTART IDENTITY CASCADE")
	return err
}

// ---- генератор ----
type gen struct {
	rng    *rand.Rand
	scale  float64
	now    time.Time
	pwHash string

	users    []models.User
	wmembers []models.WorkspaceMember
	tmembers []models.TeamMember
	ws       []models.Workspace
	teams    []models.Team
	states   []models.WorkflowState
	labels   []models.Label
	issues   []models.Issue
	ilabels  []models.IssueLabel
	rels     []models.IssueRelation
	cycles   []models.Cycle
	projects []models.Project
	mstones  []models.Milestone
	comments []models.Comment
	times    []models.TimeEntry
	docs     []models.Document
	views    []models.View
	channels []models.Channel
	chmems   []models.ChannelMember
	chreads  []models.ChannelRead
	msgs     []models.Message
	mreacts  []models.MessageReaction
	meets    []models.Meeting
	mparts   []models.MeetingParticipant
	cals     []models.CalendarEvent
	changes  []models.ChangeLog
	notifs   []models.Notification
	dashes   []models.Dashboard
	widgets  []models.DashboardWidget
	cfields  []models.CustomField
	cfvals   []models.CustomFieldValue
	autos    []models.Automation
	snaps    []models.CycleSnapshot
	creacts  []models.CommentReaction
}

func (g *gen) run(ctx context.Context, bdb *bun.DB) error {
	g.buildUsers(60)
	wsDefs := []struct{ slug, name string }{
		{"acme", "Acme Corp"}, {"globex", "Globex"}, {"initech", "Initech"},
		{"umbrella", "Umbrella Labs"}, {"hooli", "Hooli"},
	}
	for _, wd := range wsDefs {
		g.buildWorkspace(wd.slug, wd.name)
	}
	// порядок вставки уважает FK
	type batch struct {
		name string
		fn   func(context.Context, *bun.DB) error
	}
	steps := []batch{
		{"users", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.users) }},
		{"workspaces", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.ws) }},
		{"workspace_members", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.wmembers) }},
		{"teams", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.teams) }},
		{"team_members", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.tmembers) }},
		{"workflow_states", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.states) }},
		{"labels", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.labels) }},
		{"projects", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.projects) }},
		{"cycles", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.cycles) }},
		{"milestones", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.mstones) }},
		{"issues", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.issues) }},
		{"issue_labels", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.ilabels) }},
		{"issue_relations", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.rels) }},
		{"comments", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.comments) }},
		{"time_entries", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.times) }},
		{"documents", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.docs) }},
		{"views", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.views) }},
		{"channels", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.channels) }},
		{"channel_members", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.chmems) }},
		{"channel_reads", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.chreads) }},
		{"messages", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.msgs) }},
		{"message_reactions", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.mreacts) }},
		{"meetings", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.meets) }},
		{"meeting_participants", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.mparts) }},
		{"calendar_events", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.cals) }},
		{"custom_fields", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.cfields) }},
		{"custom_field_values", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.cfvals) }},
		{"automations", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.autos) }},
		{"cycle_snapshots", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.snaps) }},
		{"comment_reactions", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.creacts) }},
		{"dashboards", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.dashes) }},
		{"dashboard_widgets", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.widgets) }},
		{"change_log", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.changes) }},
		{"notifications", func(c context.Context, d *bun.DB) error { return insertAll(c, d, g.notifs) }},
	}
	for _, s := range steps {
		if err := s.fn(ctx, bdb); err != nil {
			return fmt.Errorf("insert %s: %w", s.name, err)
		}
	}
	return nil
}

// ---- сборка сущностей ----

func (g *gen) buildUsers(n int) {
	for i := 0; i < n; i++ {
		first := pick(g.rng, firstNames)
		last := pick(g.rng, lastNames)
		id := uuid.New()
		created := g.pastDays(400)
		g.users = append(g.users, models.User{
			ID:           id,
			Email:        fmt.Sprintf("%s.%s%d@example.com", strings.ToLower(first), strings.ToLower(last), i),
			Name:         first + " " + last,
			PasswordHash: &g.pwHash,
			Preferences:  jraw(`{}`),
			CreatedAt:    created,
			UpdatedAt:    created,
		})
	}
}

func (g *gen) buildWorkspace(slug, name string) {
	wsID := uuid.New()
	created := g.pastDays(380)
	// участники: 10-20 из общего пула, с пересечениями (первые 6 — «силовики» в нескольких ws)
	memberCount := g.randInt(10, 20)
	idxs := map[int]bool{}
	for i := 0; i < 6 && i < len(g.users); i++ {
		if g.chance(0.55) { // силовики часто, но не всегда → 2-3 ws
			idxs[i] = true
		}
	}
	for len(idxs) < memberCount {
		idxs[g.rng.Intn(len(g.users))] = true
	}
	var members []uuid.UUID
	first := true
	for i := range idxs {
		members = append(members, g.users[i].ID)
		role := "member"
		if first {
			role = "owner"
			first = false
		} else if g.chance(0.2) {
			role = "admin"
		} else if g.chance(0.1) {
			role = "guest"
		}
		g.wmembers = append(g.wmembers, models.WorkspaceMember{
			ID: uuid.New(), WorkspaceID: wsID, UserID: g.users[i].ID, Role: role,
			Status: "active", SectionsConfig: jraw(`{}`), JoinedAt: created, CreatedAt: created, UpdatedAt: created,
		})
	}
	owner := members[0]
	g.ws = append(g.ws, models.Workspace{
		ID: wsID, Slug: slug, Name: name, Settings: jraw(`{}`),
		CreatedBy: owner, CreatedAt: created, UpdatedAt: created,
	})

	// проекты (несколько на ws) — до команд, т.к. задачи на них ссылаются
	var projIDs []uuid.UUID
	for p := 0; p < g.randInt(2, 4); p++ {
		pid := uuid.New()
		projIDs = append(projIDs, pid)
		lead := pick(g.rng, members)
		st := pick(g.rng, []string{"planned", "in_progress", "in_progress", "paused", "completed"})
		health := pick(g.rng, []string{"on_track", "on_track", "at_risk", "off_track"})
		sd := g.pastDays(120)
		td := sd.AddDate(0, 0, g.randInt(40, 120))
		g.projects = append(g.projects, models.Project{
			ID: pid, WorkspaceID: wsID, LeadID: &lead, Name: pick(g.rng, projectNames) + " " + pick(g.rng, projectSuffix),
			Description: jraw(`{"md":""}`), Status: st, Health: &health, StartDate: &sd, TargetDate: &td,
			CreatedAt: sd, UpdatedAt: sd,
		})
	}

	// команды
	keys := sampleStr(g.rng, teamKeys, g.randInt(3, 5))
	for ti, key := range keys {
		g.buildTeam(wsID, members, projIDs, key, teamNames[ti%len(teamNames)], created)
	}

	// каналы чата
	g.buildChannels(wsID, members, owner, created)
	// документы
	g.buildDocs(wsID, members, owner, created)
	// митинги
	g.buildMeetings(wsID, members, created)
	// дашборды + виджеты (views уже созданы при сборке команд)
	g.buildDashboards(wsID, members)
	// календари участников
	for _, uid := range members {
		g.buildCalendar(wsID, uid)
	}
}

// buildCustomFields — 1-2 кастомных поля команды + значения на части задач.
func (g *gen) buildCustomFields(wsID, teamID uuid.UUID, members, issueIDs []uuid.UUID, created time.Time) {
	defs := sampleCFDefs(g.rng, g.randInt(1, 2))
	type field struct {
		id      uuid.UUID
		typ     string
		options []string
	}
	var fields []field
	for _, d := range defs {
		fid := uuid.New()
		cfg := "{}"
		if d.typ == "select" {
			cfg = `{"options":` + jval(d.options) + `}`
		}
		tid := teamID
		g.cfields = append(g.cfields, models.CustomField{
			ID: fid, WorkspaceID: wsID, TeamID: &tid, EntityType: "issue", Key: d.key, Name: d.name,
			Type: d.typ, Config: jraw(cfg), Position: ptr(frac.Between("", "")), CreatedAt: created, UpdatedAt: created,
		})
		fields = append(fields, field{fid, d.typ, d.options})
	}
	for _, iid := range issueIDs {
		if !g.chance(0.3) {
			continue
		}
		f := fields[g.rng.Intn(len(fields))]
		var val string
		switch f.typ {
		case "select":
			val = jval(pick(g.rng, f.options))
		case "number":
			val = jval(g.randInt(1, 100))
		default: // text
			val = jval(g.sentence())
		}
		g.cfvals = append(g.cfvals, models.CustomFieldValue{
			ID: uuid.New(), WorkspaceID: wsID, FieldID: f.id, IssueID: iid, Value: jraw(val),
			CreatedAt: created, UpdatedAt: created,
		})
	}
}

// buildAutomations — 1-2 правила автоматизации на команду.
func (g *gen) buildAutomations(wsID, teamID uuid.UUID, members []uuid.UUID, stateByCat map[string][]uuid.UUID, labels []uuid.UUID, created time.Time) {
	for a := 0; a < g.randInt(1, 2); a++ {
		var trigger, actions string
		switch g.rng.Intn(4) {
		case 0:
			trigger = `{"event":"issue_created"}`
			actions = fmt.Sprintf(`[{"type":"set_assignee","userId":%q}]`, pick(g.rng, members))
		case 1:
			trigger = `{"event":"status_changed","toCategory":"completed"}`
			actions = `[{"type":"set_priority","priority":0}]`
		case 2:
			trigger = `{"event":"status_changed","toCategory":"started"}`
			actions = `[{"type":"set_assignee","userId":"me"}]`
		default:
			trigger = `{"event":"issue_created"}`
			if len(labels) > 0 {
				actions = fmt.Sprintf(`[{"type":"add_label","labelId":%q}]`, pick(g.rng, labels))
			} else {
				actions = `[{"type":"set_priority","priority":2}]`
			}
		}
		tid := teamID
		g.autos = append(g.autos, models.Automation{
			ID: uuid.New(), WorkspaceID: wsID, TeamID: &tid, Name: pick(g.rng, autoNames),
			Enabled: g.chance(0.8), Trigger: jraw(trigger), Actions: jraw(actions),
			CreatedBy: pick(g.rng, members), CreatedAt: created, UpdatedAt: created,
		})
	}
}

// buildSnapshots — ежедневные burndown-снапшоты для прошедших/текущих циклов.
func (g *gen) buildSnapshots(wsID uuid.UUID, cycleIDs []uuid.UUID) {
	for _, cid := range cycleIDs {
		cy := findCycle(g.cycles, cid)
		if cy == nil || cy.StartsOn.After(g.now) {
			continue
		}
		totalCount := g.randInt(15, 40)
		totalPoints := float64(totalCount) * 2
		rc, rp := totalCount, totalPoints
		end := cy.EndsOn
		if end.After(g.now) {
			end = truncDay(g.now)
		}
		for d := cy.StartsOn; !d.After(end); d = d.AddDate(0, 0, 1) {
			g.snaps = append(g.snaps, models.CycleSnapshot{
				WorkspaceID: wsID, CycleID: cid, Day: d.Format("2006-01-02"),
				RemainingPoints: rp, RemainingCount: rc, TotalPoints: totalPoints, TotalCount: totalCount,
			})
			rc = max(0, rc-g.randInt(0, 3))
			rp = max(0, rp-float64(g.randInt(0, 6)))
		}
	}
}

// buildDashboards — 1-3 дашборда на ws с виджетами (stat/chart/view/note).
func (g *gen) buildDashboards(wsID uuid.UUID, members []uuid.UUID) {
	viewIDs := viewIDsFor(g, wsID)
	for d := 0; d < g.randInt(1, 3); d++ {
		did := uuid.New()
		owner := pick(g.rng, members)
		g.dashes = append(g.dashes, models.Dashboard{
			ID: did, WorkspaceID: wsID, Name: pick(g.rng, dashNames), ScopeKind: "workspace",
			OwnerID: &owner, CreatedAt: g.now, UpdatedAt: g.now,
		})
		x, y := 0, 0
		for w := 0; w < g.randInt(3, 6); w++ {
			typ := pick(g.rng, []string{"stat", "chart", "view", "note", "stat"})
			cfg := "{}"
			switch typ {
			case "note":
				cfg = fmt.Sprintf(`{"text":%q}`, g.sentence())
			case "stat":
				cfg = fmt.Sprintf(`{"priority":%d}`, g.randInt(0, 4))
			case "view":
				if len(viewIDs) > 0 {
					cfg = fmt.Sprintf(`{"viewId":%q}`, pick(g.rng, viewIDs))
				} else {
					typ = "chart"
				}
			}
			title := pick(g.rng, widgetTitles)
			g.widgets = append(g.widgets, models.DashboardWidget{
				ID: uuid.New(), WorkspaceID: wsID, DashboardID: did, Type: typ, Title: &title,
				X: x, Y: y, W: g.randInt(1, 2), H: 1, Config: jraw(cfg), CreatedAt: g.now, UpdatedAt: g.now,
			})
			x++
			if x >= 3 {
				x, y = 0, y+1
			}
		}
	}
}

func (g *gen) buildTeam(wsID uuid.UUID, wsMembers, projIDs []uuid.UUID, key, name string, wsCreated time.Time) {
	teamID := uuid.New()
	// 4-10 участников команды из участников ws
	tmCount := min(len(wsMembers), g.randInt(4, 10))
	teamMembers := sampleUUID(g.rng, wsMembers, tmCount)
	// число задач команды (1..count); next_issue_number = count, чтобы CreateIssue
	// в приложении продолжил нумерацию без коллизии с сид-задачами.
	count := int(float64(g.randInt(200, 500)) * g.scale)
	if count < 1 {
		count = 1
	}
	g.teams = append(g.teams, models.Team{
		ID: teamID, WorkspaceID: wsID, Path: ltreeLabel(teamID), Key: key, Name: name,
		Private: g.chance(0.15), NextIssueNumber: int64(count), EnabledFeatures: jraw(`{}`), CycleConfig: jraw(`{}`),
		Settings: jraw(`{}`), CreatedAt: wsCreated, UpdatedAt: wsCreated,
	})
	for i, uid := range teamMembers {
		role := "member"
		if i == 0 {
			role = "lead"
		}
		g.tmembers = append(g.tmembers, models.TeamMember{
			ID: uuid.New(), WorkspaceID: wsID, TeamID: teamID, UserID: uid, Role: role,
			Source: "direct", CreatedAt: wsCreated, UpdatedAt: wsCreated,
		})
	}

	// статусы команды (дефолтный набор)
	stateByCat := map[string][]uuid.UUID{}
	var allStates []uuid.UUID
	for _, sd := range defaultStates {
		sid := uuid.New()
		g.states = append(g.states, models.WorkflowState{
			ID: sid, WorkspaceID: wsID, TeamID: teamID, Category: sd.cat, Name: sd.name,
			ColorToken: "status-" + sd.cat, Position: sd.pos, CreatedAt: wsCreated, UpdatedAt: wsCreated,
		})
		stateByCat[sd.cat] = append(stateByCat[sd.cat], sid)
		allStates = append(allStates, sid)
	}

	// лейблы команды
	var teamLabels []uuid.UUID
	for _, ld := range sampleLabelDefs(g.rng, g.randInt(6, 10)) {
		lid := uuid.New()
		tid := teamID
		g.labels = append(g.labels, models.Label{
			ID: lid, WorkspaceID: wsID, TeamID: &tid, Name: ld.name, ColorToken: ld.color,
			CreatedAt: wsCreated, UpdatedAt: wsCreated,
		})
		teamLabels = append(teamLabels, lid)
	}

	// циклы (прошлый/текущий/будущий)
	var cycleIDs []uuid.UUID
	var currentCycle *uuid.UUID
	for c := 0; c < 4; c++ {
		cid := uuid.New()
		start := g.now.AddDate(0, 0, (c-2)*14) // -28,-14,0,+14 дней
		cn := c + 1
		cyName := fmt.Sprintf("Cycle %d", cn)
		g.cycles = append(g.cycles, models.Cycle{
			ID: cid, WorkspaceID: wsID, TeamID: teamID, Number: cn, Name: &cyName,
			StartsOn: truncDay(start), EndsOn: truncDay(start.AddDate(0, 0, 13)),
			CreatedAt: wsCreated, UpdatedAt: wsCreated,
		})
		cycleIDs = append(cycleIDs, cid)
		if c == 2 {
			currentCycle = &cid
		}
	}

	// майлстоуны
	var mstoneIDs []uuid.UUID
	for m := 0; m < g.randInt(3, 5); m++ {
		mid := uuid.New()
		td := g.now.AddDate(0, 0, g.randInt(-30, 90))
		status := "planned"
		if td.Before(g.now) {
			status = "done"
		} else if g.chance(0.4) {
			status = "active"
		}
		g.mstones = append(g.mstones, models.Milestone{
			ID: mid, WorkspaceID: wsID, TeamID: teamID, Name: pick(g.rng, milestoneNames),
			TargetDate: ptr(truncDay(td)), Status: status, Position: frac.Between("", ""),
			CreatedAt: wsCreated, UpdatedAt: wsCreated,
		})
		mstoneIDs = append(mstoneIDs, mid)
	}

	// задачи (count рассчитан выше, до создания команды)
	var issueIDs []uuid.UUID
	pos := frac.First()
	for n := 1; n <= count; n++ {
		iid := uuid.New()
		cat := catList[weightedIndex(g.rng, catW)]
		sid := pick(g.rng, stateByCat[cat])
		created := g.pastDays(150)
		assignee := pick(g.rng, teamMembers)
		prio := int16(weightedIndex(g.rng, prioW))
		iss := models.Issue{
			ID: iid, WorkspaceID: wsID, TeamID: teamID, Number: int64(n),
			Identifier: fmt.Sprintf("%s-%d", key, n), Title: g.issueTitle(),
			Description: jraw(jsonMD(g.paragraph(g.randInt(1, 4)))),
			StateID:     sid, Priority: prio, AssigneeID: &assignee,
			BoardPosition: pos, SubPosition: pos, CreatedBy: pick(g.rng, teamMembers),
			CreatedAt: created, UpdatedAt: created,
		}
		pos = frac.Between(pos, "")
		if g.chance(0.7) {
			est := pick(g.rng, []float64{1, 2, 3, 5, 8, 13})
			iss.Estimate = &est
		}
		// даты
		if g.chance(0.5) {
			sd := truncDay(created.AddDate(0, 0, g.randInt(0, 5)))
			iss.ScheduledDate = &sd
		}
		if g.chance(0.55) {
			dd := truncDay(g.now.AddDate(0, 0, g.randInt(-20, 40)))
			iss.DueDate = &dd
		}
		// авто-таймстемпы по категории
		switch cat {
		case "started":
			st := created.Add(time.Duration(g.randInt(1, 72)) * time.Hour)
			iss.StartedAt = &st
		case "completed":
			st := created.Add(time.Duration(g.randInt(1, 72)) * time.Hour)
			ct := st.Add(time.Duration(g.randInt(2, 240)) * time.Hour)
			iss.StartedAt, iss.CompletedAt = &st, &ct
		case "canceled":
			ct := created.Add(time.Duration(g.randInt(2, 240)) * time.Hour)
			iss.CompletedAt = &ct
		}
		// цикл / майлстоун / проект
		if g.chance(0.4) {
			cid := *currentCycle
			if g.chance(0.5) {
				cid = pick(g.rng, cycleIDs)
			}
			iss.CycleID = &cid
		}
		if len(mstoneIDs) > 0 && g.chance(0.35) {
			mid := pick(g.rng, mstoneIDs)
			iss.MilestoneID = &mid
		}
		if len(projIDs) > 0 && g.chance(0.3) {
			pid := pick(g.rng, projIDs)
			iss.ProjectID = &pid
		}
		g.issues = append(g.issues, iss)
		issueIDs = append(issueIDs, iid)

		// лейблы
		for _, lid := range sampleUUID(g.rng, teamLabels, g.randInt(0, 3)) {
			g.ilabels = append(g.ilabels, models.IssueLabel{WorkspaceID: wsID, IssueID: iid, LabelID: lid})
		}

		payload := fmt.Sprintf(`{"title":%q,"identifier":%q}`, iss.Title, iss.Identifier)

		// комментарии (на ~40% задач) + реакции + notif автору задачи
		if g.chance(0.4) {
			for c := 0; c < g.randInt(1, 6); c++ {
				ct := created.Add(time.Duration(g.randInt(1, 500)) * time.Hour)
				if ct.After(g.now) {
					ct = g.now
				}
				cmID := uuid.New()
				author := pick(g.rng, teamMembers)
				g.comments = append(g.comments, models.Comment{
					ID: cmID, WorkspaceID: wsID, IssueID: iid, AuthorID: author,
					Body: g.sentence(), CreatedAt: ct,
				})
				// реакции на комментарий
				if g.chance(0.25) {
					for _, uid := range sampleUUID(g.rng, teamMembers, g.randInt(1, 3)) {
						g.creacts = append(g.creacts, models.CommentReaction{
							ID: uuid.New(), WorkspaceID: wsID, CommentID: cmID, UserID: uid,
							Emoji: pick(g.rng, emojis), CreatedAt: ct,
						})
					}
				}
				// уведомление ассайни о комментарии
				if author != assignee && g.chance(0.5) {
					g.addNotif(wsID, assignee, "commented", iid, author, ct, payload)
				}
			}
		}

		// трекинг времени (на started/completed)
		if cat == "started" || cat == "completed" {
			for t := 0; t < g.randInt(1, 4); t++ {
				start := created.Add(time.Duration(g.randInt(1, 300)) * time.Hour)
				if start.After(g.now) {
					start = g.now.Add(-time.Hour)
				}
				end := start.Add(time.Duration(g.randInt(15, 240)) * time.Minute)
				g.times = append(g.times, models.TimeEntry{
					ID: uuid.New(), WorkspaceID: wsID, UserID: assignee, IssueID: &iid,
					Description: pick(g.rng, timeDescs), StartedAt: start, EndedAt: &end, CreatedAt: start,
				})
			}
		}

		// активность (change_log): create + апдейты + переход в done/canceled
		g.addChange(wsID, iid, 0, "create", iss.CreatedBy, created, "")
		ver := int64(1)
		last := created
		for u := 0; u < g.randInt(0, 5); u++ {
			last = last.Add(time.Duration(g.randInt(1, 200)) * time.Hour)
			if last.After(g.now) {
				last = g.now
			}
			g.addChange(wsID, iid, ver, "update", pick(g.rng, teamMembers), last, `{"fields":["`+pick(g.rng, changeFields)+`"]}`)
			ver++
		}
		if cat == "completed" || cat == "canceled" {
			when := last
			if iss.CompletedAt != nil {
				when = *iss.CompletedAt
			}
			g.addChange(wsID, iid, ver, "update", assignee, when, `{"fields":["state_id"]}`)
		}

		// нотификации (инбокс)
		if iss.CreatedBy != assignee && g.chance(0.5) {
			g.addNotif(wsID, assignee, "assigned", iid, iss.CreatedBy, created, payload)
		}
		if (cat == "started" || cat == "completed") && g.chance(0.4) {
			g.addNotif(wsID, assignee, "state_changed", iid, pick(g.rng, teamMembers), last, payload)
		}
		if g.chance(0.12) {
			g.addNotif(wsID, pick(g.rng, teamMembers), "mentioned", iid, pick(g.rng, teamMembers), created, payload)
		}
	}

	// custom fields команды + значения
	g.buildCustomFields(wsID, teamID, teamMembers, issueIDs, wsCreated)
	// automations команды
	g.buildAutomations(wsID, teamID, teamMembers, stateByCat, teamLabels, wsCreated)
	// burndown-снапшоты по циклам
	g.buildSnapshots(wsID, cycleIDs)

	// подзадачи: ~15% задач получают родителя (более ранняя задача той же команды)
	for i := range issueIDs {
		if i > 0 && g.chance(0.15) {
			parent := issueIDs[g.rng.Intn(i)]
			// найти issue в срезе и проставить parent — последние добавленные принадлежат этой команде
			setParent(g.issues, issueIDs[i], parent)
		}
	}

	// связи между задачами (blocks/related) — небольшое число
	relTypes := []string{"blocks", "related", "duplicate"}
	for r := 0; r < count/20; r++ {
		a := pick(g.rng, issueIDs)
		b := pick(g.rng, issueIDs)
		if a == b {
			continue
		}
		g.rels = append(g.rels, models.IssueRelation{
			ID: uuid.New(), WorkspaceID: wsID, SourceID: a, TargetID: b,
			Type: pick(g.rng, relTypes), CreatedAt: g.now,
		})
	}

	// views команды
	layouts := []string{"list", "board", "list", "calendar"}
	for v := 0; v < g.randInt(3, 6); v++ {
		owner := pick(g.rng, teamMembers)
		tid := teamID
		g.views = append(g.views, models.View{
			ID: uuid.New(), WorkspaceID: wsID, Name: pick(g.rng, viewNames), DataSource: "issue",
			ScopeKind: "team", TeamID: &tid, OwnerID: &owner, Shared: g.chance(0.6),
			Filter: jraw(`{}`), Layout: pick(g.rng, layouts), Position: ptr(frac.Between("", "")),
			CreatedAt: wsCreated, UpdatedAt: wsCreated,
		})
	}
	_ = allStates
}

func (g *gen) buildChannels(wsID uuid.UUID, members []uuid.UUID, owner uuid.UUID, wsCreated time.Time) {
	n := g.randInt(10, 20)
	used := map[string]bool{}
	for c := 0; c < n; c++ {
		name := pick(g.rng, channelNames)
		for used[name] {
			name = pick(g.rng, channelNames) + "-" + pick(g.rng, channelSuffix)
		}
		used[name] = true
		chID := uuid.New()
		private := g.chance(0.3)
		creator := pick(g.rng, members)
		g.channels = append(g.channels, models.Channel{
			ID: chID, WorkspaceID: wsID, Kind: "channel", Name: name, Private: private,
			CreatedBy: creator, CreatedAt: wsCreated, UpdatedAt: wsCreated,
		})
		// участники: открытый → все; приватный → подмножество
		chMembers := members
		if private {
			chMembers = sampleUUID(g.rng, members, min(len(members), g.randInt(3, 7)))
		}
		for _, uid := range chMembers {
			g.chmems = append(g.chmems, models.ChannelMember{
				ID: uuid.New(), WorkspaceID: wsID, ChannelID: chID, UserID: uid, CreatedAt: wsCreated,
			})
		}
		// сообщения
		msgCount := int(float64(g.randInt(100, 400)) * g.scale)
		var roots []uuid.UUID
		var lastTS time.Time
		for m := 0; m < msgCount; m++ {
			mid := uuid.New()
			ts := g.pastDays(120).Add(time.Duration(m) * time.Minute) // монотонно растёт внутри канала
			lastTS = ts
			msg := models.Message{
				ID: mid, WorkspaceID: wsID, ChannelID: chID, AuthorID: pick(g.rng, chMembers),
				Body: g.message(), CreatedAt: ts,
			}
			// ~20% — реплаи к недавнему корню
			if len(roots) > 0 && g.chance(0.2) {
				parent := roots[g.rng.Intn(len(roots))]
				msg.ParentID = &parent
			} else {
				roots = append(roots, mid)
				if len(roots) > 50 {
					roots = roots[len(roots)-50:] // ограничить «недавними»
				}
			}
			g.msgs = append(g.msgs, msg)
			// реакции на часть сообщений
			if g.chance(0.15) {
				for _, uid := range sampleUUID(g.rng, chMembers, g.randInt(1, 3)) {
					g.mreacts = append(g.mreacts, models.MessageReaction{
						ID: uuid.New(), WorkspaceID: wsID, MessageID: mid, UserID: uid,
						Emoji: pick(g.rng, emojis), CreatedAt: ts,
					})
				}
			}
		}
		// отметки прочтения
		for _, uid := range chMembers {
			read := lastTS
			if g.chance(0.4) { // часть участников «не дочитала» → unread-бейджи
				read = lastTS.Add(-time.Duration(g.randInt(1, 48)) * time.Hour)
			}
			g.chreads = append(g.chreads, models.ChannelRead{
				WorkspaceID: wsID, ChannelID: chID, UserID: uid, LastReadAt: read,
			})
		}
	}
}

func (g *gen) buildDocs(wsID uuid.UUID, members []uuid.UUID, owner uuid.UUID, wsCreated time.Time) {
	teamIDs := teamIDsFor(g, wsID)
	n := g.randInt(15, 40)
	var roots []uuid.UUID
	for d := 0; d < n; d++ {
		did := uuid.New()
		created := g.pastDays(120)
		var parent *uuid.UUID
		if len(roots) > 0 && g.chance(0.3) {
			p := roots[g.rng.Intn(len(roots))]
			parent = &p
		} else {
			roots = append(roots, did)
		}
		var tid *uuid.UUID
		if len(teamIDs) > 0 {
			t := pick(g.rng, teamIDs)
			tid = &t
		}
		icon := pick(g.rng, docIcons)
		g.docs = append(g.docs, models.Document{
			ID: did, WorkspaceID: wsID, ParentID: parent, ScopeKind: "team", TeamID: tid,
			Title: pick(g.rng, docTitles), Body: jraw(jsonMD(g.paragraph(g.randInt(3, 8)))),
			Icon: &icon, Position: ptr(frac.Between("", "")), CreatedBy: pick(g.rng, members),
			CreatedAt: created, UpdatedAt: created,
		})
	}
}

func (g *gen) buildMeetings(wsID uuid.UUID, members []uuid.UUID, wsCreated time.Time) {
	chIDs := channelIDsFor(g, wsID)
	n := int(float64(g.randInt(20, 50)) * g.scale)
	for m := 0; m < n; m++ {
		mid := uuid.New()
		at := g.now.AddDate(0, 0, g.randInt(-30, 30)).
			Add(time.Duration(g.randInt(8, 18)) * time.Hour).Truncate(30 * time.Minute)
		ends := at.Add(time.Duration(g.randInt(1, 4)) * 30 * time.Minute)
		status := "scheduled"
		var startedAt, endedAt *time.Time
		if at.Before(g.now) {
			status = "ended"
			startedAt, endedAt = &at, &ends
		} else if g.chance(0.05) {
			status = "live"
			startedAt = &at
		}
		var chID *uuid.UUID
		if len(chIDs) > 0 && g.chance(0.6) {
			c := pick(g.rng, chIDs)
			chID = &c
		}
		creator := pick(g.rng, members)
		g.meets = append(g.meets, models.Meeting{
			ID: mid, WorkspaceID: wsID, ChannelID: chID, Kind: "scheduled",
			Title: pick(g.rng, meetingTitles), Status: status, ScheduledAt: &at, EndsAt: &ends,
			CreatedBy: creator, StartedAt: startedAt, EndedAt: endedAt, CreatedAt: at.AddDate(0, 0, -g.randInt(1, 10)), UpdatedAt: at,
		})
		parts := sampleUUID(g.rng, members, min(len(members), g.randInt(2, 8)))
		hostSet := false
		for _, uid := range parts {
			role := "participant"
			if !hostSet {
				role = "host"
				hostSet = true
			}
			st := "invited"
			var joined, left *time.Time
			if status == "ended" {
				st = "left"
				joined, left = startedAt, endedAt
			}
			g.mparts = append(g.mparts, models.MeetingParticipant{
				ID: uuid.New(), WorkspaceID: wsID, MeetingID: mid, UserID: uid, Role: role,
				State: st, JoinedAt: joined, LeftAt: left, CreatedAt: at,
			})
		}
	}
}

func (g *gen) buildCalendar(wsID, userID uuid.UUID) {
	n := g.randInt(15, 35)
	for e := 0; e < n; e++ {
		day := g.now.AddDate(0, 0, g.randInt(-30, 30))
		kind := "event"
		if g.chance(0.12) {
			kind = "dayoff"
		}
		allDay := kind == "dayoff" || g.chance(0.15)
		var start, end time.Time
		if allDay {
			start = truncDay(day)
			end = start.AddDate(0, 0, 1)
		} else {
			h := g.randInt(8, 18)
			start = truncDay(day).Add(time.Duration(h) * time.Hour)
			end = start.Add(time.Duration(g.randInt(1, 4)) * 30 * time.Minute)
		}
		rec := "none"
		var until *time.Time
		if g.chance(0.1) {
			rec = pick(g.rng, []string{"daily", "weekly", "monthly"})
			u := start.AddDate(0, g.randInt(1, 3), 0)
			until = &u
		}
		title := pick(g.rng, calTitles)
		if kind == "dayoff" {
			title = pick(g.rng, dayoffTitles)
		}
		g.cals = append(g.cals, models.CalendarEvent{
			ID: uuid.New(), WorkspaceID: wsID, UserID: userID, Kind: kind, Title: title,
			StartsAt: start, EndsAt: end, AllDay: allDay, Recurrence: rec, RecurrenceUntil: until,
			CreatedBy: userID, CreatedAt: g.pastDays(60), UpdatedAt: g.now,
		})
	}
}

func (g *gen) report(ctx context.Context, bdb *bun.DB) {
	tables := []string{"users", "workspaces", "workspace_members", "teams", "issues",
		"comments", "comment_reactions", "time_entries", "labels", "milestones", "cycles",
		"cycle_snapshots", "documents", "views", "channels", "messages", "meetings",
		"calendar_events", "custom_fields", "custom_field_values", "automations",
		"dashboards", "dashboard_widgets", "change_log", "notifications"}
	log.Println("--- counts ---")
	for _, t := range tables {
		var c int
		_ = bdb.NewSelect().Table(t).ColumnExpr("count(*)").Scan(ctx, &c)
		log.Printf("%-20s %d", t, c)
	}
}

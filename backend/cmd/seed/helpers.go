package main

import (
	"context"
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/models"
)

// insertAll вставляет срез батчами (лимит параметров Postgres 65535).
func insertAll[T any](ctx context.Context, bdb *bun.DB, rows []T) error {
	if len(rows) == 0 {
		return nil
	}
	const chunk = 800
	for i := 0; i < len(rows); i += chunk {
		j := min(i+chunk, len(rows))
		part := rows[i:j]
		if _, err := bdb.NewInsert().Model(&part).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ---- random helpers ----
func (g *gen) randInt(lo, hi int) int { return lo + g.rng.Intn(hi-lo+1) }
func (g *gen) chance(p float64) bool   { return g.rng.Float64() < p }

// pastDays возвращает момент в пределах последних n дней (UTC).
func (g *gen) pastDays(n int) time.Time {
	d := time.Duration(g.rng.Int63n(int64(n) * int64(24*time.Hour)))
	return g.now.Add(-d)
}

func pick[T any](r *rand.Rand, s []T) T { return s[r.Intn(len(s))] }

// weightedIndex выбирает индекс пропорционально весам.
func weightedIndex(r *rand.Rand, w []int) int {
	sum := 0
	for _, x := range w {
		sum += x
	}
	n := r.Intn(sum)
	for i, x := range w {
		if n < x {
			return i
		}
		n -= x
	}
	return len(w) - 1
}

func sampleUUID(r *rand.Rand, s []uuid.UUID, n int) []uuid.UUID {
	if n >= len(s) {
		n = len(s)
	}
	idx := r.Perm(len(s))[:n]
	out := make([]uuid.UUID, 0, n)
	for _, i := range idx {
		out = append(out, s[i])
	}
	return out
}

func sampleStr(r *rand.Rand, s []string, n int) []string {
	if n >= len(s) {
		n = len(s)
	}
	idx := r.Perm(len(s))[:n]
	out := make([]string, 0, n)
	for _, i := range idx {
		out = append(out, s[i])
	}
	return out
}

type labelDef struct{ name, color string }

func sampleLabelDefs(r *rand.Rand, n int) []labelDef {
	if n >= len(labelDefs) {
		n = len(labelDefs)
	}
	idx := r.Perm(len(labelDefs))[:n]
	out := make([]labelDef, 0, n)
	for _, i := range idx {
		out = append(out, labelDefs[i])
	}
	return out
}

// ---- text generators ----
func (g *gen) issueTitle() string {
	return pick(g.rng, verbs) + " " + pick(g.rng, nouns) + " " + pick(g.rng, areas)
}

func (g *gen) sentence() string {
	n := g.randInt(6, 16)
	w := make([]string, n)
	for i := range w {
		w[i] = pick(g.rng, loremWords)
	}
	s := strings.Join(w, " ")
	return strings.ToUpper(s[:1]) + s[1:] + "."
}

func (g *gen) paragraph(sentences int) string {
	out := make([]string, sentences)
	for i := range out {
		out[i] = g.sentence()
	}
	return strings.Join(out, " ")
}

func (g *gen) message() string {
	if g.chance(0.5) {
		return pick(g.rng, chatLines)
	}
	return g.sentence()
}

// addChange добавляет запись активности (change_log) для задачи. Seq генерит БД.
func (g *gen) addChange(wsID, eid uuid.UUID, ver int64, op string, actor uuid.UUID, at time.Time, fields string) {
	var cf []byte
	if fields != "" {
		cf = []byte(fields)
	}
	a := actor
	g.changes = append(g.changes, models.ChangeLog{
		WorkspaceID: wsID, EntityType: "issue", EntityID: eid, Version: ver, Op: op,
		ChangedFields: cf, ActorID: &a, At: at,
	})
}

// addNotif добавляет нотификацию (инбокс). ~40% непрочитанные.
func (g *gen) addNotif(wsID, user uuid.UUID, typ string, eid uuid.UUID, actor uuid.UUID, at time.Time, payload string) {
	a := actor
	var read *time.Time
	if g.chance(0.6) {
		t := at.Add(time.Duration(g.randInt(1, 48)) * time.Hour)
		if t.After(g.now) {
			t = g.now
		}
		read = &t
	}
	g.notifs = append(g.notifs, models.Notification{
		ID: uuid.New(), WorkspaceID: wsID, UserID: user, Type: typ, EntityType: "issue",
		EntityID: eid, ActorID: &a, Payload: jraw(payload), ReadAt: read, At: at,
	})
}

// ---- misc ----
func ptr[T any](v T) *T { return &v }

func jraw(s string) json.RawMessage { return json.RawMessage(s) }

// jsonMD оборачивает текст в {"md": "..."} (форма description/body для FTS).
func jsonMD(md string) string {
	b, _ := json.Marshal(map[string]string{"md": md})
	return string(b)
}

func truncDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func ltreeLabel(id uuid.UUID) string {
	return strings.ReplaceAll(id.String(), "-", "_")
}

// setParent проставляет ParentID конкретной задаче в срезе (поиск с конца — быстрее).
func setParent(issues []models.Issue, child, parent uuid.UUID) {
	for i := len(issues) - 1; i >= 0; i-- {
		if issues[i].ID == child {
			issues[i].ParentID = &parent
			return
		}
	}
}

func teamIDsFor(g *gen, wsID uuid.UUID) []uuid.UUID {
	var out []uuid.UUID
	for _, t := range g.teams {
		if t.WorkspaceID == wsID {
			out = append(out, t.ID)
		}
	}
	return out
}

func channelIDsFor(g *gen, wsID uuid.UUID) []uuid.UUID {
	var out []uuid.UUID
	for _, c := range g.channels {
		if c.WorkspaceID == wsID {
			out = append(out, c.ID)
		}
	}
	return out
}

func viewIDsFor(g *gen, wsID uuid.UUID) []uuid.UUID {
	var out []uuid.UUID
	for _, v := range g.views {
		if v.WorkspaceID == wsID {
			out = append(out, v.ID)
		}
	}
	return out
}

func findCycle(cycles []models.Cycle, id uuid.UUID) *models.Cycle {
	for i := range cycles {
		if cycles[i].ID == id {
			return &cycles[i]
		}
	}
	return nil
}

// jval сериализует значение в компактный JSON (для jsonb-полей).
func jval(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

type cfDef struct {
	key, name, typ string
	options        []string
}

func sampleCFDefs(r *rand.Rand, n int) []cfDef {
	if n >= len(cfDefs) {
		n = len(cfDefs)
	}
	idx := r.Perm(len(cfDefs))[:n]
	out := make([]cfDef, 0, n)
	for _, i := range idx {
		out = append(out, cfDefs[i])
	}
	return out
}

// ---- статичные определения ----
var defaultStates = []struct{ cat, name, pos string }{
	{"backlog", "Backlog", "a"},
	{"unstarted", "Todo", "b"},
	{"started", "In Progress", "c"},
	{"completed", "Done", "d"},
	{"canceled", "Canceled", "e"},
}

// распределение задач по категориям статуса (реалистичный «хвост» бэклога).
var catList = []string{"backlog", "unstarted", "started", "completed", "canceled"}
var catW = []int{30, 25, 18, 22, 5}

// приоритеты 0..4 (none/low/medium/high/urgent).
var prioW = []int{28, 26, 24, 14, 8}

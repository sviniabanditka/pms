// Package syncsvc — bootstrap-срез и догон (Changes) для local-first синка.
package syncsvc

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/uptrace/bun"
)

var ErrForbidden = errors.New("not a workspace member")

// jarr маршалит срез в JSON-массив; nil-slice → "[]" (а не "null"), чтобы клиент всегда итерировал.
func jarr(v any) []byte {
	raw, err := json.Marshal(v)
	if err != nil || string(raw) == "null" {
		return []byte("[]")
	}
	return raw
}

// Chunk — пакет сущностей одного типа для bootstrap.
type Chunk struct {
	EntityType string
	Entities   []byte // json-массив
}

type Service struct{ db *bun.DB }

func NewService(bdb *bun.DB) *Service { return &Service{db: bdb} }

// Bootstrap отдаёт срез воркспейса + курсор. only пуст → весь срез (initial-загрузка);
// иначе — только chunk'и указанных entity_type. Второе = инкрементальный catch-up:
// WS-дельта несёт entity_type, клиент перетягивает лишь изменившиеся типы (а не весь срез).
// Семантика каждого типа идентична полному bootstrap → applyChunk+prune на клиенте те же.
func (s *Service) Bootstrap(ctx context.Context, actor, wsID uuid.UUID, only []string) ([]Chunk, int64, error) {
	var chunks []Chunk
	var cursor int64

	want := map[string]bool{}
	for _, t := range only {
		want[t] = true
	}
	wants := func(t string) bool { return len(only) == 0 || want[t] }

	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		ok, err := tx.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("user_id = ?", actor).Where("workspace_id = ?", wsID).Exists(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}

		if err := tx.NewSelect().Model((*models.ChangeLog)(nil)).
			ColumnExpr("COALESCE(MAX(seq), 0)").Scan(ctx, &cursor); err != nil {
			return err
		}

		// emit запускает builder и кладёт chunk, только если тип запрошен (или only пуст).
		emit := func(entityType string, build func() ([]byte, error)) error {
			if !wants(entityType) {
				return nil
			}
			raw, err := build()
			if err != nil {
				return err
			}
			chunks = append(chunks, Chunk{EntityType: entityType, Entities: raw})
			return nil
		}
		// table — простой builder: model-таблица с опциональными where-предикатами.
		table := func(entityType string, dest any, wheres ...func(*bun.SelectQuery)) error {
			return emit(entityType, func() ([]byte, error) {
				q := tx.NewSelect().Model(dest)
				for _, wf := range wheres {
					wf(q)
				}
				if err := q.Scan(ctx); err != nil {
					return nil, err
				}
				return jarr(dest), nil
			})
		}
		live := func(q *bun.SelectQuery) { q.Where("deleted_at IS NULL") }

		if err := table("workspace", &[]models.Workspace{}, live); err != nil {
			return err
		}
		if err := table("team", &[]models.Team{}, live); err != nil {
			return err
		}
		if err := table("workflow_state", &[]models.WorkflowState{}); err != nil {
			return err
		}
		if err := table("label", &[]models.Label{}); err != nil {
			return err
		}
		if err := table("workspace_member", &[]models.WorkspaceMember{}); err != nil {
			return err
		}
		// issues: только активные (не архив/удаление).
		if err := table("issue", &[]models.Issue{}, func(q *bun.SelectQuery) {
			q.Where("archived_at IS NULL").Where("deleted_at IS NULL")
		}); err != nil {
			return err
		}
		if err := table("issue_label", &[]models.IssueLabel{}); err != nil {
			return err
		}
		if err := table("view", &[]models.View{}, live); err != nil {
			return err
		}
		if err := table("issue_relation", &[]models.IssueRelation{}); err != nil {
			return err
		}
		if err := table("cycle", &[]models.Cycle{}); err != nil {
			return err
		}
		if err := table("custom_field", &[]models.CustomField{}, func(q *bun.SelectQuery) {
			q.Where("archived_at IS NULL")
		}); err != nil {
			return err
		}
		if err := table("custom_field_value", &[]models.CustomFieldValue{}); err != nil {
			return err
		}
		if err := table("dashboard", &[]models.Dashboard{}, live); err != nil {
			return err
		}
		if err := table("dashboard_widget", &[]models.DashboardWidget{}); err != nil {
			return err
		}
		// нотификации — только текущего пользователя (последние 100)
		if err := table("notification", &[]models.Notification{}, func(q *bun.SelectQuery) {
			q.Where("user_id = ?", actor).Order("at DESC").Limit(100)
		}); err != nil {
			return err
		}
		if err := table("document", &[]models.Document{}, live); err != nil {
			return err
		}
		// чаты: публичные каналы (не приватные) + приватные/dm, где actor участник
		if err := table("channel", &[]models.Channel{}, func(q *bun.SelectQuery) {
			q.Where("deleted_at IS NULL").
				Where("(kind = 'channel' AND private = false) OR id IN (SELECT channel_id FROM channel_members WHERE user_id = ?)", actor)
		}); err != nil {
			return err
		}
		// участники — только видимых каналов (списки приватных не утекают чужим)
		if err := table("channel_member", &[]models.ChannelMember{}, func(q *bun.SelectQuery) {
			q.Where("channel_id IN (SELECT id FROM channels WHERE deleted_at IS NULL AND ((kind = 'channel' AND private = false) OR id IN (SELECT channel_id FROM channel_members WHERE user_id = ?)))", actor)
		}); err != nil {
			return err
		}
		// непрочитанные по каналам (производный chunk: счётчик новых + last_read_at для разделителя)
		if err := emit("channel_unread", func() ([]byte, error) {
			return s.unreadChunk(ctx, tx, actor)
		}); err != nil {
			return err
		}
		// юзеры воркспейса (имена для чатов/ассайнов)
		if err := table("user", &[]models.PublicUser{}, func(q *bun.SelectQuery) {
			q.Where("id IN (SELECT user_id FROM workspace_members WHERE workspace_id = ?)", wsID)
		}); err != nil {
			return err
		}
		if err := table("invitation", &[]models.Invitation{}, func(q *bun.SelectQuery) {
			q.Where("accepted_at IS NULL")
		}); err != nil {
			return err
		}
		if err := table("milestone", &[]models.Milestone{}, live); err != nil {
			return err
		}
		// митинги: только активные (scheduled/live), завершённые не тянем.
		if err := table("meeting", &[]models.Meeting{}, func(q *bun.SelectQuery) {
			q.Where("deleted_at IS NULL").Where("status != 'ended'")
		}); err != nil {
			return err
		}
		if err := emit("meeting_participant", func() ([]byte, error) {
			var mtps []models.MeetingParticipant
			if err := tx.NewSelect().Model(&mtps).
				Join("JOIN meetings m ON m.id = mtp.meeting_id").
				Where("m.deleted_at IS NULL").Where("m.status != 'ended'").Scan(ctx); err != nil {
				return nil, err
			}
			return jarr(mtps), nil
		}); err != nil {
			return err
		}
		if err := table("calendar_event", &[]models.CalendarEvent{}, live); err != nil {
			return err
		}
		// записи учёта времени — только свои
		if err := table("time_entry", &[]models.TimeEntry{}, func(q *bun.SelectQuery) {
			q.Where("user_id = ?", actor).Where("deleted_at IS NULL")
		}); err != nil {
			return err
		}
		if err := table("automation", &[]models.Automation{}, live); err != nil {
			return err
		}
		if err := table("attachment", &[]models.Attachment{}, live); err != nil {
			return err
		}
		return table("cycle_snapshot", &[]models.CycleSnapshot{}, func(q *bun.SelectQuery) {
			q.OrderExpr("day")
		})
	})
	return chunks, cursor, err
}

// unreadChunk строит производный chunk непрочитанных каналов для actor.
func (s *Service) unreadChunk(ctx context.Context, tx bun.Tx, actor uuid.UUID) ([]byte, error) {
	type chUnread struct {
		ChannelID  string     `json:"channelId"`
		Count      int        `json:"count"`
		LastReadAt *time.Time `json:"lastReadAt"`
	}
	var reads []models.ChannelRead
	if err := tx.NewSelect().Model(&reads).Where("user_id = ?", actor).Scan(ctx); err != nil {
		return nil, err
	}
	readAt := map[uuid.UUID]*time.Time{}
	for i := range reads {
		t := reads[i].LastReadAt
		readAt[reads[i].ChannelID] = &t
	}
	var counts []struct {
		ChannelID uuid.UUID `bun:"channel_id"`
		Cnt       int       `bun:"cnt"`
	}
	if err := tx.NewSelect().TableExpr("messages AS m").
		ColumnExpr("m.channel_id").ColumnExpr("count(*) AS cnt").
		Where("m.deleted_at IS NULL").Where("m.author_id != ?", actor).
		Where("m.created_at > COALESCE((SELECT cr.last_read_at FROM channel_reads cr WHERE cr.channel_id = m.channel_id AND cr.user_id = ?), 'epoch'::timestamptz)", actor).
		Where("(EXISTS (SELECT 1 FROM channels c WHERE c.id = m.channel_id AND c.kind = 'channel' AND c.private = false AND c.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM channel_members cm WHERE cm.channel_id = m.channel_id AND cm.user_id = ?))", actor).
		GroupExpr("m.channel_id").Scan(ctx, &counts); err != nil {
		return nil, err
	}
	unreadByCh := map[uuid.UUID]int{}
	for _, c := range counts {
		unreadByCh[c.ChannelID] = c.Cnt
	}
	seen := map[uuid.UUID]bool{}
	unread := []chUnread{}
	for ch, cnt := range unreadByCh {
		unread = append(unread, chUnread{ChannelID: ch.String(), Count: cnt, LastReadAt: readAt[ch]})
		seen[ch] = true
	}
	for ch, t := range readAt {
		if !seen[ch] {
			unread = append(unread, chUnread{ChannelID: ch.String(), Count: 0, LastReadAt: t})
		}
	}
	return jarr(unread), nil
}

// Changes отдаёт дельты из change_log после курсора (догон).
func (s *Service) Changes(ctx context.Context, actor, wsID uuid.UUID, since int64) ([]models.ChangeLog, int64, error) {
	var rows []models.ChangeLog
	cursor := since

	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		ok, err := tx.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("user_id = ?", actor).Where("workspace_id = ?", wsID).Exists(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		if err := tx.NewSelect().Model(&rows).Where("seq > ?", since).Order("seq ASC").Scan(ctx); err != nil {
			return err
		}
		if n := len(rows); n > 0 {
			cursor = rows[n-1].Seq
		}
		return nil
	})
	return rows, cursor, err
}

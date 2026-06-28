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

// Bootstrap отдаёт срез воркспейса (справочники) + курсор для последующего Changes.
func (s *Service) Bootstrap(ctx context.Context, actor, wsID uuid.UUID) ([]Chunk, int64, error) {
	var chunks []Chunk
	var cursor int64

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

		// Срез: справочники, нужные клиенту сразу.
		add := func(entityType string, dest any) error {
			if err := tx.NewSelect().Model(dest).Scan(ctx); err != nil {
				return err
			}
			chunks = append(chunks, Chunk{EntityType: entityType, Entities: jarr(dest)})
			return nil
		}

		var (
			ws      []models.Workspace
			teams   []models.Team
			states  []models.WorkflowState
			labels  []models.Label
			members []models.WorkspaceMember
			iss     []models.Issue
		)
		if err := tx.NewSelect().Model(&ws).Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "workspace", Entities: jarr(ws)})
		if err := tx.NewSelect().Model(&teams).Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "team", Entities: jarr(teams)})
		if err := add("workflow_state", &states); err != nil {
			return err
		}
		if err := add("label", &labels); err != nil {
			return err
		}
		if err := add("workspace_member", &members); err != nil {
			return err
		}
		// issues: только активные (не архив/удаление).
		if err := tx.NewSelect().Model(&iss).
			Where("archived_at IS NULL").Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "issue", Entities: jarr(iss)})

		var issueLabels []models.IssueLabel
		if err := tx.NewSelect().Model(&issueLabels).Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "issue_label", Entities: jarr(issueLabels)})

		var vws []models.View
		if err := tx.NewSelect().Model(&vws).Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "view", Entities: jarr(vws)})

		var rels []models.IssueRelation
		if err := tx.NewSelect().Model(&rels).Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "issue_relation", Entities: jarr(rels)})

		var cyc []models.Cycle
		if err := tx.NewSelect().Model(&cyc).Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "cycle", Entities: jarr(cyc)})

		var cfs []models.CustomField
		if err := tx.NewSelect().Model(&cfs).Where("archived_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "custom_field", Entities: jarr(cfs)})

		var cfvs []models.CustomFieldValue
		if err := tx.NewSelect().Model(&cfvs).Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "custom_field_value", Entities: jarr(cfvs)})

		var dsh []models.Dashboard
		if err := tx.NewSelect().Model(&dsh).Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "dashboard", Entities: jarr(dsh)})

		var dws []models.DashboardWidget
		if err := tx.NewSelect().Model(&dws).Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "dashboard_widget", Entities: jarr(dws)})

		// нотификации — только текущего пользователя
		var notifs []models.Notification
		if err := tx.NewSelect().Model(&notifs).Where("user_id = ?", actor).Order("at DESC").Limit(100).Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "notification", Entities: jarr(notifs)})

		var docsList []models.Document
		if err := tx.NewSelect().Model(&docsList).Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "document", Entities: jarr(docsList)})

		// чаты: публичные каналы + dm, где actor участник
		var channels []models.Channel
		if err := tx.NewSelect().Model(&channels).
			Where("deleted_at IS NULL").
			Where("kind = 'channel' OR id IN (SELECT channel_id FROM channel_members WHERE user_id = ?)", actor).
			Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "channel", Entities: jarr(channels)})

		var chMembers []models.ChannelMember
		if err := tx.NewSelect().Model(&chMembers).Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "channel_member", Entities: jarr(chMembers)})

		// непрочитанные по каналам для actor: счётчик новых (не своих) сообщений + last_read_at для разделителя
		type chUnread struct {
			ChannelID  string     `json:"channelId"`
			Count      int        `json:"count"`
			LastReadAt *time.Time `json:"lastReadAt"`
		}
		var reads []models.ChannelRead
		if err := tx.NewSelect().Model(&reads).Where("user_id = ?", actor).Scan(ctx); err != nil {
			return err
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
			Where("(EXISTS (SELECT 1 FROM channels c WHERE c.id = m.channel_id AND c.kind = 'channel' AND c.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM channel_members cm WHERE cm.channel_id = m.channel_id AND cm.user_id = ?))", actor).
			GroupExpr("m.channel_id").Scan(ctx, &counts); err != nil {
			return err
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
		chunks = append(chunks, Chunk{EntityType: "channel_unread", Entities: jarr(unread)})

		// юзеры воркспейса (имена для чатов/ассайнов)
		var users []models.PublicUser
		if err := tx.NewSelect().Model(&users).
			Where("id IN (SELECT user_id FROM workspace_members WHERE workspace_id = ?)", wsID).
			Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "user", Entities: jarr(users)})

		var invites []models.Invitation
		if err := tx.NewSelect().Model(&invites).Where("accepted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "invitation", Entities: jarr(invites)})

		var mss []models.Milestone
		if err := tx.NewSelect().Model(&mss).Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "milestone", Entities: jarr(mss)})

		// митинги: только активные (scheduled/live), завершённые не тянем в bootstrap.
		var mts []models.Meeting
		if err := tx.NewSelect().Model(&mts).Where("deleted_at IS NULL").Where("status != 'ended'").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "meeting", Entities: jarr(mts)})

		var mtps []models.MeetingParticipant
		if err := tx.NewSelect().Model(&mtps).
			Join("JOIN meetings m ON m.id = mtp.meeting_id").
			Where("m.deleted_at IS NULL").Where("m.status != 'ended'").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "meeting_participant", Entities: jarr(mtps)})

		var atts []models.Attachment
		if err := tx.NewSelect().Model(&atts).Where("deleted_at IS NULL").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "attachment", Entities: jarr(atts)})

		var snaps []models.CycleSnapshot
		if err := tx.NewSelect().Model(&snaps).OrderExpr("day").Scan(ctx); err != nil {
			return err
		}
		chunks = append(chunks, Chunk{EntityType: "cycle_snapshot", Entities: jarr(snaps)})
		return nil
	})
	return chunks, cursor, err
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

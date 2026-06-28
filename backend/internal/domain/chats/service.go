package chats

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/domain/notifications"
	"github.com/pms/backend/internal/mentions"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var (
	ErrEmpty     = errors.New("empty message")
	ErrForbidden = errors.New("not your message")
)

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) notify(ctx context.Context, wsID uuid.UUID, channelID, op string) {
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "message", EntityID: channelID, Op: op})
}

func (s *Service) CreateChannel(ctx context.Context, actor, wsID, id uuid.UUID, name string, private bool) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		ch := &models.Channel{ID: id, WorkspaceID: wsID, Kind: "channel", Name: strings.TrimSpace(name), Private: private, CreatedBy: actor, CreatedAt: now, UpdatedAt: now}
		if _, err := tx.NewInsert().Model(ch).Exec(ctx); err != nil {
			return err
		}
		mid, _ := uuid.NewV7()
		_, err := tx.NewInsert().Model(&models.ChannelMember{ID: mid, WorkspaceID: wsID, ChannelID: id, UserID: actor, CreatedAt: now}).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "channel", EntityID: id.String(), Op: "create"})
	return nil
}

// isMember — actor состоит в канале (право управлять участниками).
func (s *Service) isMember(ctx context.Context, tx bun.Tx, channelID, userID uuid.UUID) (bool, error) {
	return tx.NewSelect().Model((*models.ChannelMember)(nil)).
		Where("channel_id = ?", channelID).Where("user_id = ?", userID).Exists(ctx)
}

// AddChannelMember — добавить участника (только член канала может). Идемпотентно.
func (s *Service) AddChannelMember(ctx context.Context, actor, wsID, channelID, userID uuid.UUID) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		ok, err := s.isMember(ctx, tx, channelID, actor)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		mid, _ := uuid.NewV7()
		_, err = tx.NewInsert().Model(&models.ChannelMember{ID: mid, WorkspaceID: wsID, ChannelID: channelID, UserID: userID, CreatedAt: now}).
			On("CONFLICT (channel_id, user_id) DO NOTHING").Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "channel", EntityID: channelID.String(), Op: "update"})
	return nil
}

// ArchiveChannel — архивировать канал (только член; для публичного член = создатель). Скрывает из списков.
func (s *Service) ArchiveChannel(ctx context.Context, actor, wsID, channelID uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		ok, err := s.isMember(ctx, tx, channelID, actor)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		_, err = tx.NewUpdate().Model((*models.Channel)(nil)).
			Set("archived_at = now()").Set("updated_at = now()").Where("id = ?", channelID).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "channel", EntityID: channelID.String(), Op: "update"})
	return nil
}

// RemoveChannelMember — убрать участника (только член канала может). Leave = убрать себя.
func (s *Service) RemoveChannelMember(ctx context.Context, actor, wsID, channelID, userID uuid.UUID) error {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		ok, err := s.isMember(ctx, tx, channelID, actor)
		if err != nil {
			return err
		}
		if !ok {
			return ErrForbidden
		}
		_, err = tx.NewDelete().Model((*models.ChannelMember)(nil)).
			Where("channel_id = ?", channelID).Where("user_id = ?", userID).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "channel", EntityID: channelID.String(), Op: "update"})
	return nil
}

func (s *Service) CreateDM(ctx context.Context, actor, wsID, id, other uuid.UUID) (uuid.UUID, error) {
	result := id
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var existing uuid.UUID
		err := tx.NewSelect().Model((*models.ChannelMember)(nil)).
			ColumnExpr("chm.channel_id").
			Join("JOIN channels ch ON ch.id = chm.channel_id").
			Where("ch.kind = 'dm'").Where("chm.user_id IN (?, ?)", actor, other).
			GroupExpr("chm.channel_id").
			Having("COUNT(DISTINCT chm.user_id) = 2").
			Limit(1).Scan(ctx, &existing)
		if err == nil && existing != uuid.Nil {
			result = existing
			return nil
		}
		ch := &models.Channel{ID: id, WorkspaceID: wsID, Kind: "dm", CreatedBy: actor, CreatedAt: now, UpdatedAt: now}
		if _, err := tx.NewInsert().Model(ch).Exec(ctx); err != nil {
			return err
		}
		for _, u := range []uuid.UUID{actor, other} {
			mid, _ := uuid.NewV7()
			if _, err := tx.NewInsert().Model(&models.ChannelMember{ID: mid, WorkspaceID: wsID, ChannelID: id, UserID: u, CreatedAt: now}).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "channel", EntityID: result.String(), Op: "create"})
	return result, nil
}

func (s *Service) SendMessage(ctx context.Context, actor, wsID, id, channelID uuid.UUID, body string, parent *uuid.UUID) error {
	// пустое тело допускается, если сообщение несёт вложения (клиент шлёт body="" + attachments)
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		m := &models.Message{ID: id, WorkspaceID: wsID, ChannelID: channelID, AuthorID: actor, ParentID: parent, Body: body, CreatedAt: time.Now().UTC()}
		if _, err := tx.NewInsert().Model(m).Exec(ctx); err != nil {
			return err
		}
		payload, _ := json.Marshal(map[string]any{"channelId": channelID.String()})
		for _, u := range mentions.Parse(body) {
			if u == actor {
				continue
			}
			if err := notifications.Record(ctx, tx, wsID, u, actor, "mention", "channel", channelID, payload); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// Actor=author → клиент не считает своё сообщение непрочитанным в других каналах
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "message", EntityID: channelID.String(), Op: "create", Actor: actor.String()})
	return nil
}

func (s *Service) EditMessage(ctx context.Context, actor, wsID, id uuid.UUID, body string) error {
	if strings.TrimSpace(body) == "" {
		return ErrEmpty
	}
	var channelID uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var m models.Message
		if err := tx.NewSelect().Model(&m).Where("id = ?", id).Scan(ctx); err != nil {
			return err
		}
		if m.AuthorID != actor {
			return ErrForbidden
		}
		channelID = m.ChannelID
		_, err := tx.NewUpdate().Model((*models.Message)(nil)).
			Set("body = ?", body).Set("edited_at = now()").Where("id = ?", id).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	s.notify(ctx, wsID, channelID.String(), "update")
	return nil
}

func (s *Service) DeleteMessage(ctx context.Context, actor, wsID, id uuid.UUID) error {
	var channelID uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var m models.Message
		if err := tx.NewSelect().Model(&m).Where("id = ?", id).Scan(ctx); err != nil {
			return err
		}
		if m.AuthorID != actor {
			return ErrForbidden
		}
		channelID = m.ChannelID
		_, err := tx.NewUpdate().Model((*models.Message)(nil)).
			Set("deleted_at = now()").Set("body = ''").Where("id = ?", id).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	s.notify(ctx, wsID, channelID.String(), "delete")
	return nil
}

func (s *Service) ToggleReaction(ctx context.Context, actor, wsID, messageID uuid.UUID, emoji string) error {
	emoji = strings.TrimSpace(emoji)
	if emoji == "" {
		return ErrEmpty
	}
	var channelID uuid.UUID
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var m models.Message
		if err := tx.NewSelect().Model(&m).Where("id = ?", messageID).Scan(ctx); err != nil {
			return err
		}
		channelID = m.ChannelID
		res, err := tx.NewDelete().Model((*models.MessageReaction)(nil)).
			Where("message_id = ?", messageID).Where("user_id = ?", actor).Where("emoji = ?", emoji).Exec(ctx)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil // была — сняли (toggle off)
		}
		rid, _ := uuid.NewV7()
		_, err = tx.NewInsert().Model(&models.MessageReaction{ID: rid, WorkspaceID: wsID, MessageID: messageID, UserID: actor, Emoji: emoji, CreatedAt: time.Now().UTC()}).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	s.notify(ctx, wsID, channelID.String(), "update")
	return nil
}

// SendTyping публикует эфемерное событие набора (без БД).
func (s *Service) SendTyping(ctx context.Context, actor, wsID, channelID uuid.UUID) error {
	return s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "typing", EntityID: "typing:" + channelID.String(), Op: channelID.String(), Actor: actor.String()})
}

// MarkChannelRead — отметить канал прочитанным до текущего момента (upsert read-маркера).
func (s *Service) MarkChannelRead(ctx context.Context, actor, wsID, channelID uuid.UUID) error {
	return db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(&models.ChannelRead{WorkspaceID: wsID, ChannelID: channelID, UserID: actor, LastReadAt: time.Now().UTC()}).
			On("CONFLICT (channel_id, user_id) DO UPDATE").Set("last_read_at = EXCLUDED.last_read_at").Exec(ctx)
		return err
	})
}

type reactionDTO struct {
	Emoji  string    `json:"emoji"`
	UserID uuid.UUID `json:"userId"`
}
type messageDTO struct {
	models.Message
	ReplyCount int           `json:"replyCount"`
	Reactions  []reactionDTO `json:"reactions"`
}

func (s *Service) GetMessages(ctx context.Context, actor, wsID, channelID uuid.UUID, parent *uuid.UUID, before string, limit int) ([]byte, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var msgs []models.Message
	var out []byte
	hasMore := false
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewSelect().Model(&msgs).Where("channel_id = ?", channelID)
		if parent != nil {
			q.Where("parent_id = ?", *parent)
		} else {
			q.Where("parent_id IS NULL")
		}
		if before != "" {
			q.Where("id < ?", before)
		}
		if err := q.Order("created_at DESC").Limit(limit + 1).Scan(ctx); err != nil {
			return err
		}
		if len(msgs) > limit {
			hasMore = true
			msgs = msgs[:limit]
		}
		// reverse → по возрастанию времени
		for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
			msgs[i], msgs[j] = msgs[j], msgs[i]
		}
		if len(msgs) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, len(msgs))
		for i := range msgs {
			ids[i] = msgs[i].ID
		}
		var reactions []models.MessageReaction
		if err := tx.NewSelect().Model(&reactions).Where("message_id IN (?)", bun.In(ids)).Scan(ctx); err != nil {
			return err
		}
		reByMsg := map[uuid.UUID][]reactionDTO{}
		for _, r := range reactions {
			reByMsg[r.MessageID] = append(reByMsg[r.MessageID], reactionDTO{Emoji: r.Emoji, UserID: r.UserID})
		}
		// reply counts
		var rc []struct {
			ParentID uuid.UUID `bun:"parent_id"`
			N        int       `bun:"n"`
		}
		if err := tx.NewSelect().Model((*models.Message)(nil)).
			ColumnExpr("parent_id").ColumnExpr("COUNT(*) AS n").
			Where("parent_id IN (?)", bun.In(ids)).Where("deleted_at IS NULL").
			GroupExpr("parent_id").Scan(ctx, &rc); err != nil {
			return err
		}
		rcByMsg := map[uuid.UUID]int{}
		for _, r := range rc {
			rcByMsg[r.ParentID] = r.N
		}
		dtos := make([]messageDTO, len(msgs))
		for i := range msgs {
			rs := reByMsg[msgs[i].ID]
			if rs == nil {
				rs = []reactionDTO{}
			}
			dtos[i] = messageDTO{Message: msgs[i], ReplyCount: rcByMsg[msgs[i].ID], Reactions: rs}
		}
		raw, _ := json.Marshal(dtos)
		out = raw
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if out == nil {
		out = []byte("[]")
	}
	return out, hasMore, nil
}

package meetings

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var ErrNotFound = errors.New("meeting not found")

// LiveKit — реквизиты SFU для подписи клиентских токенов.
type LiveKit struct {
	URL    string
	APIKey string
	Secret string
}

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
	lk  LiveKit
}

func NewService(bdb *bun.DB, pub *realtime.Publisher, lk LiveKit) *Service {
	return &Service{db: bdb, pub: pub, lk: lk}
}

// mintToken — подписанный JWT для входа в комнату LiveKit (room = meetingID, identity = userID).
func (s *Service) mintToken(room, identity, name string) (string, error) {
	at := auth.NewAccessToken(s.lk.APIKey, s.lk.Secret).
		SetIdentity(identity).SetName(name).SetValidFor(12 * time.Hour).
		AddGrant(&auth.VideoGrant{RoomJoin: true, Room: room})
	return at.ToJWT()
}

func (s *Service) delta(ctx context.Context, wsID, id uuid.UUID, op string) {
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "meeting", EntityID: id.String(), Op: op})
}

// CreateMeeting — запланированный митинг с явным кругом участников.
func (s *Service) CreateMeeting(ctx context.Context, actor, wsID, id uuid.UUID, title string, scheduledAt *time.Time, participantIDs []uuid.UUID, channelID *uuid.UUID) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		m := &models.Meeting{
			ID: id, WorkspaceID: wsID, ChannelID: channelID, Kind: "scheduled",
			Title: strings.TrimSpace(title), Status: "scheduled", ScheduledAt: scheduledAt,
			CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(m).Exec(ctx); err != nil {
			return err
		}
		seen := map[uuid.UUID]bool{}
		add := func(uid uuid.UUID, role string) error {
			if uid == uuid.Nil || seen[uid] {
				return nil
			}
			seen[uid] = true
			pid, _ := uuid.NewV7()
			_, err := tx.NewInsert().Model(&models.MeetingParticipant{
				ID: pid, WorkspaceID: wsID, MeetingID: id, UserID: uid, Role: role, State: "invited", CreatedAt: now,
			}).Exec(ctx)
			return err
		}
		if err := add(actor, "host"); err != nil {
			return err
		}
		for _, uid := range participantIDs {
			if err := add(uid, "participant"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, id, "create")
	return nil
}

// StartHuddle — ad-hoc huddle на весь канал (или 1-1 DM). Участники = члены канала, актор сразу joined.
func (s *Service) StartHuddle(ctx context.Context, actor, wsID, id, channelID uuid.UUID) (uuid.UUID, error) {
	result := id
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		// уже есть живой huddle в этом канале — присоединяемся к нему
		var existing uuid.UUID
		err := tx.NewSelect().Model((*models.Meeting)(nil)).Column("id").
			Where("channel_id = ?", channelID).Where("kind = 'huddle'").Where("status = 'live'").
			Where("deleted_at IS NULL").Limit(1).Scan(ctx, &existing)
		if err == nil && existing != uuid.Nil {
			result = existing
			return nil
		}
		m := &models.Meeting{
			ID: id, WorkspaceID: wsID, ChannelID: &channelID, Kind: "huddle",
			Title: "Huddle", Status: "live", CreatedBy: actor, CreatedAt: now, UpdatedAt: now, StartedAt: &now,
		}
		if _, err := tx.NewInsert().Model(m).Exec(ctx); err != nil {
			return err
		}
		var memberIDs []uuid.UUID
		if err := tx.NewSelect().Model((*models.ChannelMember)(nil)).Column("user_id").
			Where("channel_id = ?", channelID).Scan(ctx, &memberIDs); err != nil {
			return err
		}
		for _, uid := range memberIDs {
			pid, _ := uuid.NewV7()
			role, state := "participant", "invited"
			if uid == actor {
				role, state = "host", "joined"
			}
			p := &models.MeetingParticipant{ID: pid, WorkspaceID: wsID, MeetingID: id, UserID: uid, Role: role, State: state, CreatedAt: now}
			if uid == actor {
				p.JoinedAt = &now
			}
			if _, err := tx.NewInsert().Model(p).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	s.delta(ctx, wsID, result, "create")
	return result, nil
}

// JoinMeeting — помечает участника joined и выдаёт LiveKit-токен (комната = meetingID).
// Медиа полностью на SFU (LiveKit); сервер только авторизует вход.
func (s *Service) JoinMeeting(ctx context.Context, actor, wsID, meetingID uuid.UUID) (url, token string, err error) {
	now := time.Now().UTC()
	var name string
	err = db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_ = tx.NewSelect().Model((*models.User)(nil)).Column("name").Where("id = ?", actor).Scan(ctx, &name)
		// upsert: участник мог быть только invited или ранее left
		pid, _ := uuid.NewV7()
		_, e := tx.NewInsert().Model(&models.MeetingParticipant{
			ID: pid, WorkspaceID: wsID, MeetingID: meetingID, UserID: actor, Role: "participant", State: "joined", JoinedAt: &now, CreatedAt: now,
		}).On("CONFLICT (meeting_id, user_id) DO UPDATE").
			Set("state = 'joined'").Set("joined_at = ?", now).Set("left_at = NULL").Exec(ctx)
		if e != nil {
			return e
		}
		// scheduled → live при первом входе
		_, e = tx.NewUpdate().Model((*models.Meeting)(nil)).
			Set("status = 'live'").Set("started_at = COALESCE(started_at, ?)", now).Set("updated_at = ?", now).
			Where("id = ?", meetingID).Where("status = 'scheduled'").Exec(ctx)
		return e
	})
	if err != nil {
		return "", "", err
	}
	if name == "" {
		name = "User"
	}
	token, err = s.mintToken(meetingID.String(), actor.String(), name)
	if err != nil {
		return "", "", err
	}
	s.delta(ctx, wsID, meetingID, "update")
	return s.lk.URL, token, nil
}

// LeaveMeeting — участник вышел; если живых не осталось — митинг ended.
func (s *Service) LeaveMeeting(ctx context.Context, actor, wsID, meetingID uuid.UUID) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model((*models.MeetingParticipant)(nil)).
			Set("state = 'left'").Set("left_at = ?", now).
			Where("meeting_id = ?", meetingID).Where("user_id = ?", actor).Exec(ctx); err != nil {
			return err
		}
		var live int
		if err := tx.NewSelect().Model((*models.MeetingParticipant)(nil)).ColumnExpr("count(*)").
			Where("meeting_id = ?", meetingID).Where("state = 'joined'").Scan(ctx, &live); err != nil {
			return err
		}
		if live == 0 {
			if _, err := tx.NewUpdate().Model((*models.Meeting)(nil)).
				Set("status = 'ended'").Set("ended_at = ?", now).Set("updated_at = ?", now).
				Where("id = ?", meetingID).Where("status != 'ended'").Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, meetingID, "update")
	return nil
}

// EndMeeting — принудительно завершить (хост).
func (s *Service) EndMeeting(ctx context.Context, actor, wsID, meetingID uuid.UUID) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.Meeting)(nil)).
			Set("status = 'ended'").Set("ended_at = ?", now).Set("updated_at = ?", now).
			Where("id = ?", meetingID).Exec(ctx)
		return err
	})
	if err != nil {
		return err
	}
	s.delta(ctx, wsID, meetingID, "update")
	return nil
}

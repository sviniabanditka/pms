package notifications

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

// Record создаёт уведомление в текущей транзакции. Вызывающий сам проверяет recipient != actor.
func Record(ctx context.Context, tx bun.IDB, wsID, recipient, actor uuid.UUID, typ, entityType string, entityID uuid.UUID, payload []byte) error {
	id, _ := uuid.NewV7()
	var ap *uuid.UUID
	if actor != uuid.Nil {
		ap = &actor
	}
	n := &models.Notification{
		ID: id, WorkspaceID: wsID, UserID: recipient, Type: typ,
		EntityType: entityType, EntityID: entityID, ActorID: ap, Payload: payload, At: time.Now().UTC(),
	}
	_, err := tx.NewInsert().Model(n).Exec(ctx)
	return err
}

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) MarkRead(ctx context.Context, actor, wsID, id uuid.UUID) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.Notification)(nil)).
			Set("read_at = now()").Set("version = version + 1").
			Where("id = ?", id).Where("user_id = ?", actor).Where("read_at IS NULL").Exec(ctx)
		return err
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "notification", EntityID: id.String(), Op: "update"})
	return 0, nil
}

func (s *Service) MarkAllRead(ctx context.Context, actor, wsID uuid.UUID) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewUpdate().Model((*models.Notification)(nil)).
			Set("read_at = now()").Set("version = version + 1").
			Where("user_id = ?", actor).Where("read_at IS NULL").Exec(ctx)
		return err
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "notification", EntityID: actor.String(), Op: "update"})
	return 0, nil
}

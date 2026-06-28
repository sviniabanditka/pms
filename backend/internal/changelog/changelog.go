package changelog

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/models"
	"github.com/uptrace/bun"
)

// Record пишет запись в change_log в рамках текущей транзакции (RLS-контекст уже выставлен).
// changed — произвольная структура изменённых полей (для create/update); nil допустим.
func Record(ctx context.Context, tx bun.IDB, wsID uuid.UUID, entityType string, entityID uuid.UUID, version int64, op string, changed any, actor uuid.UUID) error {
	var cf []byte
	if changed != nil {
		cf, _ = json.Marshal(changed)
	}
	var actorPtr *uuid.UUID
	if actor != uuid.Nil {
		actorPtr = &actor
	}
	entry := &models.ChangeLog{
		WorkspaceID:   wsID,
		EntityType:    entityType,
		EntityID:      entityID,
		Version:       version,
		Op:            op,
		ChangedFields: cf,
		ActorID:       actorPtr,
		At:            time.Now().UTC(),
	}
	_, err := tx.NewInsert().Model(entry).Exec(ctx)
	return err
}

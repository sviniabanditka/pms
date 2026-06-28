package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ChangeLog — append-only журнал дельт (источник догона синка). seq генерится БД.
type ChangeLog struct {
	bun.BaseModel `bun:"table:change_log,alias:cl"`

	Seq           int64      `bun:"seq,pk,autoincrement,nullzero"`
	WorkspaceID   uuid.UUID  `bun:"workspace_id"`
	EntityType    string     `bun:"entity_type"`
	EntityID      uuid.UUID  `bun:"entity_id"`
	Version       int64      `bun:"version"`
	Op            string     `bun:"op"`
	ChangedFields []byte     `bun:"changed_fields,type:jsonb,nullzero"`
	ActorID       *uuid.UUID `bun:"actor_id"`
	At            time.Time  `bun:"at,nullzero"`
}

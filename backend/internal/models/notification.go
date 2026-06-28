package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Notification struct {
	bun.BaseModel `bun:"table:notifications,alias:n" json:"-"`

	ID          uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	UserID      uuid.UUID       `bun:"user_id" json:"userId"`
	Type        string          `bun:"type" json:"type"`
	EntityType  string          `bun:"entity_type" json:"entityType"`
	EntityID    uuid.UUID       `bun:"entity_id" json:"entityId"`
	ActorID     *uuid.UUID      `bun:"actor_id" json:"actorId,omitempty"`
	Payload     json.RawMessage `bun:"payload,type:jsonb" json:"payload,omitempty"`
	ReadAt      *time.Time      `bun:"read_at" json:"readAt,omitempty"`
	EmailedAt   *time.Time      `bun:"emailed_at" json:"-"`
	At          time.Time       `bun:"at,nullzero" json:"at"`
	Version     int64           `bun:"version" json:"version"`
}

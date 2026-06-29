package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Automation struct {
	bun.BaseModel `bun:"table:automations,alias:au" json:"-"`

	ID          uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	TeamID      *uuid.UUID      `bun:"team_id" json:"teamId,omitempty"`
	Name        string          `bun:"name" json:"name"`
	Enabled     bool            `bun:"enabled" json:"enabled"`
	Trigger     json.RawMessage `bun:"trigger,type:jsonb" json:"trigger"`
	Conditions  json.RawMessage `bun:"conditions,type:jsonb" json:"conditions,omitempty"`
	Actions     json.RawMessage `bun:"actions,type:jsonb" json:"actions"`
	CreatedBy   uuid.UUID       `bun:"created_by" json:"createdBy"`
	CreatedAt   time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64           `bun:"version" json:"version"`
	DeletedAt   *time.Time      `bun:"deleted_at" json:"deletedAt,omitempty"`
}

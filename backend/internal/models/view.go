package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type View struct {
	bun.BaseModel `bun:"table:views,alias:v" json:"-"`

	ID          uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	Name        string          `bun:"name" json:"name"`
	DataSource  string          `bun:"data_source" json:"dataSource"`
	ScopeKind   string          `bun:"scope_kind" json:"scopeKind"`
	TeamID      *uuid.UUID      `bun:"team_id" json:"teamId,omitempty"`
	OwnerID     *uuid.UUID      `bun:"owner_id" json:"ownerId,omitempty"`
	Shared      bool            `bun:"shared" json:"shared"`
	Filter      json.RawMessage `bun:"filter,type:jsonb" json:"filter"`
	Grouping    json.RawMessage `bun:"grouping,type:jsonb" json:"grouping,omitempty"`
	Ordering    json.RawMessage `bun:"ordering,type:jsonb" json:"ordering,omitempty"`
	Layout      string          `bun:"layout" json:"layout"`
	IsDefault   bool            `bun:"is_default" json:"isDefault"`
	Position    *string         `bun:"position" json:"position,omitempty"`
	CreatedAt   time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64           `bun:"version" json:"version"`
	ArchivedAt  *time.Time      `bun:"archived_at" json:"archivedAt,omitempty"`
	DeletedAt   *time.Time      `bun:"deleted_at" json:"deletedAt,omitempty"`
}

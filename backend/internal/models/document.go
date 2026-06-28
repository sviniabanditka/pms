package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Document struct {
	bun.BaseModel `bun:"table:documents,alias:doc" json:"-"`

	ID          uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	ParentID    *uuid.UUID      `bun:"parent_id" json:"parentId,omitempty"`
	ScopeKind   string          `bun:"scope_kind" json:"scopeKind"`
	TeamID      *uuid.UUID      `bun:"team_id" json:"teamId,omitempty"`
	ProjectID   *uuid.UUID      `bun:"project_id" json:"projectId,omitempty"`
	Title       string          `bun:"title" json:"title"`
	Body        json.RawMessage `bun:"body,type:jsonb" json:"body"`
	Icon        *string         `bun:"icon" json:"icon,omitempty"`
	Position    *string         `bun:"position" json:"position,omitempty"`
	CreatedBy   uuid.UUID       `bun:"created_by" json:"createdBy"`
	CreatedAt   time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64           `bun:"version" json:"version"`
	ArchivedAt  *time.Time      `bun:"archived_at" json:"archivedAt,omitempty"`
	DeletedAt   *time.Time      `bun:"deleted_at" json:"deletedAt,omitempty"`
}

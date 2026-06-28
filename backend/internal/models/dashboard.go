package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Dashboard struct {
	bun.BaseModel `bun:"table:dashboards,alias:d" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	Name        string     `bun:"name" json:"name"`
	ScopeKind   string     `bun:"scope_kind" json:"scopeKind"`
	TeamID      *uuid.UUID `bun:"team_id" json:"teamId,omitempty"`
	OwnerID     *uuid.UUID `bun:"owner_id" json:"ownerId,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time  `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64      `bun:"version" json:"version"`
	ArchivedAt  *time.Time `bun:"archived_at" json:"archivedAt,omitempty"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

type DashboardWidget struct {
	bun.BaseModel `bun:"table:dashboard_widgets,alias:dw" json:"-"`

	ID          uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	DashboardID uuid.UUID       `bun:"dashboard_id" json:"dashboardId"`
	Type        string          `bun:"type" json:"type"`
	Title       *string         `bun:"title" json:"title,omitempty"`
	X           int             `bun:"x" json:"x"`
	Y           int             `bun:"y" json:"y"`
	W           int             `bun:"w" json:"w"`
	H           int             `bun:"h" json:"h"`
	Config      json.RawMessage `bun:"config,type:jsonb" json:"config"`
	CreatedAt   time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64           `bun:"version" json:"version"`
}

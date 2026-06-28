package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Workspace struct {
	bun.BaseModel `bun:"table:workspaces,alias:w" json:"-"`

	ID         uuid.UUID       `bun:"id,pk" json:"id"`
	Slug       string          `bun:"slug" json:"slug"`
	Name       string          `bun:"name" json:"name"`
	LogoURL    *string         `bun:"logo_url" json:"logoUrl"`
	LogoKey    *string         `bun:"logo_key" json:"logoKey"`
	Settings   json.RawMessage `bun:"settings,type:jsonb" json:"-"`
	CreatedBy  uuid.UUID       `bun:"created_by" json:"createdBy"`
	CreatedAt  time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt  time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version    int64           `bun:"version" json:"version"`
	ArchivedAt *time.Time      `bun:"archived_at" json:"archivedAt,omitempty"`
	DeletedAt  *time.Time      `bun:"deleted_at" json:"deletedAt,omitempty"`
}

type WorkspaceMember struct {
	bun.BaseModel `bun:"table:workspace_members,alias:wm" json:"-"`

	ID             uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID    uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	UserID         uuid.UUID       `bun:"user_id" json:"userId"`
	Role           string          `bun:"role" json:"role"`
	Status         string          `bun:"status" json:"status"`
	SectionsConfig json.RawMessage `bun:"sections_config,type:jsonb" json:"sectionsConfig"`
	JoinedAt       time.Time       `bun:"joined_at,nullzero" json:"joinedAt"`
	CreatedAt      time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt      time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version        int64           `bun:"version" json:"version"`
}

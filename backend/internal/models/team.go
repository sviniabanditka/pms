package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Team struct {
	bun.BaseModel `bun:"table:teams,alias:t" json:"-"`

	ID              uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID     uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	ParentID        *uuid.UUID      `bun:"parent_id" json:"parentId,omitempty"`
	Path            string          `bun:"path,type:ltree" json:"path"`
	Key             string          `bun:"key" json:"key"`
	Name            string          `bun:"name" json:"name"`
	Private         bool            `bun:"private" json:"private"`
	NextIssueNumber int64           `bun:"next_issue_number" json:"nextIssueNumber"`
	EnabledFeatures json.RawMessage `bun:"enabled_features,type:jsonb" json:"-"`
	CycleConfig     json.RawMessage `bun:"cycle_config,type:jsonb" json:"-"`
	Settings        json.RawMessage `bun:"settings,type:jsonb" json:"-"`
	CreatedAt       time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt       time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version         int64           `bun:"version" json:"version"`
	ArchivedAt      *time.Time      `bun:"archived_at" json:"archivedAt,omitempty"`
	DeletedAt       *time.Time      `bun:"deleted_at" json:"deletedAt,omitempty"`
}

type TeamMember struct {
	bun.BaseModel `bun:"table:team_members,alias:tm" json:"-"`

	ID          uuid.UUID `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	TeamID      uuid.UUID `bun:"team_id" json:"teamId"`
	UserID      uuid.UUID `bun:"user_id" json:"userId"`
	Role        string    `bun:"role" json:"role"`
	Source      string    `bun:"source" json:"source"`
	CreatedAt   time.Time `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64     `bun:"version" json:"version"`
}

type WorkflowState struct {
	bun.BaseModel `bun:"table:workflow_states,alias:ws" json:"-"`

	ID          uuid.UUID `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	TeamID      uuid.UUID `bun:"team_id" json:"teamId"`
	Category    string    `bun:"category" json:"category"`
	Name        string    `bun:"name" json:"name"`
	ColorToken  string    `bun:"color_token" json:"colorToken"`
	Position    string    `bun:"position" json:"position"`
	CreatedAt   time.Time `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64     `bun:"version" json:"version"`
}

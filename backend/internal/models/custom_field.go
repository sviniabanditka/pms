package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type CustomField struct {
	bun.BaseModel `bun:"table:custom_fields,alias:cf" json:"-"`

	ID          uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	TeamID      *uuid.UUID      `bun:"team_id" json:"teamId,omitempty"`
	EntityType  string          `bun:"entity_type" json:"entityType"`
	Key         string          `bun:"key" json:"key"`
	Name        string          `bun:"name" json:"name"`
	Type        string          `bun:"type" json:"type"`
	Config      json.RawMessage `bun:"config,type:jsonb" json:"config"`
	Position    *string         `bun:"position" json:"position,omitempty"`
	CreatedAt   time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64           `bun:"version" json:"version"`
	ArchivedAt  *time.Time      `bun:"archived_at" json:"archivedAt,omitempty"`
}

type CustomFieldValue struct {
	bun.BaseModel `bun:"table:custom_field_values,alias:cfv" json:"-"`

	ID          uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	FieldID     uuid.UUID       `bun:"field_id" json:"fieldId"`
	IssueID     uuid.UUID       `bun:"issue_id" json:"issueId"`
	Value       json.RawMessage `bun:"value,type:jsonb" json:"value"`
	CreatedAt   time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64           `bun:"version" json:"version"`
}

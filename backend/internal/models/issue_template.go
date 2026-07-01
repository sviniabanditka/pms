package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// IssueTemplate — переиспользуемый blueprint issue. recurrence<>none → RecurringIssueWorker
// плодит issue по расписанию (next_run_at).
type IssueTemplate struct {
	bun.BaseModel `bun:"table:issue_templates,alias:it" json:"-"`

	ID            uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID   uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	TeamID        uuid.UUID       `bun:"team_id" json:"teamId"`
	Name          string          `bun:"name" json:"name"`
	Title         string          `bun:"title" json:"title"`
	Description   json.RawMessage `bun:"description,type:jsonb" json:"description,omitempty"`
	Priority      int16           `bun:"priority" json:"priority"`
	AssigneeID    *uuid.UUID      `bun:"assignee_id" json:"assigneeId,omitempty"`
	Estimate      *float64        `bun:"estimate" json:"estimate,omitempty"`
	LabelIDs      json.RawMessage `bun:"label_ids,type:jsonb" json:"labelIds,omitempty"`
	Recurrence    string          `bun:"recurrence" json:"recurrence"` // none|daily|weekly|monthly
	NextRunAt     *time.Time      `bun:"next_run_at" json:"nextRunAt,omitempty"`
	LastSpawnedAt *time.Time      `bun:"last_spawned_at" json:"lastSpawnedAt,omitempty"`
	CreatedBy     uuid.UUID       `bun:"created_by" json:"createdBy"`
	CreatedAt     time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt     time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	DeletedAt     *time.Time      `bun:"deleted_at" json:"deletedAt,omitempty"`
}

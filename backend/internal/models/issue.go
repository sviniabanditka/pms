package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Issue struct {
	bun.BaseModel `bun:"table:issues,alias:i" json:"-"`

	ID            uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID   uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	TeamID        uuid.UUID       `bun:"team_id" json:"teamId"`
	Number        int64           `bun:"number" json:"number"`
	Identifier    string          `bun:"identifier" json:"identifier"`
	ParentID      *uuid.UUID      `bun:"parent_id" json:"parentId,omitempty"`
	Title         string          `bun:"title" json:"title"`
	Description   json.RawMessage `bun:"description,type:jsonb" json:"description,omitempty"`
	StateID       uuid.UUID       `bun:"state_id" json:"stateId"`
	Priority      int16           `bun:"priority" json:"priority"`
	AssigneeID    *uuid.UUID      `bun:"assignee_id" json:"assigneeId,omitempty"`
	Estimate      *float64        `bun:"estimate" json:"estimate,omitempty"`
	ProjectID     *uuid.UUID      `bun:"project_id" json:"projectId,omitempty"`
	CycleID       *uuid.UUID      `bun:"cycle_id" json:"cycleId,omitempty"`
	MilestoneID   *uuid.UUID      `bun:"milestone_id" json:"milestoneId,omitempty"`
	DueDate       *time.Time      `bun:"due_date" json:"dueDate,omitempty"`
	StartDate     *time.Time      `bun:"start_date" json:"startDate,omitempty"`
	ScheduledDate *time.Time      `bun:"scheduled_date" json:"scheduledDate,omitempty"`
	StartedAt     *time.Time      `bun:"started_at" json:"startedAt,omitempty"`
	CompletedAt   *time.Time      `bun:"completed_at" json:"completedAt,omitempty"`
	BoardPosition float64         `bun:"board_position" json:"boardPosition"`
	SubPosition   float64         `bun:"sub_position" json:"subPosition"`
	CreatedBy     uuid.UUID       `bun:"created_by" json:"createdBy"`
	CreatedAt     time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt     time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version       int64           `bun:"version" json:"version"`
	ArchivedAt    *time.Time      `bun:"archived_at" json:"archivedAt,omitempty"`
	DeletedAt     *time.Time      `bun:"deleted_at" json:"deletedAt,omitempty"`
}

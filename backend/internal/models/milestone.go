package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Milestone struct {
	bun.BaseModel `bun:"table:milestones,alias:ms" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	TeamID      uuid.UUID  `bun:"team_id" json:"teamId"`
	Name        string     `bun:"name" json:"name"`
	TargetDate  *time.Time `bun:"target_date" json:"targetDate,omitempty"`
	Status      string     `bun:"status" json:"status"`
	Position    string     `bun:"position" json:"position"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time  `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64      `bun:"version" json:"version"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

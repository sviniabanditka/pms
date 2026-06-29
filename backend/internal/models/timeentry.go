package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TimeEntry struct {
	bun.BaseModel `bun:"table:time_entries,alias:te" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	UserID      uuid.UUID  `bun:"user_id" json:"userId"`
	IssueID     *uuid.UUID `bun:"issue_id" json:"issueId,omitempty"`
	Description string     `bun:"description" json:"description"`
	StartedAt   time.Time  `bun:"started_at" json:"startedAt"`
	EndedAt     *time.Time `bun:"ended_at" json:"endedAt,omitempty"` // nil = таймер идёт
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	Version     int64      `bun:"version" json:"version"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

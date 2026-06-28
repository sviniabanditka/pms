package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type IssueRelation struct {
	bun.BaseModel `bun:"table:issue_relations,alias:ir" json:"-"`

	ID          uuid.UUID `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	SourceID    uuid.UUID `bun:"source_id" json:"sourceId"`
	TargetID    uuid.UUID `bun:"target_id" json:"targetId"`
	Type        string    `bun:"type" json:"type"`
	CreatedAt   time.Time `bun:"created_at,nullzero" json:"createdAt"`
	Version     int64     `bun:"version" json:"version"`
}

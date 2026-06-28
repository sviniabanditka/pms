package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Label struct {
	bun.BaseModel `bun:"table:labels,alias:lb" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	TeamID      *uuid.UUID `bun:"team_id" json:"teamId,omitempty"`
	GroupID     *uuid.UUID `bun:"group_id" json:"groupId,omitempty"`
	IsGroup     bool       `bun:"is_group" json:"isGroup"`
	Name        string     `bun:"name" json:"name"`
	ColorToken  string     `bun:"color_token" json:"colorToken"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time  `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64      `bun:"version" json:"version"`
}

type IssueLabel struct {
	bun.BaseModel `bun:"table:issue_labels,alias:il" json:"-"`

	WorkspaceID uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	IssueID     uuid.UUID `bun:"issue_id" json:"issueId"`
	LabelID     uuid.UUID `bun:"label_id" json:"labelId"`
}

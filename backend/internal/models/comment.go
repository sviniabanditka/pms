package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Comment struct {
	bun.BaseModel `bun:"table:comments,alias:cm" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	IssueID     uuid.UUID  `bun:"issue_id" json:"issueId"`
	AuthorID    uuid.UUID  `bun:"author_id" json:"authorId"`
	Body        string     `bun:"body" json:"body"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	EditedAt    *time.Time `bun:"edited_at" json:"editedAt,omitempty"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

type CommentReaction struct {
	bun.BaseModel `bun:"table:comment_reactions,alias:cmr" json:"-"`

	ID          uuid.UUID `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	CommentID   uuid.UUID `bun:"comment_id" json:"commentId"`
	UserID      uuid.UUID `bun:"user_id" json:"userId"`
	Emoji       string    `bun:"emoji" json:"emoji"`
	CreatedAt   time.Time `bun:"created_at,nullzero" json:"createdAt"`
}

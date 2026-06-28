package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Channel struct {
	bun.BaseModel `bun:"table:channels,alias:ch" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	Kind        string     `bun:"kind" json:"kind"`
	Name        string     `bun:"name" json:"name"`
	Private     bool       `bun:"private" json:"private"`
	CreatedBy   uuid.UUID  `bun:"created_by" json:"createdBy"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time  `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64      `bun:"version" json:"version"`
	ArchivedAt  *time.Time `bun:"archived_at" json:"archivedAt,omitempty"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

type ChannelMember struct {
	bun.BaseModel `bun:"table:channel_members,alias:chm" json:"-"`

	ID          uuid.UUID `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	ChannelID   uuid.UUID `bun:"channel_id" json:"channelId"`
	UserID      uuid.UUID `bun:"user_id" json:"userId"`
	CreatedAt   time.Time `bun:"created_at,nullzero" json:"createdAt"`
}

type ChannelRead struct {
	bun.BaseModel `bun:"table:channel_reads,alias:cr" json:"-"`

	WorkspaceID uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	ChannelID   uuid.UUID `bun:"channel_id" json:"channelId"`
	UserID      uuid.UUID `bun:"user_id" json:"userId"`
	LastReadAt  time.Time `bun:"last_read_at" json:"lastReadAt"`
}

type Message struct {
	bun.BaseModel `bun:"table:messages,alias:msg" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	ChannelID   uuid.UUID  `bun:"channel_id" json:"channelId"`
	AuthorID    uuid.UUID  `bun:"author_id" json:"authorId"`
	ParentID    *uuid.UUID `bun:"parent_id" json:"parentId,omitempty"`
	Body        string     `bun:"body" json:"body"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	EditedAt    *time.Time `bun:"edited_at" json:"editedAt,omitempty"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

type MessageReaction struct {
	bun.BaseModel `bun:"table:message_reactions,alias:mr" json:"-"`

	ID          uuid.UUID `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	MessageID   uuid.UUID `bun:"message_id" json:"messageId"`
	UserID      uuid.UUID `bun:"user_id" json:"userId"`
	Emoji       string    `bun:"emoji" json:"emoji"`
	CreatedAt   time.Time `bun:"created_at,nullzero" json:"createdAt"`
}

// PublicUser — облегчённая карточка юзера для клиента (имена в чатах/ассайнах).
type PublicUser struct {
	bun.BaseModel `bun:"table:users,alias:u" json:"-"`

	ID        uuid.UUID `bun:"id,pk" json:"id"`
	Email     string    `bun:"email" json:"email"`
	Name      string    `bun:"name" json:"name"`
	AvatarKey *string   `bun:"avatar_key" json:"avatarKey"`
}

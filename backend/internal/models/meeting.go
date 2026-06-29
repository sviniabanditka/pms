package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Meeting struct {
	bun.BaseModel `bun:"table:meetings,alias:mt" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	ChannelID   *uuid.UUID `bun:"channel_id" json:"channelId,omitempty"`
	Kind        string     `bun:"kind" json:"kind"`
	Title       string     `bun:"title" json:"title"`
	Status      string     `bun:"status" json:"status"`
	ScheduledAt *time.Time `bun:"scheduled_at" json:"scheduledAt,omitempty"`
	EndsAt      *time.Time `bun:"ends_at" json:"endsAt,omitempty"`
	CreatedBy   uuid.UUID  `bun:"created_by" json:"createdBy"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time  `bun:"updated_at,nullzero" json:"updatedAt"`
	StartedAt   *time.Time `bun:"started_at" json:"startedAt,omitempty"`
	EndedAt     *time.Time `bun:"ended_at" json:"endedAt,omitempty"`
	Version     int64      `bun:"version" json:"version"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

type MeetingParticipant struct {
	bun.BaseModel `bun:"table:meeting_participants,alias:mtp" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	MeetingID   uuid.UUID  `bun:"meeting_id" json:"meetingId"`
	UserID      uuid.UUID  `bun:"user_id" json:"userId"`
	Role        string     `bun:"role" json:"role"`
	State       string     `bun:"state" json:"state"`
	JoinedAt    *time.Time `bun:"joined_at" json:"joinedAt,omitempty"`
	LeftAt      *time.Time `bun:"left_at" json:"leftAt,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
}

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type CalendarEvent struct {
	bun.BaseModel `bun:"table:calendar_events,alias:ce" json:"-"`

	ID              uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID     uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	UserID          uuid.UUID  `bun:"user_id" json:"userId"`
	Kind            string     `bun:"kind" json:"kind"`
	Title           string     `bun:"title" json:"title"`
	Description     string     `bun:"description" json:"description"`
	StartsAt        time.Time  `bun:"starts_at" json:"startsAt"`
	EndsAt          time.Time  `bun:"ends_at" json:"endsAt"`
	AllDay          bool       `bun:"all_day" json:"allDay"`
	Recurrence      string     `bun:"recurrence" json:"recurrence"` // none|daily|weekly|monthly
	RecurrenceUntil *time.Time `bun:"recurrence_until" json:"recurrenceUntil,omitempty"`
	CreatedBy       uuid.UUID  `bun:"created_by" json:"createdBy"`
	CreatedAt       time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt       time.Time  `bun:"updated_at,nullzero" json:"updatedAt"`
	Version         int64      `bun:"version" json:"version"`
	DeletedAt       *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

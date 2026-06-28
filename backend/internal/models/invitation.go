package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Invitation struct {
	bun.BaseModel `bun:"table:invitations,alias:inv" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	Email       string     `bun:"email" json:"email"`
	Role        string     `bun:"role" json:"role"`
	TokenHash   string     `bun:"token_hash" json:"-"`
	InvitedBy   uuid.UUID  `bun:"invited_by" json:"invitedBy"`
	ExpiresAt   time.Time  `bun:"expires_at" json:"expiresAt"`
	AcceptedAt  *time.Time `bun:"accepted_at" json:"acceptedAt,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time  `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64      `bun:"version" json:"version"`
}

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// APIToken — Personal Access Token публичного API. token_hash = sha256(raw); raw не хранится.
type APIToken struct {
	bun.BaseModel `bun:"table:api_tokens,alias:at" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	UserID      uuid.UUID  `bun:"user_id" json:"userId"`
	Name        string     `bun:"name" json:"name"`
	TokenHash   string     `bun:"token_hash" json:"-"`
	Prefix      string     `bun:"prefix" json:"prefix"`
	ReadOnly    bool       `bun:"read_only" json:"readOnly"`
	LastUsedAt  *time.Time `bun:"last_used_at" json:"lastUsedAt,omitempty"`
	ExpiresAt   *time.Time `bun:"expires_at" json:"expiresAt,omitempty"`
	RevokedAt   *time.Time `bun:"revoked_at" json:"revokedAt,omitempty"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
}

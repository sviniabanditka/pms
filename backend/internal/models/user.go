package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User — глобальная учётка (без workspace_id, без RLS).
type User struct {
	bun.BaseModel `bun:"table:users,alias:u"`

	ID           uuid.UUID       `bun:"id,pk"`
	Email        string          `bun:"email"`
	Name         string          `bun:"name"`
	PasswordHash *string         `bun:"password_hash"`
	Preferences  json.RawMessage `bun:"preferences,type:jsonb"`
	IcalToken    *string         `bun:"ical_token"`
	CreatedAt    time.Time       `bun:"created_at,nullzero"`
	UpdatedAt    time.Time       `bun:"updated_at,nullzero"`
	Version      int64           `bun:"version"`
	DeletedAt    *time.Time      `bun:"deleted_at"`
}

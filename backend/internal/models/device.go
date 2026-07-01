package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// DeviceToken — Expo push-токен устройства (нативный клиент). БЕЗ RLS, user-scoped.
type DeviceToken struct {
	bun.BaseModel `bun:"table:device_tokens,alias:dt" json:"-"`

	ID        uuid.UUID `bun:"id,pk" json:"id"`
	UserID    uuid.UUID `bun:"user_id" json:"userId"`
	Token     string    `bun:"token" json:"token"`
	Platform  string    `bun:"platform" json:"platform"`
	CreatedAt time.Time `bun:"created_at,nullzero" json:"createdAt"`
	LastSeen  time.Time `bun:"last_seen,nullzero" json:"lastSeen"`
}

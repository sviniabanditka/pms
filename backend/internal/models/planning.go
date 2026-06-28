package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Project struct {
	bun.BaseModel `bun:"table:projects,alias:p" json:"-"`

	ID          uuid.UUID       `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID       `bun:"workspace_id" json:"workspaceId"`
	LeadID      *uuid.UUID      `bun:"lead_id" json:"leadId,omitempty"`
	Name        string          `bun:"name" json:"name"`
	Description json.RawMessage `bun:"description,type:jsonb" json:"-"`
	Status      string          `bun:"status" json:"status"`
	Health      *string         `bun:"health" json:"health,omitempty"`
	StartDate   *time.Time      `bun:"start_date" json:"startDate,omitempty"`
	TargetDate  *time.Time      `bun:"target_date" json:"targetDate,omitempty"`
	ColorToken  *string         `bun:"color_token" json:"colorToken,omitempty"`
	CreatedAt   time.Time       `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time       `bun:"updated_at,nullzero" json:"updatedAt"`
	Version     int64           `bun:"version" json:"version"`
	ArchivedAt  *time.Time      `bun:"archived_at" json:"archivedAt,omitempty"`
	DeletedAt   *time.Time      `bun:"deleted_at" json:"deletedAt,omitempty"`
}

type CycleSnapshot struct {
	bun.BaseModel `bun:"table:cycle_snapshots,alias:cs" json:"-"`

	WorkspaceID     uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	CycleID         uuid.UUID `bun:"cycle_id" json:"cycleId"`
	Day             string    `bun:"day" json:"day"`
	RemainingPoints float64   `bun:"remaining_points" json:"remainingPoints"`
	RemainingCount  int       `bun:"remaining_count" json:"remainingCount"`
	TotalPoints     float64   `bun:"total_points" json:"totalPoints"`
	TotalCount      int       `bun:"total_count" json:"totalCount"`
}

type Cycle struct {
	bun.BaseModel `bun:"table:cycles,alias:cy" json:"-"`

	ID           uuid.UUID `bun:"id,pk" json:"id"`
	WorkspaceID  uuid.UUID `bun:"workspace_id" json:"workspaceId"`
	TeamID       uuid.UUID `bun:"team_id" json:"teamId"`
	Number       int       `bun:"number" json:"number"`
	Name         *string   `bun:"name" json:"name,omitempty"`
	StartsOn     time.Time `bun:"starts_on" json:"startsOn"`
	EndsOn       time.Time `bun:"ends_on" json:"endsOn"`
	CooldownDays int       `bun:"cooldown_days" json:"cooldownDays"`
	CreatedAt    time.Time `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt    time.Time `bun:"updated_at,nullzero" json:"updatedAt"`
	Version      int64     `bun:"version" json:"version"`
}

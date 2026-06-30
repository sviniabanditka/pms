package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Webhook — подписка воркспейса на события (доставка по change_log).
type Webhook struct {
	bun.BaseModel `bun:"table:webhooks,alias:wh" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	URL         string     `bun:"url" json:"url"`
	Secret      string     `bun:"secret" json:"-"` // не отдаём в list (показываем один раз при создании)
	Events      []string   `bun:"events,array" json:"events"`
	Active      bool       `bun:"active" json:"active"`
	CreatedBy   uuid.UUID  `bun:"created_by" json:"createdBy"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	UpdatedAt   time.Time  `bun:"updated_at,nullzero" json:"updatedAt"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"-"`
}

// WebhookDelivery — запись попытки доставки (лог).
type WebhookDelivery struct {
	bun.BaseModel `bun:"table:webhook_deliveries,alias:wd" json:"-"`

	ID           uuid.UUID  `bun:"id,pk" json:"id"`
	WebhookID    uuid.UUID  `bun:"webhook_id" json:"webhookId"`
	WorkspaceID  uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	Event        string     `bun:"event" json:"event"`
	Status       string     `bun:"status" json:"status"`
	ResponseCode *int       `bun:"response_code" json:"responseCode,omitempty"`
	Attempts     int        `bun:"attempts" json:"attempts"`
	CreatedAt    time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	DeliveredAt  *time.Time `bun:"delivered_at" json:"deliveredAt,omitempty"`
}

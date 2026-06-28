package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Attachment struct {
	bun.BaseModel `bun:"table:attachments,alias:att" json:"-"`

	ID          uuid.UUID  `bun:"id,pk" json:"id"`
	WorkspaceID uuid.UUID  `bun:"workspace_id" json:"workspaceId"`
	EntityType  string     `bun:"entity_type" json:"entityType"`
	EntityID    uuid.UUID  `bun:"entity_id" json:"entityId"`
	Uploader    uuid.UUID  `bun:"uploader" json:"uploader"`
	Filename    string     `bun:"filename" json:"filename"`
	ContentType string     `bun:"content_type" json:"contentType"`
	Size        int64      `bun:"size" json:"size"`
	S3Key       string     `bun:"s3_key" json:"s3Key"`
	CreatedAt   time.Time  `bun:"created_at,nullzero" json:"createdAt"`
	DeletedAt   *time.Time `bun:"deleted_at" json:"deletedAt,omitempty"`
}

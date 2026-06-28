package customfields

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/changelog"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/uptrace/bun"
)

var (
	ErrName = errors.New("field name required")
	ErrType = errors.New("invalid field type")
)

var fieldTypes = map[string]bool{
	"text": true, "number": true, "date": true, "checkbox": true,
	"select": true, "multiselect": true, "url": true,
}
var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

type Service struct {
	db  *bun.DB
	pub *realtime.Publisher
}

func NewService(bdb *bun.DB, pub *realtime.Publisher) *Service { return &Service{db: bdb, pub: pub} }

func (s *Service) CreateCustomField(ctx context.Context, actor, wsID, id uuid.UUID, teamID *uuid.UUID, name, typ string, config []byte) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, ErrName
	}
	if !fieldTypes[typ] {
		return 0, ErrType
	}
	if len(config) == 0 {
		config = []byte("{}")
	}
	key := strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(name), "_"), "_")
	if key == "" {
		key = "field"
	}
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		f := &models.CustomField{
			ID: id, WorkspaceID: wsID, TeamID: teamID, EntityType: "issue",
			Key: key, Name: name, Type: typ, Config: config, CreatedAt: now, UpdatedAt: now,
		}
		if _, err := tx.NewInsert().Model(f).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "custom_field", id, 0, "create", map[string]any{"name": name}, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "custom_field", EntityID: id.String(), Op: "create"})
	return 0, nil
}

func (s *Service) ArchiveCustomField(ctx context.Context, actor, wsID, id uuid.UUID) (int64, error) {
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewUpdate().Model((*models.CustomField)(nil)).
			Set("archived_at = now()").Set("updated_at = now()").Set("version = version + 1").
			Where("id = ?", id).Exec(ctx); err != nil {
			return err
		}
		return changelog.Record(ctx, tx, wsID, "custom_field", id, 0, "archive", nil, actor)
	})
	if err != nil {
		return 0, err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "custom_field", EntityID: id.String(), Op: "archive"})
	return 0, nil
}

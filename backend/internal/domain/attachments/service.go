package attachments

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/pms/backend/internal/storage"
	"github.com/uptrace/bun"
)

var ErrForbidden = errors.New("not a workspace member")

type Service struct {
	db    *bun.DB
	pub   *realtime.Publisher
	store *storage.Store
}

func NewService(bdb *bun.DB, pub *realtime.Publisher, store *storage.Store) *Service {
	return &Service{db: bdb, pub: pub, store: store}
}

// Create регистрирует уже залитый в S3 объект как вложение сущности.
func (s *Service) Create(ctx context.Context, actor, wsID, id uuid.UUID, entityType string, entityID uuid.UUID, filename, contentType string, size int64, s3Key string) error {
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, e := tx.NewInsert().Model(&models.Attachment{
			ID: id, WorkspaceID: wsID, EntityType: entityType, EntityID: entityID,
			Uploader: actor, Filename: filename, ContentType: contentType, Size: size, S3Key: s3Key, CreatedAt: now,
		}).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "attachment", EntityID: entityID.String(), Op: "create"})
	return nil
}

func (s *Service) Delete(ctx context.Context, actor, wsID, id uuid.UUID) error {
	now := time.Now().UTC()
	var key string
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var att models.Attachment
		if e := tx.NewSelect().Model(&att).Where("id = ?", id).Scan(ctx); e != nil {
			return e
		}
		key = att.S3Key
		_, e := tx.NewUpdate().Model((*models.Attachment)(nil)).Set("deleted_at = ?", now).Where("id = ?", id).Exec(ctx)
		return e
	})
	if err != nil {
		return err
	}
	if key != "" {
		_ = s.store.Delete(ctx, key)
	}
	_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "attachment", EntityID: id.String(), Op: "delete"})
	return nil
}

// URL — временная presigned-ссылка на объект (для <img>/<a>/скачивания).
// Ключи имеют вид ws/<wsID>/...; выдаём ссылку только члену этого воркспейса (IDOR-защита).
func (s *Service) URL(ctx context.Context, actor uuid.UUID, key string) (string, error) {
	wsID, ok := workspaceFromKey(key)
	if !ok {
		return "", ErrForbidden
	}
	member := false
	if err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		ex, err := tx.NewSelect().Model((*models.WorkspaceMember)(nil)).
			Where("workspace_id = ?", wsID).Where("user_id = ?", actor).Exists(ctx)
		member = ex
		return err
	}); err != nil {
		return "", err
	}
	if !member {
		return "", ErrForbidden
	}
	return s.store.PresignGet(ctx, key, time.Hour)
}

// workspaceFromKey извлекает wsID из ключа объекта ws/<wsID>/<obj>/<name>.
func workspaceFromKey(key string) (uuid.UUID, bool) {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) < 2 || parts[0] != "ws" {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, false
	}
	return id, true
}

// SetUserAvatar — ключ объекта аватарки текущего пользователя (users без RLS).
func (s *Service) SetUserAvatar(ctx context.Context, actor, wsID uuid.UUID, key string) error {
	var val any = key
	if key == "" {
		val = nil
	}
	_, err := s.db.NewUpdate().Table("users").
		Set("avatar_key = ?", val).Set("updated_at = now()").Where("id = ?", actor).Exec(ctx)
	if err == nil {
		_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "user", EntityID: actor.String(), Op: "update"})
	}
	return err
}

// SetWorkspaceLogo — ключ объекта лого воркспейса.
func (s *Service) SetWorkspaceLogo(ctx context.Context, actor, wsID uuid.UUID, key string) error {
	var val any = key
	if key == "" {
		val = nil
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, e := tx.NewUpdate().Model((*models.Workspace)(nil)).
			Set("logo_key = ?", val).Set("updated_at = now()").Where("id = ?", wsID).Exec(ctx)
		return e
	})
	if err == nil {
		_ = s.pub.PublishDelta(ctx, wsID, realtime.Delta{EntityType: "workspace", EntityID: wsID.String(), Op: "update"})
	}
	return err
}

// Package apitokens — управление Personal Access Tokens публичного API.
// Таблица api_tokens без RLS (auth-артефакт, как users); хранится только хеш.
package apitokens

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/token"
)

var ErrName = errors.New("token name required")

type Service struct{ db *bun.DB }

func NewService(bdb *bun.DB) *Service { return &Service{db: bdb} }

// Create выпускает токен, возвращает RAW-секрет (показывается один раз) + id.
func (s *Service) Create(ctx context.Context, actor, wsID, id uuid.UUID, name string, expiresAt *time.Time) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrName
	}
	raw, hash := token.Generate("pms_")
	at := &models.APIToken{
		ID: id, WorkspaceID: wsID, UserID: actor, Name: name,
		TokenHash: hash, Prefix: raw[:12], ExpiresAt: expiresAt, CreatedAt: time.Now().UTC(),
	}
	if _, err := s.db.NewInsert().Model(at).Exec(ctx); err != nil {
		return "", err
	}
	return raw, nil
}

// List — активные (не отозванные) токены воркспейса (без секретов).
func (s *Service) List(ctx context.Context, wsID uuid.UUID) ([]models.APIToken, error) {
	out := []models.APIToken{}
	err := s.db.NewSelect().Model(&out).
		Where("workspace_id = ?", wsID).Where("revoked_at IS NULL").
		Order("created_at DESC").Scan(ctx)
	return out, err
}

// Revoke отзывает токен (только владелец). Возвращает число затронутых строк.
func (s *Service) Revoke(ctx context.Context, actor, wsID, id uuid.UUID) (int64, error) {
	res, err := s.db.NewUpdate().Model((*models.APIToken)(nil)).
		Set("revoked_at = now()").
		Where("id = ?", id).Where("workspace_id = ?", wsID).Where("user_id = ?", actor).Where("revoked_at IS NULL").
		Exec(ctx)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

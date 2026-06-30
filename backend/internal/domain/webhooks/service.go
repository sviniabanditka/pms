// Package webhooks — управление вебхуками воркспейса (CRUD). Доставку выполняет
// фоновый диспетчер (jobs), читающий change_log; этот сервис — только конфиг.
package webhooks

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/token"
)

var (
	ErrURL    = errors.New("https url required")
	ErrEvents = errors.New("at least one event required")
)

type Service struct{ db *bun.DB }

func NewService(bdb *bun.DB) *Service { return &Service{db: bdb} }

// Matches — подписан ли набор events на событие <entityType>.<op>.
// Поддержка: '*' (всё), 'issue.*' (все события сущности), 'issue.create' (точное).
func Matches(events []string, entityType, op string) bool {
	exact := entityType + "." + op
	wild := entityType + ".*"
	for _, e := range events {
		if e == "*" || e == wild || e == exact {
			return true
		}
	}
	return false
}

// Create регистрирует вебхук, возвращает RAW-секрет (показывается один раз).
func (s *Service) Create(ctx context.Context, actor, wsID, id uuid.UUID, url string, events []string) (string, error) {
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return "", ErrURL
	}
	if len(events) == 0 {
		return "", ErrEvents
	}
	secret, _ := token.Generate("whsec_")
	now := time.Now().UTC()
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, e := tx.NewInsert().Model(&models.Webhook{
			ID: id, WorkspaceID: wsID, URL: url, Secret: secret, Events: events,
			Active: true, CreatedBy: actor, CreatedAt: now, UpdatedAt: now,
		}).Exec(ctx)
		return e
	})
	if err != nil {
		return "", err
	}
	return secret, nil
}

func (s *Service) List(ctx context.Context, wsID uuid.UUID) ([]models.Webhook, error) {
	out := []models.Webhook{}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewSelect().Model(&out).Where("deleted_at IS NULL").Order("created_at DESC").Scan(ctx)
	})
	return out, err
}

// Update патчит url/events/active.
func (s *Service) Update(ctx context.Context, actor, wsID, id uuid.UUID, url *string, events []string, active *bool) error {
	return db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewUpdate().Model((*models.Webhook)(nil)).Where("id = ?", id).Set("updated_at = now()")
		if url != nil && strings.HasPrefix(*url, "http") {
			q.Set("url = ?", strings.TrimSpace(*url))
		}
		if events != nil {
			q.Set("events = ?", bun.NullZero(events))
		}
		if active != nil {
			q.Set("active = ?", *active)
		}
		_, e := q.Exec(ctx)
		return e
	})
}

func (s *Service) Delete(ctx context.Context, actor, wsID, id uuid.UUID) error {
	return db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, e := tx.NewUpdate().Model((*models.Webhook)(nil)).
			Set("deleted_at = now()").Set("active = false").Where("id = ?", id).Exec(ctx)
		return e
	})
}

// Deliveries — последние записи лога доставок вебхука.
func (s *Service) Deliveries(ctx context.Context, wsID, webhookID uuid.UUID) ([]models.WebhookDelivery, error) {
	out := []models.WebhookDelivery{}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewSelect().Model(&out).Where("webhook_id = ?", webhookID).Order("created_at DESC").Limit(20).Scan(ctx)
	})
	return out, err
}

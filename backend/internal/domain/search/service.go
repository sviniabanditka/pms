// Package search — серверный полнотекстовый-lite поиск (ILIKE) по issues/docs/channels.
// ponytail: ILIKE-скан под текущий масштаб; при росте — pg_trgm/tsvector или OpenSearch (D21).
package search

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/filter"
	"github.com/pms/backend/internal/models"
	"github.com/uptrace/bun"
)

type Service struct{ db *bun.DB }

func NewService(bdb *bun.DB) *Service { return &Service{db: bdb} }

type Result struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	TeamID   string `json:"teamId,omitempty"`
}

// RunView выполняет View server-side: компилирует Filter-AST → SQL и возвращает issues (JSON).
// Только data_source="issue" (доводка для масштаба/экспорта, D21). Группировка — на клиенте.
func (s *Service) RunView(ctx context.Context, actor, wsID uuid.UUID, dataSource string, node *filter.Node, limit int) ([]byte, error) {
	if dataSource != "" && dataSource != "issue" {
		return nil, errors.New("unsupported data_source")
	}
	where, args, err := filter.Compile(node)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	var iss []models.Issue
	err = db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewSelect().Model(&iss).Where("archived_at IS NULL AND deleted_at IS NULL")
		if where != "" {
			q = q.Where(where, args...)
		}
		return q.OrderExpr("board_position ASC").Limit(limit).Scan(ctx)
	})
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(iss)
	if raw == nil || string(raw) == "null" {
		return []byte("[]"), nil
	}
	return raw, nil
}

func (s *Service) Search(ctx context.Context, actor, wsID uuid.UUID, query string) ([]byte, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return []byte("[]"), nil
	}
	like := "%" + q + "%"
	out := []Result{}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		var iss []models.Issue
		if err := tx.NewSelect().Model(&iss).
			Where("archived_at IS NULL AND deleted_at IS NULL").
			Where("identifier ILIKE ? OR title ILIKE ?", like, like).
			Order("updated_at DESC").Limit(12).Scan(ctx); err != nil {
			return err
		}
		for _, i := range iss {
			out = append(out, Result{Type: "issue", ID: i.ID.String(), Title: i.Title, Subtitle: i.Identifier, TeamID: i.TeamID.String()})
		}
		var docs []models.Document
		if err := tx.NewSelect().Model(&docs).Where("deleted_at IS NULL").Where("title ILIKE ?", like).Limit(8).Scan(ctx); err != nil {
			return err
		}
		for _, d := range docs {
			tid := ""
			if d.TeamID != nil {
				tid = d.TeamID.String()
			}
			out = append(out, Result{Type: "doc", ID: d.ID.String(), Title: d.Title, Subtitle: "Doc", TeamID: tid})
		}
		var chans []models.Channel
		if err := tx.NewSelect().Model(&chans).Where("deleted_at IS NULL AND kind='channel'").Where("name ILIKE ?", like).Limit(6).Scan(ctx); err != nil {
			return err
		}
		for _, c := range chans {
			out = append(out, Result{Type: "channel", ID: c.ID.String(), Title: c.Name, Subtitle: "Channel"})
		}
		// Сообщения чата: тело ILIKE, только в видимых каналах (публичный канал ИЛИ участник DM).
		var msgs []struct {
			ChannelID   uuid.UUID `bun:"channel_id"`
			Body        string    `bun:"body"`
			ChannelName string    `bun:"channel_name"`
		}
		if err := tx.NewSelect().Model((*models.Message)(nil)).
			ColumnExpr("msg.channel_id").ColumnExpr("msg.body").ColumnExpr("ch.name AS channel_name").
			Join("JOIN channels ch ON ch.id = msg.channel_id").
			Where("msg.deleted_at IS NULL").Where("msg.body ILIKE ?", like).
			Where("ch.kind = 'channel' OR EXISTS (SELECT 1 FROM channel_members cmx WHERE cmx.channel_id = msg.channel_id AND cmx.user_id = ?)", actor).
			Order("msg.created_at DESC").Limit(8).Scan(ctx, &msgs); err != nil {
			return err
		}
		for _, m := range msgs {
			title := strings.TrimSpace(m.Body)
			if len(title) > 80 {
				title = title[:80] + "…"
			}
			sub := "Message"
			if m.ChannelName != "" {
				sub = "Message in #" + m.ChannelName
			}
			// ID = channelId → клик открывает канал.
			out = append(out, Result{Type: "message", ID: m.ChannelID.String(), Title: title, Subtitle: sub})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(out)
	return raw, nil
}

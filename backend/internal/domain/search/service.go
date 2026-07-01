// Package search — серверный полнотекстовый-lite поиск (ILIKE) по issues/docs/channels.
// ponytail: ILIKE-скан под текущий масштаб; при росте — pg_trgm/tsvector или OpenSearch (D21).
package search

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/db"
	"github.com/pms/backend/internal/embed"
	"github.com/pms/backend/internal/filter"
	"github.com/pms/backend/internal/models"
	"github.com/uptrace/bun"
)

var (
	reMention = regexp.MustCompile(`@\[([^\]]*)\]\([^)]*\)`) // @[Name](id) → @Name
	reImage   = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)   // ![alt](url) → (убрать)
	reLink    = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)  // [text](url) → text
	reWS      = regexp.MustCompile(`\s+`)
)

// cleanSnippet — markdown/mention/url → читаемый текст для превью (≤80).
func cleanSnippet(body string) string {
	s := reMention.ReplaceAllString(body, "@$1")
	s = reImage.ReplaceAllString(s, "")
	s = reLink.ReplaceAllString(s, "$1")
	s = strings.NewReplacer("#", "", "*", "", "_", "", "`", "", "~", "", ">", "").Replace(s)
	s = strings.TrimSpace(reWS.ReplaceAllString(s, " "))
	if len([]rune(s)) > 80 {
		s = string([]rune(s)[:80]) + "…"
	}
	return s
}

type Service struct {
	db    *bun.DB
	embed *embed.Client
}

func NewService(bdb *bun.DB, e *embed.Client) *Service { return &Service{db: bdb, embed: e} }

var ErrNoEmbed = errors.New("embeddings unavailable")

// SemanticSearch — семантический поиск issue по эмбеддингу запроса (cosine, HNSW).
// Изоляция по воркспейсу — через RLS (db.WithWorkspace). Возвращает JSON-массив issue.
func (s *Service) SemanticSearch(ctx context.Context, actor, wsID uuid.UUID, query string, limit int) ([]byte, error) {
	if s.embed == nil {
		return nil, ErrNoEmbed
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	vecs, err := s.embed.Embed(ctx, []string{query}, "query")
	if err != nil || len(vecs) == 0 {
		return nil, ErrNoEmbed
	}
	lit := embed.VecLiteral(vecs[0])
	var iss []models.Issue
	err = db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewSelect().Model(&iss).
			Where("archived_at IS NULL AND deleted_at IS NULL AND embedding IS NOT NULL").
			OrderExpr("embedding <=> ?::vector", lit).Limit(limit).Scan(ctx)
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

// SimilarIssues — похожие на заданный issue (по его вектору, исключая сам issue).
func (s *Service) SimilarIssues(ctx context.Context, actor, wsID, issueID uuid.UUID, limit int) ([]byte, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	var iss []models.Issue
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewSelect().Model(&iss).
			Where("archived_at IS NULL AND deleted_at IS NULL AND embedding IS NOT NULL").
			Where("id <> ?", issueID).
			Where("(SELECT embedding FROM issues WHERE id = ?) IS NOT NULL", issueID).
			OrderExpr("embedding <=> (SELECT embedding FROM issues WHERE id = ?)", issueID).
			Limit(limit).Scan(ctx)
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
		// FTS по телу (title+description) с ранжированием; ILIKE по identifier; trgm для опечаток.
		var iss []models.Issue
		if err := tx.NewSelect().Model(&iss).
			Where("archived_at IS NULL AND deleted_at IS NULL").
			Where("search_vector @@ websearch_to_tsquery('simple', ?) OR identifier ILIKE ? OR title % ?", q, like, q).
			OrderExpr("ts_rank(search_vector, websearch_to_tsquery('simple', ?)) DESC, updated_at DESC", q).
			Limit(12).Scan(ctx); err != nil {
			return err
		}
		for _, i := range iss {
			out = append(out, Result{Type: "issue", ID: i.ID.String(), Title: i.Title, Subtitle: i.Identifier, TeamID: i.TeamID.String()})
		}
		// FTS по телу дока (title+body); раньше искались только заголовки.
		var docs []models.Document
		if err := tx.NewSelect().Model(&docs).Where("deleted_at IS NULL").
			Where("search_vector @@ websearch_to_tsquery('simple', ?) OR title % ?", q, q).
			OrderExpr("ts_rank(search_vector, websearch_to_tsquery('simple', ?)) DESC", q).
			Limit(8).Scan(ctx); err != nil {
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
		// Матч по ВИДИМОМУ тексту: убираем "](payload)" mention/link/image (id/url не должны
		// ловиться в поиске, напр. uuid внутри @[Name](id)).
		if err := tx.NewSelect().Model((*models.Message)(nil)).
			ColumnExpr("msg.channel_id").ColumnExpr("msg.body").ColumnExpr("ch.name AS channel_name").
			Join("JOIN channels ch ON ch.id = msg.channel_id").
			Where("msg.deleted_at IS NULL").
			Where(`regexp_replace(msg.body, '\]\([^)]*\)', '', 'g') ILIKE ?`, like).
			Where("ch.kind = 'channel' OR EXISTS (SELECT 1 FROM channel_members cmx WHERE cmx.channel_id = msg.channel_id AND cmx.user_id = ?)", actor).
			Order("msg.created_at DESC").Limit(8).Scan(ctx, &msgs); err != nil {
			return err
		}
		for _, m := range msgs {
			title := cleanSnippet(m.Body)
			if title == "" {
				continue
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

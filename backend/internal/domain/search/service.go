// Package search — серверный полнотекстовый-lite поиск (ILIKE) по issues/docs/channels.
// ponytail: ILIKE-скан под текущий масштаб; при росте — pg_trgm/tsvector или OpenSearch (D21).
package search

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
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

// SemanticSearch — cross-entity семантический поиск (issue/doc/comment) через единую
// таблицу embeddings. RLS-scoped. Возвращает JSON Result[]: comment → навигация на
// родительский issue, doc → сам doc.
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

	var ranked []struct {
		EntityType string    `bun:"entity_type"`
		EntityID   uuid.UUID `bun:"entity_id"`
	}
	results := []Result{}
	err = db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		if err := tx.NewRaw(`SELECT entity_type, entity_id FROM embeddings ORDER BY embedding <=> ?::vector LIMIT ?`, lit, limit).Scan(ctx, &ranked); err != nil {
			return err
		}
		if len(ranked) == 0 {
			return nil
		}
		var issueIDs, docIDs, commentIDs []uuid.UUID
		for _, r := range ranked {
			switch r.EntityType {
			case "issue":
				issueIDs = append(issueIDs, r.EntityID)
			case "doc":
				docIDs = append(docIDs, r.EntityID)
			case "comment":
				commentIDs = append(commentIDs, r.EntityID)
			}
		}
		byID := map[string]Result{}
		if len(issueIDs) > 0 {
			var rows []struct {
				ID         uuid.UUID `bun:"id"`
				Identifier string    `bun:"identifier"`
				Title      string    `bun:"title"`
				TeamID     uuid.UUID `bun:"team_id"`
			}
			_ = tx.NewSelect().Table("issues").Column("id", "identifier", "title", "team_id").Where("id IN (?)", bun.In(issueIDs)).Scan(ctx, &rows)
			for _, r := range rows {
				byID["issue:"+r.ID.String()] = Result{Type: "issue", ID: r.ID.String(), Title: r.Identifier + " " + r.Title, TeamID: r.TeamID.String()}
			}
		}
		if len(docIDs) > 0 {
			var rows []struct {
				ID    uuid.UUID `bun:"id"`
				Title string    `bun:"title"`
			}
			_ = tx.NewSelect().Table("documents").Column("id", "title").Where("id IN (?)", bun.In(docIDs)).Scan(ctx, &rows)
			for _, r := range rows {
				t := r.Title
				if t == "" {
					t = "Untitled doc"
				}
				byID["doc:"+r.ID.String()] = Result{Type: "doc", ID: r.ID.String(), Title: "Doc · " + t}
			}
		}
		if len(commentIDs) > 0 {
			var rows []struct {
				ID      uuid.UUID `bun:"id"`
				IssueID uuid.UUID `bun:"issue_id"`
				Body    string    `bun:"body"`
			}
			_ = tx.NewSelect().Table("comments").Column("id", "issue_id", "body").Where("id IN (?)", bun.In(commentIDs)).Scan(ctx, &rows)
			for _, r := range rows {
				byID["comment:"+r.ID.String()] = Result{Type: "issue", ID: r.IssueID.String(), Title: "💬 " + cleanSnippet(r.Body)}
			}
		}
		for _, r := range ranked {
			if res, ok := byID[r.EntityType+":"+r.EntityID.String()]; ok {
				results = append(results, res)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(results)
	return raw, nil
}

// SimilarIssues — похожие на заданный issue (по вектору из embeddings, исключая сам).
func (s *Service) SimilarIssues(ctx context.Context, actor, wsID, issueID uuid.UUID, limit int) ([]byte, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	var iss []models.Issue
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewSelect().Model(&iss).
			Join("JOIN embeddings e ON e.entity_type = 'issue' AND e.entity_id = ?TableAlias.id").
			Where("?TableAlias.id <> ?", issueID).
			Where("?TableAlias.deleted_at IS NULL AND ?TableAlias.archived_at IS NULL").
			Where("EXISTS (SELECT 1 FROM embeddings WHERE entity_type='issue' AND entity_id = ?)", issueID).
			OrderExpr("e.embedding <=> (SELECT embedding FROM embeddings WHERE entity_type='issue' AND entity_id = ?)", issueID).
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

// Suggest — kNN-подсказки для черновика issue (AI Фаза 2): по тексту находим
// ближайшие issue, агрегируем их team/assignee/labels (вес = 1/(rank+1)) →
// топ-предложения + similar (возможные дубли). RLS-scoped.
func (s *Service) Suggest(ctx context.Context, actor, wsID uuid.UUID, text, teamID string, limit int) ([]byte, error) {
	if s.embed == nil {
		return nil, ErrNoEmbed
	}
	if limit <= 0 || limit > 10 {
		limit = 6
	}
	const k = 15
	vecs, err := s.embed.Embed(ctx, []string{text}, "query")
	if err != nil || len(vecs) == 0 {
		return nil, ErrNoEmbed
	}
	lit := embed.VecLiteral(vecs[0])

	var neigh []models.Issue
	var labelRows []struct {
		IssueID uuid.UUID `bun:"issue_id"`
		LabelID uuid.UUID `bun:"label_id"`
	}
	err = db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		q := tx.NewSelect().Model(&neigh).
			Join("JOIN embeddings e ON e.entity_type = 'issue' AND e.entity_id = ?TableAlias.id").
			Where("?TableAlias.deleted_at IS NULL AND ?TableAlias.archived_at IS NULL")
		if teamID != "" {
			q = q.Where("?TableAlias.team_id = ?", teamID)
		}
		if err := q.OrderExpr("e.embedding <=> ?::vector", lit).Limit(k).Scan(ctx); err != nil {
			return err
		}
		if len(neigh) == 0 {
			return nil
		}
		ids := make([]uuid.UUID, len(neigh))
		for i, n := range neigh {
			ids[i] = n.ID
		}
		return tx.NewSelect().Table("issue_labels").Column("issue_id", "label_id").
			Where("issue_id IN (?)", bun.In(ids)).Scan(ctx, &labelRows)
	})
	if err != nil {
		return nil, err
	}

	labelsByIssue := map[uuid.UUID][]uuid.UUID{}
	for _, lr := range labelRows {
		labelsByIssue[lr.IssueID] = append(labelsByIssue[lr.IssueID], lr.LabelID)
	}
	teamW := map[uuid.UUID]float64{}
	asgW := map[uuid.UUID]float64{}
	labW := map[uuid.UUID]float64{}
	for i, n := range neigh {
		w := 1.0 / float64(i+1) // rank-вес: ближайший весит больше
		teamW[n.TeamID] += w
		if n.AssigneeID != nil {
			asgW[*n.AssigneeID] += w
		}
		for _, l := range labelsByIssue[n.ID] {
			labW[l] += w
		}
	}
	similar := neigh
	if len(similar) > limit {
		similar = similar[:limit]
	}
	out := map[string]any{
		"teamId":     argmaxID(teamW),
		"assigneeId": argmaxID(asgW),
		"labelIds":   topNIDs(labW, 3),
		"similar":    similar,
	}
	// Фаза 3: предсказания обученных моделей (priority/estimate/slip), если есть.
	pv := [][]float32{vecs[0]}
	if p, c, e := s.embed.Predict(ctx, wsID.String(), "priority", pv); e == nil && len(p) > 0 && len(c) > 0 && c[0] >= 0.35 {
		out["priority"] = int(p[0] + 0.5)
		out["priorityConfidence"] = c[0]
	}
	if p, _, e := s.embed.Predict(ctx, wsID.String(), "estimate", pv); e == nil && len(p) > 0 {
		out["estimate"] = p[0]
	}
	if p, c, e := s.embed.Predict(ctx, wsID.String(), "slip", pv); e == nil && len(p) > 0 && len(c) > 0 {
		if int(p[0]+0.5) == 1 && c[0] >= 0.55 {
			out["slipRisk"] = true
			out["slipConfidence"] = c[0]
		}
	}
	raw, _ := json.Marshal(out)
	return raw, nil
}

func argmaxID(m map[uuid.UUID]float64) string {
	var best uuid.UUID
	var bw float64
	for id, w := range m {
		if w > bw {
			bw, best = w, id
		}
	}
	if best == uuid.Nil {
		return ""
	}
	return best.String()
}

func topNIDs(m map[uuid.UUID]float64, n int) []string {
	type kv struct {
		id uuid.UUID
		w  float64
	}
	arr := make([]kv, 0, len(m))
	for id, w := range m {
		arr = append(arr, kv{id, w})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].w > arr[j].w })
	out := []string{}
	for i := 0; i < len(arr) && i < n; i++ {
		out = append(out, arr[i].id.String())
	}
	return out
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

// LogFeedback — записать реакцию пользователя на AI-подсказку (петля обратной связи).
func (s *Service) LogFeedback(ctx context.Context, actor, wsID uuid.UUID, kind, action string, entityID *uuid.UUID, suggested string) error {
	return db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewRaw(
			`INSERT INTO ai_feedback (workspace_id, actor_id, kind, entity_id, suggested, action) VALUES (?, ?, ?, ?, ?, ?)`,
			wsID, actor, kind, entityID, suggested, action,
		).Exec(ctx)
		return err
	})
}

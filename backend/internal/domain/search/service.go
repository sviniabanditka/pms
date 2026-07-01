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

// cleanSnippetFull — markdown/mention/url → читаемый текст (без обрезки).
func cleanSnippetFull(body string) string {
	s := reMention.ReplaceAllString(body, "@$1")
	s = reImage.ReplaceAllString(s, "")
	s = reLink.ReplaceAllString(s, "$1")
	s = strings.NewReplacer("#", "", "*", "", "_", "", "`", "", "~", "", ">", "").Replace(s)
	return strings.TrimSpace(reWS.ReplaceAllString(s, " "))
}

// cleanSnippet — то же, обрезано под превью (≤80).
func cleanSnippet(body string) string {
	s := cleanSnippetFull(body)
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

// stopwords для лейблов тем (en + ru общие) — не тащим отдельную либу.
var themeStop = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "add": true, "fix": true, "issue": true,
	"you": true, "are": true, "not": true, "can": true, "should": true, "under": true, "then": true, "since": true,
	"это": true, "для": true, "как": true, "или": true, "что": true, "into": true, "from": true,
	"remove": true, "update": true, "support": true, "implement": true, "handle": true, "improve": true, "document": true,
}

func themeLabel(titles []string) string {
	freq := map[string]int{}
	for _, t := range titles {
		for _, w := range reWS.Split(strings.ToLower(t), -1) {
			w = strings.Trim(w, ".,:;!?()[]{}\"'`")
			if len([]rune(w)) < 3 || themeStop[w] {
				continue
			}
			freq[w]++
		}
	}
	type kv struct {
		w string
		n int
	}
	arr := make([]kv, 0, len(freq))
	for w, n := range freq {
		arr = append(arr, kv{w, n})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].n > arr[j].n })
	top := []string{}
	for i := 0; i < len(arr) && i < 3; i++ {
		top = append(top, arr[i].w)
	}
	if len(top) == 0 {
		return "misc"
	}
	return strings.Join(top, " · ")
}

type Theme struct {
	Label  string   `json:"label"`
	Size   int      `json:"size"`
	Sample []string `json:"sample"`
}

// Themes — топик-темы воркспейса (AI Фаза 4): кластеры issue_clusters, лейбл из
// частых слов заголовков, sample-идентификаторы. RLS-scoped.
func (s *Service) Themes(ctx context.Context, actor, wsID uuid.UUID) ([]byte, error) {
	var rows []struct {
		Cluster    int    `bun:"cluster"`
		Identifier string `bun:"identifier"`
		Title      string `bun:"title"`
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewRaw(
			`SELECT ic.cluster, i.identifier, i.title
			 FROM issue_clusters ic JOIN issues i ON i.id = ic.issue_id
			 WHERE ic.cluster >= 0 AND i.deleted_at IS NULL AND i.archived_at IS NULL`).Scan(ctx, &rows)
	})
	if err != nil {
		return nil, err
	}
	type agg struct {
		titles []string
		ident  []string
	}
	byCluster := map[int]*agg{}
	for _, r := range rows {
		a := byCluster[r.Cluster]
		if a == nil {
			a = &agg{}
			byCluster[r.Cluster] = a
		}
		a.titles = append(a.titles, r.Title)
		if len(a.ident) < 4 {
			a.ident = append(a.ident, r.Identifier)
		}
	}
	themes := make([]Theme, 0, len(byCluster))
	for _, a := range byCluster {
		themes = append(themes, Theme{Label: themeLabel(a.titles), Size: len(a.titles), Sample: a.ident})
	}
	sort.Slice(themes, func(i, j int) bool { return themes[i].Size > themes[j].Size })
	if len(themes) > 12 {
		themes = themes[:12]
	}
	raw, _ := json.Marshal(themes)
	return raw, nil
}

// --- Фаза 4 (остаток): аномалии, резюме, sentiment ---

type Outlier struct {
	Identifier string  `json:"identifier"`
	Title      string  `json:"title"`
	Score      float64 `json:"score"`
}

// Anomalies — issue, эмбеддинги которых наиболее «выбиваются» (IsolationForest).
// RLS-scoped. Топ-N по outlier-скору (порог 0.6).
func (s *Service) Anomalies(ctx context.Context, actor, wsID uuid.UUID, limit int) ([]byte, error) {
	if s.embed == nil {
		return nil, ErrNoEmbed
	}
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	var rows []struct {
		Identifier string `bun:"identifier"`
		Title      string `bun:"title"`
		Vec        string `bun:"vec"`
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewRaw(
			`SELECT i.identifier, i.title, e.embedding::text AS vec
			 FROM issues i JOIN embeddings e ON e.entity_type='issue' AND e.entity_id = i.id
			 WHERE i.deleted_at IS NULL AND i.archived_at IS NULL`).Scan(ctx, &rows)
	})
	if err != nil {
		return nil, err
	}
	if len(rows) < 20 {
		return []byte("[]"), nil
	}
	vecs := make([][]float32, 0, len(rows))
	for _, r := range rows {
		vecs = append(vecs, embed.ParseVecLiteral(r.Vec))
	}
	scores, err := s.embed.Anomaly(ctx, vecs)
	if err != nil || len(scores) != len(rows) {
		return nil, err
	}
	out := make([]Outlier, 0, len(rows))
	for i, r := range rows {
		if scores[i] >= 0.6 {
			out = append(out, Outlier{Identifier: r.Identifier, Title: r.Title, Score: scores[i]})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > limit {
		out = out[:limit]
	}
	raw, _ := json.Marshal(out)
	return raw, nil
}

// Summarize — экстрактивное резюме описания issue (TextRank в сайдкаре). RLS-scoped.
func (s *Service) Summarize(ctx context.Context, actor, wsID, issueID uuid.UUID, maxSentences int) ([]byte, error) {
	if s.embed == nil {
		return nil, ErrNoEmbed
	}
	var row struct {
		Title string `bun:"title"`
		MD    string `bun:"md"`
	}
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewRaw(
			`SELECT title, COALESCE(description->>'md','') AS md
			 FROM issues WHERE id = ? AND deleted_at IS NULL`, issueID).Scan(ctx, &row)
	})
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(row.Title + ". " + cleanSnippetFull(row.MD))
	if maxSentences <= 0 || maxSentences > 6 {
		maxSentences = 3
	}
	sum, err := s.embed.Summarize(ctx, text, maxSentences)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(map[string]string{"summary": sum})
	return raw, nil
}

// Mood — агрегат тональности по последним комментам воркспейса (Фаза 4). RLS-scoped.
func (s *Service) Mood(ctx context.Context, actor, wsID uuid.UUID) ([]byte, error) {
	if s.embed == nil {
		return nil, ErrNoEmbed
	}
	var bodies []string
	err := db.WithWorkspace(ctx, s.db, wsID, func(ctx context.Context, tx bun.Tx) error {
		return tx.NewRaw(
			`SELECT body FROM comments WHERE deleted_at IS NULL ORDER BY created_at DESC LIMIT 200`).Scan(ctx, &bodies)
	})
	if err != nil {
		return nil, err
	}
	if len(bodies) < 5 {
		return []byte(`{"n":0,"avg":0,"pos":0,"neg":0,"neu":0}`), nil
	}
	clean := make([]string, len(bodies))
	for i, b := range bodies {
		clean[i] = cleanSnippetFull(b)
	}
	scores, err := s.embed.Sentiment(ctx, clean)
	if err != nil {
		return nil, err
	}
	var sum float64
	var pos, neg, neu int
	for _, sc := range scores {
		sum += sc
		switch {
		case sc > 0.15:
			pos++
		case sc < -0.15:
			neg++
		default:
			neu++
		}
	}
	raw, _ := json.Marshal(map[string]any{
		"n": len(scores), "avg": sum / float64(len(scores)), "pos": pos, "neg": neg, "neu": neu,
	})
	return raw, nil
}

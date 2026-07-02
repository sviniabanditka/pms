package httpx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/pms/backend/internal/auth"
	"github.com/pms/backend/internal/models"
	"github.com/pms/backend/internal/realtime"
	"github.com/pms/backend/internal/rpc"
	"github.com/pms/backend/internal/storage"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

// NewRouter собирает chi-роутер: health/readiness, auth, WS, connect-сервисы.
func NewRouter(bdb *bun.DB, rdb *redis.Client, authSvc *auth.Service, gw *realtime.Gateway, svcs rpc.Services, store *storage.Store, allowedOrigins []string) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(corsCSRF(allowedOrigins)) // CORS (credentials) + origin-check CSRF; dev=любой origin, прод=allowlist
	r.Use(authSvc.Authenticator)    // кладёт user в контекст, если есть сессия
	r.Use(auth.ReadOnlyGate)        // read-only PAT: REST-мутации запрещены (connect гейтит интерсептор)

	// Liveness — процесс жив.
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Readiness — зависимости доступны (db + redis).
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 2*time.Second)
		defer cancel()
		if err := bdb.PingContext(ctx); err != nil {
			http.Error(w, "db: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			http.Error(w, "redis: "+err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	// Auth: /auth/register, /login, /logout, /me.
	r.Mount("/auth", authSvc.Routes())

	// WS realtime (требует сессии).
	r.With(auth.RequireAuth).Get("/ws", gw.Handle)

	// «Мои воркспейсы» (для свитчера) — cross-tenant список по user_id.
	r.With(auth.RequireAuth).Get("/me/workspaces", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		list, err := svcs.Workspaces.ListForUser(req.Context(), uid)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(list)
	})

	// Метаданные воркспейса (teams/states/users/labels) для нативного клиента — unary
	// вместо bootstrap-стрима. Только участник воркспейса.
	r.With(auth.RequireAuth).Get("/me/workspaces/{id}/meta", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		wsID, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, "invalid workspace id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		meta, err := svcs.Workspaces.Meta(req.Context(), wsID)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(meta)
	})

	// Полный снапшот воркспейса (все сущности) — unary-эквивалент bootstrap-стрима
	// для нативного клиента (local-first стор). Дельты — через SyncService.Changes.
	r.With(auth.RequireAuth).Get("/me/workspaces/{id}/snapshot", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		wsID, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, "invalid workspace id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var only []string
		if q := req.URL.Query().Get("only"); q != "" {
			only = strings.Split(q, ",")
		}
		chunks, cursor, err := svcs.Sync.Bootstrap(req.Context(), uid, wsID, only)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		type outChunk struct {
			EntityType string          `json:"entityType"`
			Entities   json.RawMessage `json:"entities"`
		}
		out := struct {
			Cursor int64      `json:"cursor"`
			Chunks []outChunk `json:"chunks"`
		}{Cursor: cursor}
		for _, c := range chunks {
			out.Chunks = append(out.Chunks, outChunk{EntityType: c.EntityType, Entities: c.Entities})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	// Inbox: последние уведомления пользователя в воркспейсе (unary для нативного клиента).
	r.With(auth.RequireAuth).Get("/me/workspaces/{id}/notifications", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		wsID, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, "invalid workspace id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		list, err := svcs.Notifications.List(req.Context(), wsID, uid, 50)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if list == nil {
			list = []models.Notification{}
		}
		_ = json.NewEncoder(w).Encode(list)
	})

	// AI петля фидбэка: лог реакции пользователя на подсказку (applied|dismissed).
	r.With(auth.RequireAuth).Post("/me/workspaces/{id}/ai/feedback", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		wsID, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, "invalid workspace id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var body struct {
			Kind      string `json:"kind"`
			Action    string `json:"action"`
			EntityID  string `json:"entityId"`
			Suggested string `json:"suggested"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Kind == "" || body.Action == "" {
			http.Error(w, "kind and action required", http.StatusBadRequest)
			return
		}
		var eid *uuid.UUID
		if id, err := uuid.Parse(body.EntityID); err == nil {
			eid = &id
		}
		if err := svcs.Search.LogFeedback(req.Context(), uid, wsID, body.Kind, body.Action, eid, body.Suggested); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// Push: регистрация/снятие Expo push-токена устройства (нативный клиент).
	r.With(auth.RequireAuth).Post("/me/devices", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		var body struct {
			Token    string `json:"token"`
			Platform string `json:"platform"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil || body.Token == "" {
			http.Error(w, "token required", http.StatusBadRequest)
			return
		}
		if err := svcs.Notifications.RegisterDevice(req.Context(), uid, body.Token, body.Platform); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	r.With(auth.RequireAuth).Delete("/me/devices/{token}", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		if err := svcs.Notifications.UnregisterDevice(req.Context(), uid, chi.URLParam(req, "token")); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// Загрузка файла в S3 (multipart). Возвращает {key, filename, contentType, size}.
	// Аплоад идёт через бэкенд → нет CORS-префлайта на Garage; download — presigned GET (RPC GetAttachmentURL).
	r.With(auth.RequireAuth).Post("/files/upload", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		req.Body = http.MaxBytesReader(w, req.Body, 64<<20) // жёсткий потолок 64MB (не только in-memory буфер)
		if err := req.ParseMultipartForm(64 << 20); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		wsID, err := uuid.Parse(req.FormValue("workspace_id"))
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		// загружать в воркспейс может только его участник
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		file, hdr, err := req.FormFile("file")
		if err != nil {
			http.Error(w, "no file", http.StatusBadRequest)
			return
		}
		defer file.Close()
		ct := hdr.Header.Get("Content-Type")
		if ct == "" {
			ct = "application/octet-stream"
		}
		oid, _ := uuid.NewV7()
		// ключ: ws/<wsId>/<obj>/<имя> — uploader доступен для аудита, путь не угадывается
		key := "ws/" + wsID.String() + "/" + oid.String() + "/" + sanitizeName(hdr.Filename)
		if err := store.Put(req.Context(), key, ct, file, hdr.Size); err != nil {
			http.Error(w, "upload failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"key": key, "filename": hdr.Filename, "contentType": ct, "size": hdr.Size,
		})
	})

	// Стабильная ссылка на объект: бэкенд проксирует байты из S3 (тот же origin → <img> грузится без
	// redirect/CORS-нюансов; ссылка вечная). Без auth: ключ непредсказуем (uuid-путь).
	r.Get("/files/view/*", func(w http.ResponseWriter, req *http.Request) {
		key := chi.URLParam(req, "*")
		if key == "" {
			http.Error(w, "no key", http.StatusBadRequest)
			return
		}
		body, ct, sz, err := store.Get(req.Context(), key)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer body.Close()
		// Stored-XSS защита: запретить браузеру угадывать тип и рендерить активный контент
		// (HTML/SVG) инлайн в origin приложения. Безопасные типы (картинки/видео/pdf) — инлайн,
		// всё остальное отдаём как attachment + octet-stream.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		forceDownload := req.URL.Query().Get("dl") != "" || !safeInline(ct)
		if forceDownload {
			w.Header().Set("Content-Type", "application/octet-stream")
		} else {
			w.Header().Set("Content-Type", ct)
		}
		if sz > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(sz, 10))
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
		if forceDownload {
			name := path.Base(key)
			w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
		}
		_, _ = io.Copy(w, body)
	})

	// iCal: персональный токен фида (выдать/ротировать) — требует сессии.
	r.With(auth.RequireAuth).Get("/ical/token", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		tok, err := svcs.Calendar.EnsureFeedToken(req.Context(), uid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"token": tok, "url": "/ical/" + tok + ".ics"})
	})
	r.With(auth.RequireAuth).Post("/ical/token/rotate", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		tok, err := svcs.Calendar.RotateFeedToken(req.Context(), uid)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"token": tok, "url": "/ical/" + tok + ".ics"})
	})

	// iCal-фид (публичный по токену; календарные клиенты не шлют cookie). `.ics` суффикс опционален.
	r.Get("/ical/{token}", func(w http.ResponseWriter, req *http.Request) {
		token := strings.TrimSuffix(chi.URLParam(req, "token"), ".ics")
		ics, err := svcs.Calendar.Feed(req.Context(), token)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Cache-Control", "private, max-age=300")
		_, _ = w.Write([]byte(ics))
	})

	// Public API — управление API-токенами (PAT). За session-auth; сами токены
	// аутентифицируют внешние вызовы к connect-эндпоинтам (Authorization: Bearer pms_…).
	r.With(auth.RequireAuth).Post("/api/tokens", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		var body struct {
			WorkspaceID   string `json:"workspaceId"`
			Name          string `json:"name"`
			ExpiresInDays int    `json:"expiresInDays"`
			ReadOnly      bool   `json:"readOnly"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		wsID, err := uuid.Parse(body.WorkspaceID)
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		var exp *time.Time
		if body.ExpiresInDays > 0 {
			t := time.Now().AddDate(0, 0, body.ExpiresInDays)
			exp = &t
		}
		id, _ := uuid.NewV7()
		raw, err := svcs.ApiTokens.Create(req.Context(), uid, wsID, id, body.Name, body.ReadOnly, exp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "token": raw, "prefix": raw[:12]})
	})
	r.With(auth.RequireAuth).Get("/api/tokens", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		wsID, err := uuid.Parse(req.URL.Query().Get("workspaceId"))
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		list, err := svcs.ApiTokens.List(req.Context(), wsID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})
	r.With(auth.RequireAuth).Delete("/api/tokens/{id}", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		id, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		wsID, err := uuid.Parse(req.URL.Query().Get("workspaceId"))
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		n, err := svcs.ApiTokens.Revoke(req.Context(), uid, wsID, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"revoked": n > 0})
	})

	// Webhooks — CRUD за session-auth. Доставку выполняет River-диспетчер (change_log).
	r.With(auth.RequireAuth).Post("/api/webhooks", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		var body struct {
			WorkspaceID string   `json:"workspaceId"`
			URL         string   `json:"url"`
			Events      []string `json:"events"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		wsID, err := uuid.Parse(body.WorkspaceID)
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		id, _ := uuid.NewV7()
		secret, err := svcs.Webhooks.Create(req.Context(), uid, wsID, id, body.URL, body.Events)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "secret": secret})
	})
	r.With(auth.RequireAuth).Get("/api/webhooks", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		wsID, err := uuid.Parse(req.URL.Query().Get("workspaceId"))
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		list, err := svcs.Webhooks.List(req.Context(), wsID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})
	r.With(auth.RequireAuth).Patch("/api/webhooks/{id}", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		id, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var body struct {
			WorkspaceID string   `json:"workspaceId"`
			URL         *string  `json:"url"`
			Events      []string `json:"events"`
			Active      *bool    `json:"active"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		wsID, err := uuid.Parse(body.WorkspaceID)
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := svcs.Webhooks.Update(req.Context(), uid, wsID, id, body.URL, body.Events, body.Active); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	r.With(auth.RequireAuth).Delete("/api/webhooks/{id}", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		id, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		wsID, err := uuid.Parse(req.URL.Query().Get("workspaceId"))
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		if err := svcs.Webhooks.Delete(req.Context(), uid, wsID, id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	r.With(auth.RequireAuth).Get("/api/webhooks/{id}/deliveries", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		id, err := uuid.Parse(chi.URLParam(req, "id"))
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		wsID, err := uuid.Parse(req.URL.Query().Get("workspaceId"))
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
			return
		}
		if member, err := svcs.Workspaces.IsMember(req.Context(), wsID, uid); err != nil || !member || !auth.WorkspaceAllowed(req.Context(), wsID) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		list, err := svcs.Webhooks.Deliveries(req.Context(), wsID, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	})

	// ConnectRPC-сервисы. Streaming-ответы (bootstrap) жмём gzip на HTTP-уровне
	// (Content-Encoding) — браузер декодит прозрачно; connect-web не умеет
	// connect-protocol compression, а bootstrap большой (×6-7 экономии). Unary
	// connect-go жмёт сам через Accept-Encoding, поэтому в allowlist только
	// streaming-типы (иначе риск двойного сжатия).
	r.Group(func(cr chi.Router) {
		cr.Use(middleware.Compress(5, "application/connect+json", "application/connect+proto"))
		rpc.Mount(cr, svcs)
	})

	return r
}

// safeInline — типы, которые браузер может безопасно показать инлайн (без активного скрипта).
// image/svg+xml исключён намеренно: SVG может нести <script>.
func safeInline(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch ct {
	case "image/svg+xml":
		return false
	case "application/pdf", "text/plain":
		return true
	}
	return strings.HasPrefix(ct, "image/") || strings.HasPrefix(ct, "video/") || strings.HasPrefix(ct, "audio/")
}

func sanitizeName(name string) string {
	name = path.Base(name)
	name = strings.ReplaceAll(name, " ", "_")
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	return name
}

// corsCSRF — CORS с поддержкой credentials (cookie-auth) + origin-check CSRF.
// allowed пуст (dev): любой Origin отражается (echo) → удобно с любого порта.
// allowed задан (прод): только из списка; чужой Origin на мутирующем запросе → 403 (CSRF).
// Cookie сессии = SameSite=Lax (браузер не шлёт её на cross-site POST) + этот origin-check = двойная защита.
func corsCSRF(allowed []string) func(http.Handler) http.Handler {
	allow := func(origin string) bool {
		if origin == "" {
			return false
		}
		if len(allowed) == 0 {
			return true // dev: любой origin
		}
		for _, a := range allowed {
			if a == origin {
				return true
			}
		}
		return false
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			ok := allow(origin)
			h := w.Header()
			if ok {
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Credentials", "true")
				h.Set("Vary", "Origin")
				h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				if reqH := r.Header.Get("Access-Control-Request-Headers"); reqH != "" {
					h.Set("Access-Control-Allow-Headers", reqH)
				} else {
					h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Connect-Protocol-Version, Connect-Timeout-Ms")
				}
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			// CSRF: мутирующий запрос с чужим Origin в проде → отказ. Запросы без Origin
			// (curl/CLI с Bearer) проходят — у них нет cookie, отправленной браузером.
			if len(allowed) > 0 && origin != "" && !ok && r.Method != http.MethodGet && r.Method != http.MethodHead {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

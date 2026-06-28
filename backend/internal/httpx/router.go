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
	"github.com/pms/backend/internal/realtime"
	"github.com/pms/backend/internal/rpc"
	"github.com/pms/backend/internal/storage"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
)

// NewRouter собирает chi-роутер: health/readiness, auth, WS, connect-сервисы.
func NewRouter(bdb *bun.DB, rdb *redis.Client, authSvc *auth.Service, gw *realtime.Gateway, svcs rpc.Services, store *storage.Store) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(corsDev)               // dev CORS для SPA на :3000 (Bearer, без credentials)
	r.Use(authSvc.Authenticator) // кладёт user в контекст, если есть сессия

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

	// Загрузка файла в S3 (multipart). Возвращает {key, filename, contentType, size}.
	// Аплоад идёт через бэкенд → нет CORS-префлайта на Garage; download — presigned GET (RPC GetAttachmentURL).
	r.With(auth.RequireAuth).Post("/files/upload", func(w http.ResponseWriter, req *http.Request) {
		uid, _ := auth.UserID(req.Context())
		if err := req.ParseMultipartForm(64 << 20); err != nil { // до 64MB
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		wsID, err := uuid.Parse(req.FormValue("workspace_id"))
		if err != nil {
			http.Error(w, "invalid workspace_id", http.StatusBadRequest)
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
		_ = uid
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
		w.Header().Set("Content-Type", ct)
		if sz > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(sz, 10))
		}
		w.Header().Set("Cache-Control", "private, max-age=3600")
		if req.URL.Query().Get("dl") != "" {
			name := path.Base(key)
			w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
		}
		_, _ = io.Copy(w, body)
	})

	// ConnectRPC-сервисы.
	rpc.Mount(r, svcs)

	return r
}

func sanitizeName(name string) string {
	name = path.Base(name)
	name = strings.ReplaceAll(name, " ", "_")
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	return name
}

// corsDev — разрешающий CORS для локальной разработки (SPA на :3000, Bearer-токен).
// В проде — same-origin или строгая политика; это только dev-удобство.
func corsDev(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if reqH := r.Header.Get("Access-Control-Request-Headers"); reqH != "" {
			h.Set("Access-Control-Allow-Headers", reqH)
		} else {
			h.Set("Access-Control-Allow-Headers", "*")
		}
		h.Set("Access-Control-Expose-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

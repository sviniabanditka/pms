package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/pms/backend/internal/models"
	tokenpkg "github.com/pms/backend/internal/token"
)

const cookieName = "pms_session"

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxToken
	ctxReadOnly
	ctxTokenWS
)

// Authenticator извлекает сессию (cookie | Bearer | ?token=) и кладёт user_id в контекст.
// Не отклоняет анонимов — это делает RequireAuth (для гибкости публичных роутов).
// Bearer/query нужны не-браузерным клиентам (WS из CLI, мобильные).
func (s *Service) Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := extractToken(r)
		switch {
		case strings.HasPrefix(tok, "pms_"):
			// API-токен публичного API → user_id (членство энфорсит интерсептор).
			if at, ok := s.resolveAPIToken(r.Context(), tok); ok {
				ctx := WithUserID(r.Context(), at.UserID)
				if at.ReadOnly {
					ctx = WithReadOnly(ctx)
				}
				// PAT выпущен для конкретного воркспейса — привязываем; интерсептор
				// и REST-роуты сверяют wsID запроса с этой привязкой.
				ctx = context.WithValue(ctx, ctxTokenWS, at.WorkspaceID)
				r = r.WithContext(ctx)
			}
		case tok != "":
			if sess, err := s.sessions.Get(r.Context(), tok); err == nil && sess != nil {
				ctx := context.WithValue(WithUserID(r.Context(), sess.UserID), ctxToken, tok)
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func extractToken(r *http.Request) string {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

// RequireAuth отклоняет запросы без валидной сессии.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserID(r.Context()); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// resolveAPIToken — raw `pms_…` → токен, если валиден (не отозван, не истёк).
func (s *Service) resolveAPIToken(ctx context.Context, raw string) (*models.APIToken, bool) {
	var at models.APIToken
	err := s.db.NewSelect().Model(&at).
		Where("token_hash = ?", tokenpkg.Hash(raw)).
		Where("revoked_at IS NULL").
		Where("expires_at IS NULL OR expires_at > now()").
		Limit(1).Scan(ctx)
	if err != nil {
		return nil, false
	}
	// best-effort отметка использования (не блокируем запрос; фоновый ctx)
	_, _ = s.db.NewUpdate().Model((*models.APIToken)(nil)).
		Set("last_used_at = now()").Where("id = ?", at.ID).Exec(context.Background())
	return &at, true
}

// TokenWorkspace — воркспейс, к которому привязан API-токен запроса (если это PAT).
func TokenWorkspace(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxTokenWS).(uuid.UUID)
	return v, ok
}

// WorkspaceAllowed — false, если запрос аутентифицирован PAT'ом другого воркспейса.
// Для сессий (нет привязки) всегда true; членство проверяется отдельно.
func WorkspaceAllowed(ctx context.Context, wsID uuid.UUID) bool {
	tws, ok := TokenWorkspace(ctx)
	return !ok || tws == wsID
}

// ReadOnlyGate — read-only PAT не может дёргать мутирующие REST-эндпоинты
// (POST /api/tokens, вебхуки, upload, devices…). Connect-процедуры (/pms.v1.*)
// гейтит unary-интерсептор, поэтому их POST'ы пропускаем.
func ReadOnlyGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if IsReadOnly(r.Context()) &&
			r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions &&
			!strings.HasPrefix(r.URL.Path, "/pms.v1.") {
			http.Error(w, "read-only API token", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// WithReadOnly помечает контекст как read-only (read-only API-токен).
func WithReadOnly(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxReadOnly, true)
}

// IsReadOnly — read-only ли вызов (мутации запрещены интерсептором).
func IsReadOnly(ctx context.Context) bool {
	v, _ := ctx.Value(ctxReadOnly).(bool)
	return v
}

// WithUserID кладёт id аутентифицированного пользователя в контекст (симметрично UserID).
func WithUserID(ctx context.Context, uid uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxUserID, uid)
}

// UserID достаёт id пользователя из контекста.
func UserID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxUserID).(uuid.UUID)
	return v, ok
}

func token(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxToken).(string)
	return v, ok
}

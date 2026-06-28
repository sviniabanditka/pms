package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const cookieName = "pms_session"

type ctxKey int

const (
	ctxUserID ctxKey = iota
	ctxToken
)

// Authenticator извлекает сессию (cookie | Bearer | ?token=) и кладёт user_id в контекст.
// Не отклоняет анонимов — это делает RequireAuth (для гибкости публичных роутов).
// Bearer/query нужны не-браузерным клиентам (WS из CLI, мобильные).
func (s *Service) Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tok := extractToken(r)
		if tok != "" {
			if sess, err := s.sessions.Get(r.Context(), tok); err == nil && sess != nil {
				ctx := context.WithValue(r.Context(), ctxUserID, sess.UserID)
				ctx = context.WithValue(ctx, ctxToken, tok)
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

// UserID достаёт id пользователя из контекста.
func UserID(ctx context.Context) (uuid.UUID, bool) {
	v, ok := ctx.Value(ctxUserID).(uuid.UUID)
	return v, ok
}

func token(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ctxToken).(string)
	return v, ok
}

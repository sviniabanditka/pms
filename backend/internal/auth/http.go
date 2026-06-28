package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// Routes монтирует /auth/* и защищённый /me.
func (s *Service) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", s.handleRegister)
	r.Post("/login", s.handleLogin)
	r.Post("/logout", s.handleLogout)
	r.With(RequireAuth).Get("/me", s.handleMe)
	return r
}

type credsReq struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (s *Service) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req credsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	u, err := s.Register(r.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrWeakInput):
			writeErr(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrEmailTaken):
			writeErr(w, http.StatusConflict, err.Error())
		default:
			writeErr(w, http.StatusInternalServerError, "register failed")
		}
		return
	}
	// Авто-логин после регистрации.
	tok, _, err := s.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "login failed")
		return
	}
	s.setCookie(w, tok)
	// token в теле — для cross-origin SPA (Bearer); cookie остаётся для same-origin.
	writeJSON(w, http.StatusCreated, map[string]any{"id": u.ID, "email": u.Email, "name": u.Name, "token": tok})
}

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req credsReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "bad request")
		return
	}
	tok, u, err := s.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrRateLimited):
			writeErr(w, http.StatusTooManyRequests, err.Error())
		default:
			writeErr(w, http.StatusUnauthorized, "invalid credentials")
		}
		return
	}
	s.setCookie(w, tok)
	writeJSON(w, http.StatusOK, map[string]any{"id": u.ID, "email": u.Email, "name": u.Name, "token": tok})
}

func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) {
	if tok, ok := token(r.Context()); ok {
		_ = s.sessions.Delete(r.Context(), tok)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleMe(w http.ResponseWriter, r *http.Request) {
	uid, _ := UserID(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"user_id": uid})
}

func (s *Service) setCookie(w http.ResponseWriter, tok string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode, // ponytail: CSRF-токен для cross-site — отдельная веха; SameSite покрывает базовый случай
		Secure:   false,                // dev; true за TLS в проде
		Expires:  time.Now().Add(s.sessions.ttl),
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pms/backend/internal/models"
	"github.com/uptrace/bun"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrWeakInput          = errors.New("email and password (min 8) required")
	ErrRateLimited        = errors.New("too many attempts")
)

type Service struct {
	db       *bun.DB
	sessions *SessionStore
}

func NewService(db *bun.DB, sessions *SessionStore) *Service {
	return &Service{db: db, sessions: sessions}
}

func (s *Service) Register(ctx context.Context, email, name, password string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 8 {
		return nil, ErrWeakInput
	}
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	u := &models.User{
		ID: id, Email: email, Name: name, PasswordHash: &hash,
		Preferences: []byte("{}"), CreatedAt: now, UpdatedAt: now,
	}
	if _, err := s.db.NewInsert().Model(u).Exec(ctx); err != nil {
		if strings.Contains(err.Error(), "users_email_key") {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	ok, err := s.sessions.CheckLoginRate(ctx, email, 10, time.Minute)
	if err != nil {
		return "", nil, err
	}
	if !ok {
		return "", nil, ErrRateLimited
	}

	u := new(models.User)
	err = s.db.NewSelect().Model(u).Where("email = ?", email).Where("deleted_at IS NULL").Scan(ctx)
	if err != nil || u.PasswordHash == nil {
		return "", nil, ErrInvalidCredentials
	}
	valid, err := VerifyPassword(password, *u.PasswordHash)
	if err != nil || !valid {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.sessions.Create(ctx, u.ID)
	if err != nil {
		return "", nil, err
	}
	return token, u, nil
}

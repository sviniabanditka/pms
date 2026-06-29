package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Session — состояние сессии в Redis (opaque-токен, ADR-0006).
type Session struct {
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSessionStore(rdb *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{rdb: rdb, ttl: ttl}
}

func sessionKey(token string) string { return "session:" + token }
func userSetKey(uid uuid.UUID) string { return "user_sessions:" + uid.String() }

// Create генерит непрозрачный токен и кладёт сессию в Redis (+ индекс по пользователю для отзыва).
func (s *SessionStore) Create(ctx context.Context, uid uuid.UUID) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	val, _ := json.Marshal(Session{UserID: uid, CreatedAt: time.Now().UTC()})
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, sessionKey(token), val, s.ttl)
	pipe.SAdd(ctx, userSetKey(uid), token)
	pipe.Expire(ctx, userSetKey(uid), s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", err
	}
	return token, nil
}

// Get возвращает сессию по токену; скользящее продление TTL.
func (s *SessionStore) Get(ctx context.Context, token string) (*Session, error) {
	raw, err := s.rdb.Get(ctx, sessionKey(token)).Bytes()
	if err != nil {
		return nil, err // redis.Nil = нет сессии
	}
	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, err
	}
	s.rdb.Expire(ctx, sessionKey(token), s.ttl) // sliding
	return &sess, nil
}

// Delete удаляет одну сессию (logout).
func (s *SessionStore) Delete(ctx context.Context, token string) error {
	sess, err := s.Get(ctx, token)
	if err == nil && sess != nil {
		s.rdb.SRem(ctx, userSetKey(sess.UserID), token)
	}
	return s.rdb.Del(ctx, sessionKey(token)).Err()
}

// DeleteAllForUser отзывает все сессии пользователя (бан/смена пароля).
func (s *SessionStore) DeleteAllForUser(ctx context.Context, uid uuid.UUID) error {
	tokens, err := s.rdb.SMembers(ctx, userSetKey(uid)).Result()
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(tokens)+1)
	for _, t := range tokens {
		keys = append(keys, sessionKey(t))
	}
	keys = append(keys, userSetKey(uid))
	return s.rdb.Del(ctx, keys...).Err()
}

// CheckLoginRate простой лимит попыток логина по ключу (email/ip): N за окно.
func (s *SessionStore) CheckLoginRate(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	k := fmt.Sprintf("loginrate:%s", key)
	n, err := s.rdb.Incr(ctx, k).Result()
	if err != nil {
		return false, err
	}
	// ExpireNX выставляет TTL только если его ещё нет → атомарно и самовосстанавливается,
	// если процесс упал между Incr и Expire (иначе ключ завис бы навсегда).
	s.rdb.ExpireNX(ctx, k, window)
	return n <= int64(limit), nil
}

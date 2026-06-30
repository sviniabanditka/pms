// Package token — генерация/хеширование секретных токенов (API-токены, и т.п.).
// Храним только sha256-хеш; raw показываем владельцу один раз.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// Generate возвращает raw-токен (prefix + 48 hex) и его sha256-хеш.
func Generate(prefix string) (raw, hash string) {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	raw = prefix + hex.EncodeToString(b)
	return raw, Hash(raw)
}

// Hash — sha256-hex от raw-токена.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

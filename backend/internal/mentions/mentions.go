// Package mentions парсит токены упоминаний вида @[Name](uuid) из markdown-тела.
package mentions

import (
	"regexp"

	"github.com/google/uuid"
)

var re = regexp.MustCompile(`@\[[^\]]+\]\(([0-9a-fA-F-]{36})\)`)

// Parse возвращает уникальные userId из всех @[..](id) в тексте.
func Parse(body string) []uuid.UUID {
	seen := map[uuid.UUID]bool{}
	out := []uuid.UUID{}
	for _, m := range re.FindAllStringSubmatch(body, -1) {
		id, err := uuid.Parse(m[1])
		if err != nil || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}
